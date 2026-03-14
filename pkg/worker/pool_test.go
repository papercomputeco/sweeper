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

func TestPoolRunStream(t *testing.T) {
	tasks := []Task{
		{ID: 1, File: "a.go", Issues: []linter.Issue{{File: "a.go", Line: 1, Linter: "revive", Message: "msg"}}},
		{ID: 2, File: "b.go", Issues: []linter.Issue{{File: "b.go", Line: 2, Linter: "revive", Message: "msg"}}},
		{ID: 3, File: "c.go", Issues: []linter.Issue{{File: "c.go", Line: 3, Linter: "revive", Message: "msg"}}},
	}
	executor := func(ctx context.Context, t Task) Result {
		time.Sleep(10 * time.Millisecond)
		return Result{TaskID: t.ID, File: t.File, Success: true, IssuesFix: len(t.Issues)}
	}
	pool := NewPool(2, executor)
	ch := pool.RunStream(context.Background(), tasks)

	var results []Result
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("task %d failed", r.TaskID)
		}
	}
}

func TestPoolRunStreamEmpty(t *testing.T) {
	executor := func(ctx context.Context, task Task) Result {
		t.Fatal("executor should not be called for empty tasks")
		return Result{}
	}
	pool := NewPool(2, executor)
	ch := pool.RunStream(context.Background(), nil)

	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 results from empty stream, got %d", count)
	}
}

func TestPoolRunEmpty(t *testing.T) {
	executor := func(ctx context.Context, task Task) Result {
		t.Fatal("executor should not be called for empty tasks")
		return Result{}
	}
	pool := NewPool(2, executor)
	results := pool.Run(context.Background(), nil)
	if results != nil {
		t.Errorf("expected nil results for empty tasks, got %v", results)
	}
}
