package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/papercomputeco/sweeper/pkg/linter"
	"github.com/papercomputeco/sweeper/pkg/planner"
	"github.com/papercomputeco/sweeper/pkg/tapes"
	"github.com/papercomputeco/sweeper/pkg/telemetry"
	"github.com/papercomputeco/sweeper/pkg/worker"
)

type LinterFunc func(ctx context.Context, dir string) (linter.ParseResult, error)

type Summary struct {
	TotalIssues int
	Tasks       int
	Fixed       int
	Failed      int
}

type Agent struct {
	cfg      config.Config
	linterFn LinterFunc
	executor worker.Executor
	pub      *telemetry.Publisher
}

type Option func(*Agent)

func WithLinterFunc(fn LinterFunc) Option {
	return func(a *Agent) { a.linterFn = fn }
}

func WithExecutor(exec worker.Executor) Option {
	return func(a *Agent) { a.executor = exec }
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
	defer a.pub.Close()

	if !a.cfg.NoTapes {
		status := tapes.CheckInstallation(tapes.FindDB(a.cfg.TargetDir))
		if status.Available {
			fmt.Printf("Tapes: using %s\n", status.DBPath)
		} else if status.Message != "" {
			fmt.Printf("Warning: %s\n", status.Message)
		}
	}

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

	fixTasks := planner.GroupByFile(issues)
	tasks := make([]worker.Task, len(fixTasks))
	for i, ft := range fixTasks {
		tasks[i] = worker.Task{
			ID:     i,
			File:   ft.File,
			Dir:    a.cfg.TargetDir,
			Issues: ft.Issues,
		}
		tasks[i].Prompt = worker.BuildPrompt(tasks[i])
	}
	fmt.Printf("Created %d fix tasks.\n", len(tasks))

	if a.cfg.DryRun {
		fmt.Println("Dry run - would fix:")
		for _, t := range tasks {
			fmt.Printf("  - %s (%d issues)\n", t.File, len(t.Issues))
		}
		return Summary{TotalIssues: len(issues), Tasks: len(tasks)}, nil
	}

	pool := worker.NewPool(a.cfg.Concurrency, a.executor)
	results := pool.Run(ctx, tasks)

	summary := Summary{TotalIssues: len(issues), Tasks: len(tasks)}
	for _, r := range results {
		if r.Success {
			summary.Fixed += r.IssuesFix
		} else {
			summary.Failed++
		}
		a.pub.Publish(telemetry.Event{
			Timestamp: time.Now(),
			Type:      "fix_attempt",
			Data: map[string]any{
				"file":     r.File,
				"success":  r.Success,
				"duration": r.Duration.String(),
				"issues":   r.IssuesFix,
				"error":    r.Error,
				"linter":   linterName,
			},
		})
	}

	fmt.Printf("Results: %d fixed, %d failed out of %d tasks.\n", summary.Fixed, summary.Failed, summary.Tasks)
	return summary, nil
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

	summary := Summary{TotalIssues: 1, Tasks: 1}
	for _, r := range results {
		if r.Success {
			summary.Fixed++
		} else {
			summary.Failed++
		}
		a.pub.Publish(telemetry.Event{
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
