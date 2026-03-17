package linter

import (
	"context"
	"os"
	"testing"
)

func TestParseOutput(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample_lint_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	result := ParseOutput(string(data))
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result.Issues))
	}
	first := result.Issues[0]
	if first.File != "server.go" {
		t.Errorf("expected file server.go, got %s", first.File)
	}
	if first.Line != 42 {
		t.Errorf("expected line 42, got %d", first.Line)
	}
	if first.Linter != "ineffassign" {
		t.Errorf("expected linter ineffassign, got %s", first.Linter)
	}
}

func TestParseOutputEmpty(t *testing.T) {
	result := ParseOutput("")
	if result.Parsed {
		t.Error("expected Parsed to be false for empty input")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(result.Issues))
	}
}

func TestParseOutputGolangciFormat(t *testing.T) {
	raw := `server.go:42:2: err is not used (ineffassign)
handler.go:10:5: comment missing (revive)`
	result := ParseOutput(raw)
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].Linter != "ineffassign" {
		t.Errorf("expected linter ineffassign, got %s", result.Issues[0].Linter)
	}
	if result.Issues[1].Linter != "revive" {
		t.Errorf("expected linter revive, got %s", result.Issues[1].Linter)
	}
}

func TestParseOutputGenericFormat(t *testing.T) {
	raw := `src/App.tsx:12:5: 'useState' is defined but never used
src/utils.ts:22:3: 'console' is not allowed`
	result := ParseOutput(raw)
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].File != "src/App.tsx" {
		t.Errorf("expected file src/App.tsx, got %s", result.Issues[0].File)
	}
	if result.Issues[0].Col != 5 {
		t.Errorf("expected col 5, got %d", result.Issues[0].Col)
	}
	if result.Issues[0].Linter != "custom" {
		t.Errorf("expected linter custom, got %s", result.Issues[0].Linter)
	}
}

func TestParseOutputMinimalFormat(t *testing.T) {
	raw := `main.py:15: Missing module docstring
views.py:8: Unable to import 'django'`
	result := ParseOutput(raw)
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].File != "main.py" {
		t.Errorf("expected file main.py, got %s", result.Issues[0].File)
	}
	if result.Issues[0].Line != 15 {
		t.Errorf("expected line 15, got %d", result.Issues[0].Line)
	}
	if result.Issues[0].Col != 0 {
		t.Errorf("expected col 0, got %d", result.Issues[0].Col)
	}
}

func TestParseOutputESLint(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample_eslint_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	result := ParseOutput(string(data))
	if !result.Parsed {
		t.Fatal("expected Parsed to be true for ESLint output")
	}
	if len(result.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result.Issues))
	}
}

func TestParseOutputESLintStylish(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample_eslint_stylish_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	result := ParseOutput(string(data))
	if !result.Parsed {
		t.Fatal("expected Parsed to be true for ESLint stylish output")
	}
	if len(result.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result.Issues))
	}

	// Verify first issue
	first := result.Issues[0]
	if first.File != "src/App.tsx" {
		t.Errorf("expected file src/App.tsx, got %s", first.File)
	}
	if first.Line != 2 {
		t.Errorf("expected line 2, got %d", first.Line)
	}
	if first.Col != 10 {
		t.Errorf("expected col 10, got %d", first.Col)
	}
	if first.Linter != "@typescript-eslint/no-unused-vars" {
		t.Errorf("expected linter @typescript-eslint/no-unused-vars, got %s", first.Linter)
	}

	// Verify issue from a different file
	third := result.Issues[2]
	if third.File != "src/components/Header.tsx" {
		t.Errorf("expected file src/components/Header.tsx, got %s", third.File)
	}
	if third.Linter != "@typescript-eslint/explicit-function-return-type" {
		t.Errorf("expected linter @typescript-eslint/explicit-function-return-type, got %s", third.Linter)
	}

	// Verify last issue uses simple rule name
	last := result.Issues[3]
	if last.Linter != "no-console" {
		t.Errorf("expected linter no-console, got %s", last.Linter)
	}
}

