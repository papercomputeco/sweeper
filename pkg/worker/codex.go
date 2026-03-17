package worker

import (
	"context"
	"os/exec"
	"time"
)

// NewCodexExecutor returns an Executor that invokes the codex CLI.
// Codex uses --quiet for minimal output and --approval-mode full-auto
// so it applies fixes without interactive approval.
func NewCodexExecutor() Executor {
	return func(ctx context.Context, task Task) Result {
		start := time.Now()
		prompt := task.Prompt
		if prompt == "" {
			prompt = BuildPrompt(task)
		}
		cmd := exec.CommandContext(ctx, "codex",
			"--quiet",
			"--approval-mode", "full-auto",
			prompt,
		)
		cmd.Dir = task.Dir
		out, err := cmd.CombinedOutput()
		duration := time.Since(start)
		if err != nil {
			return Result{
				TaskID: task.ID, File: task.File, Success: false,
				Output: string(out), Error: err.Error(), Duration: duration,
				Provider: "codex",
			}
		}
		return Result{
			TaskID: task.ID, File: task.File, Success: true,
			Output: string(out), Duration: duration, IssuesFix: len(task.Issues),
			Provider: "codex",
		}
	}
}
