package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/papercomputeco/sweeper/pkg/linter"
	"github.com/papercomputeco/sweeper/pkg/loop"
	"github.com/papercomputeco/sweeper/pkg/planner"
	"github.com/papercomputeco/sweeper/pkg/session"
	"github.com/papercomputeco/sweeper/pkg/tapes"
	"github.com/papercomputeco/sweeper/pkg/telemetry"
	"github.com/papercomputeco/sweeper/pkg/worker"
)

type LinterFunc func(ctx context.Context, dir string) (linter.ParseResult, error)

// VMManager is the interface for VM lifecycle management.
type VMManager interface {
	Shutdown() error
}

type Summary struct {
	TotalIssues int
	Tasks       int
	Fixed       int
	Failed      int
	Rounds      int
}

type Agent struct {
	cfg         config.Config
	linterFn    LinterFunc
	executor    worker.Executor
	pub         *telemetry.Publisher
	vm          VMManager
	sessionPath string
}

type Option func(*Agent)

func WithLinterFunc(fn LinterFunc) Option {
	return func(a *Agent) { a.linterFn = fn }
}

func WithExecutor(exec worker.Executor) Option {
	return func(a *Agent) { a.executor = exec }
}

func WithVM(vm VMManager) Option {
	return func(a *Agent) { a.vm = vm }
}

func defaultLinterFunc(ctx context.Context, dir string) (linter.ParseResult, error) {
	return linter.Run(ctx, dir)
}

