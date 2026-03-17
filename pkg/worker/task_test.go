package worker

import (
	"os"
	"path/filepath"
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
	if !strings.Contains(got, "previous attempt") && !strings.Contains(got, "was tried") {
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
	if !strings.Contains(got, "Previous approaches have not resolved") {
		t.Error("expected stagnation context")
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

func TestBuildAPIPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := Task{
		File:   "main.go",
		Dir:    dir,
		Issues: []linter.Issue{{File: "main.go", Line: 1, Message: "unused import", Linter: "revive"}},
	}
	got := BuildAPIPrompt(task)
	if !strings.Contains(got, "main.go") {
		t.Error("expected file name")
	}
	if !strings.Contains(got, "package main") {
		t.Error("expected file content")
	}
	if !strings.Contains(got, "unified diff") {
		t.Error("expected diff instructions")
	}
	if !strings.Contains(got, "```diff") {
		t.Error("expected diff marker instructions")
	}
}

func TestBuildAPIPromptMissingFile(t *testing.T) {
	task := Task{
		File:   "nonexistent.go",
		Dir:    t.TempDir(),
		Issues: []linter.Issue{{File: "nonexistent.go", Line: 1, Message: "err", Linter: "x"}},
	}
	got := BuildAPIPrompt(task)
	if !strings.Contains(got, "could not read") {
		t.Error("expected graceful degradation for missing file")
	}
}

func TestBuildAPIRetryPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package handler\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := Task{
		File:   "handler.go",
		Dir:    dir,
		Issues: []linter.Issue{{File: "handler.go", Line: 5, Message: "cyclomatic", Linter: "revive"}},
	}
	got := BuildAPIRetryPrompt(task, "tried adding else branch")
	if !strings.Contains(got, "previous attempt") || !strings.Contains(got, "was tried") {
		t.Error("expected prior attempt reference")
	}
	if !strings.Contains(got, "tried adding else branch") {
		t.Error("expected prior output")
	}
	if !strings.Contains(got, "package handler") {
		t.Error("expected file content")
	}
	if !strings.Contains(got, "different approach") {
		t.Error("expected retry instruction")
	}
}

func TestBuildAPIExplorationPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "server.go"), []byte("package server\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := Task{
		File:   "server.go",
		Dir:    dir,
		Issues: []linter.Issue{{File: "server.go", Line: 10, Message: "too complex", Linter: "gocyclo"}},
	}
	got := BuildAPIExplorationPrompt(task, "simplified conditions")
	if !strings.Contains(got, "Previous approaches have not resolved") {
		t.Error("expected stagnation context")
	}
	if !strings.Contains(got, "refactoring") {
		t.Error("expected refactoring instruction")
	}
	if !strings.Contains(got, "package server") {
		t.Error("expected file content")
	}
}

func TestReadFileContent(t *testing.T) {
	dir := t.TempDir()
	content := "package main\n\nfunc hello() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "test.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readFileContent(dir, "test.go")
	if !strings.Contains(got, content) {
		t.Errorf("expected file content, got %s", got)
	}
	if !strings.HasPrefix(got, "```\n") {
		t.Error("expected opening code fence")
	}
}

func TestReadFileContentMissing(t *testing.T) {
	got := readFileContent(t.TempDir(), "nope.go")
	if !strings.Contains(got, "could not read") {
		t.Error("expected error message for missing file")
	}
}
