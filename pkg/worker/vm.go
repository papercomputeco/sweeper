package worker

import (
	"context"
	"time"
)

// VMExecer is the interface the VM executor needs.
type VMExecer interface {
	Exec(ctx context.Context, args ...string) ([]byte, error)
}

// NewVMExecutor returns an Executor that runs claude inside a stereOS VM.
func NewVMExecutor(vm VMExecer) Executor {
	return func(ctx context.Context, task Task) Result {
		start := time.Now()
		out, err := vm.Exec(ctx,
			"claude", "--print", "--dangerously-skip-permissions",
			task.Prompt,
		)
		duration := time.Since(start)
		if err != nil {
			return Result{
				TaskID:   task.ID,
				File:     task.File,
				Success:  false,
				Output:   string(out),
				Error:    err.Error(),
				Duration: duration,
			}
		}
		return Result{
			TaskID:    task.ID,
			File:      task.File,
			Success:   true,
			Output:    string(out),
			Duration:  duration,
			IssuesFix: len(task.Issues),
		}
	}
}