func New(cfg config.Config, opts ...Option) *Agent {
	a := &Agent{
		cfg:      cfg,
		linterFn: defaultLinterFunc,
		executor: worker.ClaudeExecutor,
		pub:      telemetry.NewPublisher(cfg.TelemetryDir),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Agent) Run(ctx context.Context) (Summary, error) {
	defer func() { _ = a.pub.Close() }()
	if a.vm != nil {
		defer func() { _ = a.vm.Shutdown() }()
	}

	if !a.cfg.NoTapes {
		status := tapes.CheckInstallation(tapes.FindDB(a.cfg.TargetDir))
		if status.Available {
			fmt.Printf("Tapes: using %s\n", status.DBPath)
		} else if status.Message != "" {
			fmt.Printf("Warning: %s\n", status.Message)
		}
	}

	lintCmd := "golangci-lint run ./..."
	if len(a.cfg.LintCommand) > 0 {
		lintCmd = strings.Join(a.cfg.LintCommand, " ")
	}
	sessionCfg := session.Config{
		Objective:   "Fix lint issues",
		LintCommand: lintCmd,
		TargetDir:   a.cfg.TargetDir,
		MaxRounds:   a.cfg.MaxRounds,
	}
	sp, err := session.Generate(filepath.Join(a.cfg.TargetDir, ".sweeper"), sessionCfg)
	if err != nil {
		fmt.Printf("Warning: session doc: %v\n", err)
	} else {
		a.sessionPath = sp
		fmt.Printf("Session: %s\n", sp)
	}

	_ = a.pub.Publish(telemetry.Event{
		Timestamp: time.Now(),
		Type:      "init",
		Data: map[string]any{
			"name":           fmt.Sprintf("sweep-%s", time.Now().Format("2006-01-02")),
			"linterCommand":  sessionCfg.LintCommand,
			"targetDir":      a.cfg.TargetDir,
			"maxRounds":      a.cfg.MaxRounds,
			"staleThreshold": a.cfg.StaleThreshold,
		},
	})

	fmt.Println("Running linter...")
	result, err := a.linterFn(ctx, a.cfg.TargetDir)
	if err != nil {
		return Summary{}, fmt.Errorf("linting: %w", err)
	}

	linterName := a.cfg.LinterName
	if linterName == "" {
		linterName = "golangci-lint"
	}

	if result.Parsed {
		return a.runParsed(ctx, result, linterName)
	}
	if result.RawOutput != "" {
		return a.runRaw(ctx, result, linterName)
	}
	fmt.Println("No lint issues found.")
	return Summary{}, nil
}

func (a *Agent) runParsed(ctx context.Context, result linter.ParseResult, linterName string) (Summary, error) {
	issues := result.Issues
	fmt.Printf("Found %d lint issues across files.\n", len(issues))

	maxRounds := a.cfg.MaxRounds
	if maxRounds < 1 {
		maxRounds = 1
	}

	summary := Summary{TotalIssues: len(issues)}
	fileHistories := make(map[string]*loop.FileHistory)
	explorationAttempted := make(map[string]bool)

	for round := 0; round < maxRounds; round++ {
		if len(issues) == 0 {
			break
		}

		fixTasks := planner.GroupByFile(issues)
		tasks := make([]worker.Task, len(fixTasks))
		strategies := make([]loop.Strategy, len(fixTasks))

		for i, ft := range fixTasks {
			fh := safeHistory(fileHistories[ft.File])
			strategy := loop.PickStrategy(round, fh, a.cfg.StaleThreshold)
			strategies[i] = strategy

			tasks[i] = worker.Task{
				ID:     i,
				File:   ft.File,
				Dir:    a.cfg.TargetDir,
				Issues: ft.Issues,
			}
			switch strategy {
			case loop.StrategyRetry:
				tasks[i].Prompt = worker.BuildRetryPrompt(tasks[i], fh.LastOutput())
			case loop.StrategyExploration:
				tasks[i].Prompt = worker.BuildExplorationPrompt(tasks[i], fh.LastOutput())
				explorationAttempted[ft.File] = true
			default:
				tasks[i].Prompt = worker.BuildPrompt(tasks[i])
			}
		}

		if round == 0 {
			summary.Tasks = len(tasks)
			fmt.Printf("Created %d fix tasks.\n", len(tasks))
		} else {
			fmt.Printf("Round %d: %d files with remaining issues.\n", round+1, len(tasks))
		}

		if a.cfg.DryRun {
			fmt.Println("Dry run - would fix:")
			for _, t := range tasks {
				fmt.Printf("  - %s (%d issues)\n", t.File, len(t.Issues))
			}
			summary.Rounds = round + 1
			return summary, nil
		}

		results := a.runRound(ctx, tasks)
		summary.Rounds = round + 1

		for i, r := range results {
			strategy := strategies[i]
			a.publishFixAttempt(r, linterName, round, strategy)

			// Update file history
			fh, ok := fileHistories[r.File]
			if !ok {
				fh = &loop.FileHistory{File: r.File}
				fileHistories[r.File] = fh
			}
			fh.Rounds = append(fh.Rounds, loop.RoundResult{
				File:         r.File,
				Round:        round,
				Strategy:     strategy,
				IssuesBefore: len(tasks[i].Issues),
				Output:       r.Output,
				Success:      r.Success,
				Error:        r.Error,
			})
		}

		a.publishRoundComplete(round, linterName, len(tasks), results)

		// If last round, tally results and stop
		if round >= maxRounds-1 {
			for _, r := range results {
				if r.Success {
					summary.Fixed += r.IssuesFix
				} else {
					summary.Failed++
				}
			}
			break
		}

		// Re-lint to check remaining issues
		reResult, err := a.linterFn(ctx, a.cfg.TargetDir)
		if err != nil {
			// Don't fail the whole run; tally current results and stop
			for _, r := range results {
				if r.Success {
					summary.Fixed += r.IssuesFix
				} else {
					summary.Failed++
				}
			}
			break
		}

		// Count fixes from this round based on re-lint results
		remainingByFile := make(map[string]int)
		if reResult.Parsed {
			for _, iss := range reResult.Issues {
				remainingByFile[iss.File]++
			}
		}
		for i, r := range results {
			before := len(tasks[i].Issues)
			after := remainingByFile[r.File]
			fixed := before - after
			if fixed < 0 {
				fixed = 0
			}
			summary.Fixed += fixed

			// Update round result with actual fix counts
			fh := fileHistories[r.File]
			last := &fh.Rounds[len(fh.Rounds)-1]
			last.IssuesAfter = after
			last.Fixed = fixed
		}

		if !reResult.Parsed || len(reResult.Issues) == 0 {
			fmt.Println("All issues resolved!")
			if a.sessionPath != "" {
				_ = session.UpdateStatus(a.sessionPath, round+1, len(reResult.Issues), summary.Fixed, 0)
			}
			break
		}

		// Filter to retryable issues
		issues = filterRetryableIssues(reResult.Issues, fileHistories, explorationAttempted, a.cfg.StaleThreshold)

		if a.sessionPath != "" {
			_ = session.UpdateStatus(a.sessionPath, round+1, len(reResult.Issues), summary.Fixed, len(issues))
		}
	}

	fmt.Printf("Results: %d fixed, %d failed (%d rounds).\n", summary.Fixed, summary.Failed, summary.Rounds)
	return summary, nil
}

func (a *Agent) runRound(ctx context.Context, tasks []worker.Task) []worker.Result {
	pool := worker.NewPool(a.cfg.Concurrency, a.executor)
	return pool.Run(ctx, tasks)
}

func (a *Agent) publishFixAttempt(r worker.Result, linterName string, round int, strategy loop.Strategy) {
	_ = a.pub.Publish(telemetry.Event{
		Timestamp: time.Now(),
		Type:      "fix_attempt",
		Data: map[string]any{
			"file":     r.File,
			"success":  r.Success,
			"duration": r.Duration.String(),
			"issues":   r.IssuesFix,
			"error":    r.Error,
			"linter":   linterName,
			"round":    round + 1,
			"strategy": strategy.String(),
		},
	})
}

func (a *Agent) publishRoundComplete(round int, linterName string, taskCount int, results []worker.Result) {
	fixed := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			fixed += r.IssuesFix
		} else {
			failed++
		}
	}
	_ = a.pub.Publish(telemetry.Event{
		Timestamp: time.Now(),
		Type:      "round_complete",
		Data: map[string]any{
			"round":  round + 1,
			"linter": linterName,
			"tasks":  taskCount,
			"fixed":  fixed,
			"failed": failed,
		},
	})
}

