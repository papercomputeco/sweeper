package worker

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// NewClaudeExecutor returns an Executor that invokes claude with the given
// allowed tools. This lets callers configure which tools sub-agents can use
// without reverting to --dangerously-skip-permissions.
func NewClaudeExecutor(allowedTools []string) Executor {
	toolsArg := strings.Join(allowedTools, ",")
	return func(ctx context.Context, task Task) Result {
		start := time.Now()
		prompt := task.Prompt
		if prompt == "" {
			prompt = BuildPrompt(task)
		}
		cmd := exec.CommandContext(ctx, "claude",
			"--print",
			"--allowedTools", toolsArg,
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
