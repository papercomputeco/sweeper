package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSessionDoc(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sweeper")
	cfg := Config{
		Objective:   "Fix lint issues",
		LintCommand: "golangci-lint run ./...",
		TargetDir:   "/tmp/project",
		MaxRounds:   3,
		Constraints: "none",
	}

	path, err := Generate(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "sweeper.md" {
		t.Errorf("expected sweeper.md, got %s", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "Fix lint issues") {
		t.Error("expected objective in session doc")
	}
	if !strings.Contains(content, "golangci-lint run ./...") {
		t.Error("expected lint command in session doc")
	}
	if !strings.Contains(content, "Max rounds:** 3") {
		t.Error("expected max rounds in session doc")
	}
	if !strings.Contains(content, "/tmp/project") {
		t.Error("expected target dir in session doc")
	}
}

func TestGenerateSessionDocResume(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sweeper")
	os.MkdirAll(dir, 0o755)

	existing := "existing content"
	existingPath := filepath.Join(dir, "sweeper.md")
	os.WriteFile(existingPath, []byte(existing), 0o644)

	cfg := Config{
		Objective:   "Fix lint issues",
		LintCommand: "golangci-lint run ./...",
		TargetDir:   "/tmp/project",
		MaxRounds:   3,
	}

	path, err := Generate(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Error("existing sweeper.md should not be overwritten")
	}
}

func TestUpdateStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sweeper.md")
	os.WriteFile(path, []byte("# Session\n"), 0o644)

	err := UpdateStatus(path, 1, 10, 3, 7)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "Round 1") {
		t.Error("expected Round 1 in status update")
	}
	if !strings.Contains(content, "Fixed: 3") {
		t.Error("expected Fixed: 3 in status update")
	}
	if !strings.Contains(content, "Remaining: 7") {
		t.Error("expected Remaining: 7 in status update")
	}
	if !strings.Contains(content, "Issues at start: 10") {
		t.Error("expected Issues at start: 10 in status update")
	}
}

func TestUpdateStatusMissingFile(t *testing.T) {
	err := UpdateStatus("/nonexistent/path/sweeper.md", 1, 5, 2, 3)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGenerateSessionDocMkdirError(t *testing.T) {
	// Use a path under a read-only file to force MkdirAll to fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	os.WriteFile(blocker, []byte("x"), 0o444)

	dir := filepath.Join(blocker, "subdir")
	cfg := Config{Objective: "test"}

	_, err := Generate(dir, cfg)
	if err == nil {
		t.Error("expected error when MkdirAll fails")
	}
}

func TestGenerateSessionDocWriteError(t *testing.T) {
	// Create the directory as read-only so WriteFile fails.
	dir := filepath.Join(t.TempDir(), ".sweeper")
	os.MkdirAll(dir, 0o755)
	// Make directory read-only after creation so WriteFile fails.
	os.Chmod(dir, 0o555)
	defer os.Chmod(dir, 0o755) // cleanup

	cfg := Config{Objective: "test"}

	_, err := Generate(dir, cfg)
	if err == nil {
		t.Error("expected error when WriteFile fails")
	}
}