func (a *Agent) runRaw(ctx context.Context, result linter.ParseResult, linterName string) (Summary, error) {
	fmt.Println("Could not parse structured issues; passing raw output to agent.")

	task := worker.Task{
		ID:        0,
		Dir:       a.cfg.TargetDir,
		RawOutput: result.RawOutput,
	}
	task.Prompt = worker.BuildRawPrompt(task)

	if a.cfg.DryRun {
		fmt.Println("Dry run - would send raw lint output to agent for analysis.")
		return Summary{TotalIssues: 1, Tasks: 1}, nil
	}

	pool := worker.NewPool(a.cfg.Concurrency, a.executor)
	results := pool.Run(ctx, []worker.Task{task})

	summary := Summary{TotalIssues: 1, Tasks: 1, Rounds: 1}
	for _, r := range results {
		if r.Success {
			summary.Fixed++
		} else {
			summary.Failed++
		}
		_ = a.pub.Publish(telemetry.Event{
			Timestamp: time.Now(),
			Type:      "fix_attempt",
			Data: map[string]any{
				"file":     "raw",
				"success":  r.Success,
				"duration": r.Duration.String(),
				"issues":   1,
				"error":    r.Error,
				"linter":   linterName,
			},
		})
	}

	fmt.Printf("Results: %d fixed, %d failed out of %d tasks.\n", summary.Fixed, summary.Failed, summary.Tasks)
	return summary, nil
}

func safeHistory(fh *loop.FileHistory) loop.FileHistory {
	if fh == nil {
		return loop.FileHistory{}
	}
	return *fh
}

// filterRetryableIssues removes issues for files that have exhausted all strategies.
// A file is removed if exploration was attempted and it's still stagnant.
func filterRetryableIssues(
	issues []linter.Issue,
	histories map[string]*loop.FileHistory,
	explorationAttempted map[string]bool,
	staleThreshold int,
) []linter.Issue {
	var retryable []linter.Issue
	for _, iss := range issues {
		if explorationAttempted[iss.File] && loop.DetectStagnation(safeHistory(histories[iss.File]), staleThreshold) {
			continue
		}
		retryable = append(retryable, iss)
	}
	return retryable
}
