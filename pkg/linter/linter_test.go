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
	issues := ParseOutput(string(data))
	if len(issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}
	first := issues[0]
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
	issues := ParseOutput("")
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}
