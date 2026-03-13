package worker

import (
	"strings"
	"testing"

	"github.com/papercomputeco/sweeper/pkg/linter"
)

func TestBuildRetryPrompt(t *testing.T) {
	task := Task{
		File:   "main.go",
		Issues: []linter.Issue{{File: "main.go", Line: 5, Message: "exported without comment", Linter: "golint"}},
	}
	got := BuildRetryPrompt(task, "added a comment but wrong format")
	if !strings.Contains(got, "main.go") {
		t.Error("expected file name")
	}
	if !strings.Contains(got, "Line 5") {
		t.Error("expected line number")
	}
	if !strings.Contains(got, "previous attempt") {
		t.Error("expected prior attempt reference")
	}
	if !strings.Contains(got, "added a comment but wrong format") {
		t.Error("expected prior output")
	}
	if !strings.Contains(got, "different approach") {
		t.Error("expected retry instruction")
	}
}

func TestBuildRetryPromptTruncation(t *testing.T) {
	task := Task{
		File:   "main.go",
		Issues: []linter.Issue{{File: "main.go", Line: 1, Message: "err", Linter: "x"}},
	}
	longOutput := strings.Repeat("x", 3000)
	got := BuildRetryPrompt(task, longOutput)
	if strings.Contains(got, strings.Repeat("x", 2500)) {
		t.Error("expected truncation of long output")
	}
	if !strings.Contains(got, "...") {
		t.Error("expected truncation marker")
	}
}

func TestBuildExplorationPrompt(t *testing.T) {
	task := Task{
		File:   "handler.go",
		Issues: []linter.Issue{{File: "handler.go", Line: 20, Message: "cyclomatic complexity", Linter: "revive"}},
	}
	got := BuildExplorationPrompt(task, "tried simplifying condition")
	if !strings.Contains(got, "WARNING") {
		t.Error("expected WARNING directive")
	}
	if !strings.Contains(got, "refactoring") {
		t.Error("expected refactoring instruction")
	}
	if !strings.Contains(got, "tried simplifying condition") {
		t.Error("expected prior output")
	}
}

func TestBuildExplorationPromptTruncation(t *testing.T) {
	task := Task{
		File:   "main.go",
		Issues: []linter.Issue{{File: "main.go", Line: 1, Message: "err", Linter: "x"}},
	}
	longOutput := strings.Repeat("y", 3000)
	got := BuildExplorationPrompt(task, longOutput)
	if strings.Contains(got, strings.Repeat("y", 2500)) {
		t.Error("expected truncation")
	}
}

func TestTruncateOutputShort(t *testing.T) {
	got := truncateOutput("hello", 10)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateOutputExact(t *testing.T) {
	got := truncateOutput("hello", 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateOutputLong(t *testing.T) {
	got := truncateOutput("hello world", 5)
	if got != "hello..." {
		t.Errorf("expected %q, got %q", "hello...", got)
	}
}
