package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

func TestPoolRunsTasksConcurrently(t *testing.T) {
	tasks := []Task{
		{ID: 1, File: "a.go", Issues: []linter.Issue{{File: "a.go", Line: 1, Linter: "revive", Message: "msg"}}},
		{ID: 2, File: "b.go", Issues: []linter.Issue{{File: "b.go", Line: 2, Linter: "revive", Message: "msg"}}},
	}
	executor := func(ctx context.Context, t Task) Result {
		return Result{TaskID: t.ID, File: t.File, Success: true, IssuesFix: len(t.Issues)}
	}
	pool := NewPool(2, executor)
	results := pool.Run(context.Background(), tasks)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("task %d failed", r.TaskID)
		}
	}
}

func TestPoolRespectsMaxConcurrency(t *testing.T) {
	var maxConcurrent int64
	var current int64
	var mu sync.Mutex
	tasks := make([]Task, 10)
	for i := range tasks {
		tasks[i] = Task{ID: i, File: fmt.Sprintf("%d.go", i)}
	}
	executor := func(ctx context.Context, task Task) Result {
		mu.Lock()
		current++
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		current--
		mu.Unlock()
		return Result{TaskID: task.ID, Success: true}
	}
	pool := NewPool(3, executor)
	pool.Run(context.Background(), tasks)
	if maxConcurrent > 3 {
		t.Errorf("max concurrency exceeded: got %d, want <= 3", maxConcurrent)
	}
}
