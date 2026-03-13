package linter

import (
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
