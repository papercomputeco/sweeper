package agent

import (
	"context"
	"testing"
	"github.com/papercomputeco/sweeper/pkg/config"
	"github.com/papercomputeco/sweeper/pkg/linter"
	"github.com/papercomputeco/sweeper/pkg/worker"
)

func TestAgentRunPrintsTapesWarning(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
	}
	fakeLinter := func(ctx context.Context, dir string) ([]linter.Issue, error) {
		return nil, nil
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	_, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentRunSkipsTapesWithFlag(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		NoTapes:      true,
	}
	fakeLinter := func(ctx context.Context, dir string) ([]linter.Issue, error) {
		return nil, nil
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	_, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentRunWithFakeExecutor(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		TargetDir:    dir,
		Concurrency:  2,
		TelemetryDir: t.TempDir(),
	}
	fakeIssues := []linter.Issue{
		{File: "a.go", Line: 1, Linter: "revive", Message: "comment missing"},
	}
	fakeLinter := func(ctx context.Context, dir string) ([]linter.Issue, error) {
		return fakeIssues, nil
	}
	fakeExecutor := func(ctx context.Context, task worker.Task) worker.Result {
		return worker.Result{TaskID: task.ID, File: task.File, Success: true, IssuesFix: 1}
	}
	a := New(cfg, WithLinterFunc(fakeLinter), WithExecutor(fakeExecutor))
	summary, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalIssues != 1 {
		t.Errorf("expected 1 total issue, got %d", summary.TotalIssues)
	}
	if summary.Fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", summary.Fixed)
	}
}
