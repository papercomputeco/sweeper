package worker

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// allowedTools is the narrow set of tools sweeper agents need for lint fixing.
// Using --allowedTools instead of --dangerously-skip-permissions gives each
// agent only the permissions it needs rather than a blanket safety bypass.
var allowedTools = []string{
	"Read",
	"Write",
	"Edit",
	"Glob",
	"Grep",
	"Bash(go build:*)",
	"Bash(go vet:*)",
}

func ClaudeExecutor(ctx context.Context, task Task) Result {
	start := time.Now()
	prompt := task.Prompt
	if prompt == "" {
		prompt = BuildPrompt(task)
	}
	cmd := exec.CommandContext(ctx, "claude",
		"--print",
		"--allowedTools", strings.Join(allowedTools, ","),
		prompt,
	)
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
