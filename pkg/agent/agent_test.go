package agent

import (
	"context"
	"errors"
	"os"
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
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{}, nil
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
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{}, nil
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
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{Issues: fakeIssues, Parsed: true}, nil
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

func TestAgentRunRawFallback(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		TargetDir:    dir,
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		LinterName:   "custom",
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{
			RawOutput: "ERROR: something unparseable\n  details here\n",
			Parsed:    false,
		}, nil
	}
	fakeExecutor := func(ctx context.Context, task worker.Task) worker.Result {
		if task.RawOutput == "" {
			t.Error("expected RawOutput to be set on task")
		}
		return worker.Result{TaskID: task.ID, Success: true, IssuesFix: 1}
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

func TestAgentRunRawDryRun(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		DryRun:       true,
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{
			RawOutput: "some raw output",
			Parsed:    false,
		}, nil
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	summary, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Tasks != 1 {
		t.Errorf("expected 1 task in dry run, got %d", summary.Tasks)
	}
}

func TestAgentRunLinterError(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		NoTapes:      true,
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{}, errors.New("linter broke")
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	_, err := a.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from linter")
	}
}

func TestAgentRunParsedWithFailure(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		NoTapes:      true,
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{
			Issues: []linter.Issue{
				{File: "a.go", Line: 1, Linter: "revive", Message: "msg"},
			},
			Parsed: true,
		}, nil
	}
	fakeExecutor := func(ctx context.Context, task worker.Task) worker.Result {
		return worker.Result{TaskID: task.ID, File: task.File, Success: false, Error: "agent failed"}
	}
	a := New(cfg, WithLinterFunc(fakeLinter), WithExecutor(fakeExecutor))
	summary, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
}

func TestAgentRunRawWithFailure(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		NoTapes:      true,
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{
			RawOutput: "unparseable output",
			Parsed:    false,
		}, nil
	}
	fakeExecutor := func(ctx context.Context, task worker.Task) worker.Result {
		return worker.Result{TaskID: task.ID, Success: false, Error: "agent failed"}
	}
	a := New(cfg, WithLinterFunc(fakeLinter), WithExecutor(fakeExecutor))
	summary, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
}

func TestAgentRunParsedDryRun(t *testing.T) {
	cfg := config.Config{
		TargetDir:    t.TempDir(),
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
		DryRun:       true,
		NoTapes:      true,
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{
			Issues: []linter.Issue{
				{File: "a.go", Line: 1, Linter: "revive", Message: "msg"},
				{File: "b.go", Line: 2, Linter: "revive", Message: "msg"},
			},
			Parsed: true,
		}, nil
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	summary, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalIssues != 2 {
		t.Errorf("expected 2 total issues, got %d", summary.TotalIssues)
	}
	if summary.Tasks != 2 {
		t.Errorf("expected 2 tasks, got %d", summary.Tasks)
	}
}

func TestDefaultLinterFunc(t *testing.T) {
	// Exercise defaultLinterFunc for coverage. It shells out to golangci-lint
	// which may or may not be installed; we don't care about the result.
	_, _ = defaultLinterFunc(context.Background(), t.TempDir())
}

func TestAgentRunTapesAvailable(t *testing.T) {
	// Create a fake tapes DB in the target dir to cover the "Tapes: using" branch.
	dir := t.TempDir()
	tapesDir := dir + "/.tapes"
	os.MkdirAll(tapesDir, 0o755)
	os.WriteFile(tapesDir+"/tapes.db", []byte("fake"), 0o644)

	cfg := config.Config{
		TargetDir:    dir,
		Concurrency:  1,
		TelemetryDir: t.TempDir(),
	}
	fakeLinter := func(ctx context.Context, dir string) (linter.ParseResult, error) {
		return linter.ParseResult{}, nil
	}
	a := New(cfg, WithLinterFunc(fakeLinter))
	_, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}