func TestParseOutputESLintStylishOrphanLine(t *testing.T) {
	// Issue line appearing before any file header should be skipped
	raw := "  1:5  error  Unexpected var  no-var\n\nsrc/app.ts\n  3:1  warning  Missing semicolon  semi\n"
	result := ParseOutput(raw)
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue (orphan skipped), got %d", len(result.Issues))
	}
	if result.Issues[0].File != "src/app.ts" {
		t.Errorf("expected file src/app.ts, got %s", result.Issues[0].File)
	}
}

func TestParseOutputPylint(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample_pylint_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	result := ParseOutput(string(data))
	if !result.Parsed {
		t.Fatal("expected Parsed to be true for pylint output")
	}
	if len(result.Issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(result.Issues))
	}
}

func TestParseOutputRawFallback(t *testing.T) {
	data, err := os.ReadFile("../../testdata/sample_raw_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	result := ParseOutput(string(data))
	if result.Parsed {
		t.Error("expected Parsed to be false for raw output")
	}
	if result.RawOutput != string(data) {
		t.Error("expected RawOutput to contain the full input")
	}
}

func TestParseOutputMixedFormats(t *testing.T) {
	raw := `server.go:42:2: err is not used (ineffassign)
some non-matching line
handler.go:10:5: comment missing`
	result := ParseOutput(raw)
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
	if result.Issues[0].Linter != "ineffassign" {
		t.Errorf("expected first issue to use golangci pattern, got linter %s", result.Issues[0].Linter)
	}
	if result.Issues[1].Linter != "custom" {
		t.Errorf("expected second issue to use generic pattern, got linter %s", result.Issues[1].Linter)
	}
}

func TestParseOutputPreservesRaw(t *testing.T) {
	raw := "server.go:42:2: err is not used (ineffassign)\n"
	result := ParseOutput(raw)
	if result.RawOutput != raw {
		t.Error("expected RawOutput to always be set")
	}
}

func TestRunCommandSuccess(t *testing.T) {
	result, err := RunCommand(context.Background(), ".", []string{"echo", "fake.go:1:1: test error (testlint)"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Parsed {
		t.Fatal("expected Parsed to be true")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].File != "fake.go" {
		t.Errorf("expected file fake.go, got %s", result.Issues[0].File)
	}
}

func TestRunCommandExitError(t *testing.T) {
	// bash -c exits non-zero; the ExitError path is taken and output is still parsed.
	result, err := RunCommand(context.Background(), ".", []string{"bash", "-c", "echo 'a.go:1:1: bad (lint)' && exit 1"})
	if err != nil {
		t.Fatalf("unexpected error for ExitError: %v", err)
	}
	if !result.Parsed {
		t.Fatal("expected Parsed to be true even with non-zero exit")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestRunCommandNotFound(t *testing.T) {
	_, err := RunCommand(context.Background(), ".", []string{"nonexistent-command-sweeper-test"})
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestRun(t *testing.T) {
	// Run tries golangci-lint. Whether installed or not, the function itself is exercised.
	_, _ = Run(context.Background(), t.TempDir())
}

func TestParseMinimalSkipsCompilerHeaders(t *testing.T) {
	// golangci-lint typecheck errors produce lines like:
	// ../../../tmp/project/main.go:1: : # testproject
	raw := `../../../tmp/project/main.go:1: : # testproject
./main.go:5:2: "os" imported and not used
./main.go:9:2: declared and not used: x`
	result := ParseOutput(raw)
	// Should parse the 2 real issues but skip the compiler header.
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
		for _, iss := range result.Issues {
			t.Logf("  %s:%d: %s", iss.File, iss.Line, iss.Message)
		}
	}
}

func TestNormalizeIssuePaths(t *testing.T) {
	dir := t.TempDir()
	issues := []Issue{
		{File: "./main.go", Line: 1, Message: "err"},
		{File: "pkg/foo.go", Line: 2, Message: "err"},
	}
	normalizeIssuePaths(issues, dir)
	if issues[0].File != "main.go" {
		t.Errorf("expected normalized main.go, got %s", issues[0].File)
	}
	if issues[1].File != "pkg/foo.go" {
		t.Errorf("expected pkg/foo.go unchanged, got %s", issues[1].File)
	}
}
