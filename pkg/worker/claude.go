package worker

import (
	"context"
	"os/exec"
	"time"
)

func ClaudeExecutor(ctx context.Context, task Task) Result {
	start := time.Now()
	prompt := BuildPrompt(task)
	cmd := exec.CommandContext(ctx, "claude", "--print", "--dangerously-skip-permissions", prompt)
	cmd.Dir = task.Dir
	out, err := cmd.CombinedOutput()
	duration := time.Since(start)
	if err != nil {
		return Result{
			TaskID: task.ID, File: task.File, Success: false,
			Output: string(out), Error: err.Error(), Duration: duration,
		}
	}
	return Result{
		TaskID: task.ID, File: task.File, Success: true,
		Output: string(out), Duration: duration, IssuesFix: len(task.Issues),
	}
}
