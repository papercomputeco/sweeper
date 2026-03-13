package worker

import (
	"context"
	"fmt"
	"testing"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

type fakeVM struct {
	execFunc func(ctx context.Context, args ...string) ([]byte, error)
}

func (f *fakeVM) Exec(ctx context.Context, args ...string) ([]byte, error) {
	return f.execFunc(ctx, args...)
}

func TestNewVMExecutor(t *testing.T) {
	vm := &fakeVM{
		execFunc: func(ctx context.Context, args ...string) ([]byte, error) {
			return []byte("fixed"), nil
		},
	}
	exec := NewVMExecutor(vm)
	task := Task{
		ID:   0,
		File: "src/main.go",
		Dir:  "/host/project",
		Issues: []linter.Issue{
			{File: "src/main.go", Line: 10, Message: "unused var", Linter: "revive"},
		},
		Prompt: "Fix the lint issues",
	}
	result := exec(context.Background(), task)
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Output != "fixed" {
		t.Errorf("expected 'fixed', got %s", result.Output)
	}
	if result.IssuesFix != 1 {
		t.Errorf("expected 1 issue fix, got %d", result.IssuesFix)
	}
}

func TestNewVMExecutorError(t *testing.T) {
	vm := &fakeVM{
		execFunc: func(ctx context.Context, args ...string) ([]byte, error) {
			return []byte("error output"), fmt.Errorf("exit 1")
		},
	}
	exec := NewVMExecutor(vm)
	task := Task{
		ID:     0,
		File:   "main.go",
		Dir:    "/host/project",
		Prompt: "Fix it",
	}
	result := exec(context.Background(), task)
	if result.Success {
		t.Error("expected failure")
	}
	if result.Output != "error output" {
		t.Errorf("expected error output, got %s", result.Output)
	}
}
