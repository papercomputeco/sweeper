package worker

import (
	"context"
	"os/exec"
	"time"
)

// NewClaudeExecutor returns an Executor that invokes claude with
// --dangerously-skip-permissions for non-interactive sub-agent use.
func NewClaudeExecutor() Executor {
	return func(ctx context.Context, task Task) Result {
		start := time.Now()
		prompt := task.Prompt
		if prompt == "" {
			prompt = BuildPrompt(task)
		}
		cmd := exec.CommandContext(ctx, "claude",
			"--print",
			"--dangerously-skip-permissions",
			prompt,
		)
		cmd.Dir = task.Dir
		out, err := cmd.CombinedOutput()
		duration := time.Since(start)
		if err != nil {
			return Result{
				TaskID: task.ID, File: task.File, Success: false,
				Output: string(out), Error: err.Error(), Duration: duration,
				Provider: "claude",
			}
		}
		return Result{
			TaskID: task.ID, File: task.File, Success: true,
			Output: string(out), Duration: duration, IssuesFix: len(task.Issues),
			Provider: "claude",
		}
	}
}
