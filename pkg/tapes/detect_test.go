package tapes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDBFound(t *testing.T) {
	dir := t.TempDir()
	tapesDir := filepath.Join(dir, ".tapes")
	os.MkdirAll(tapesDir, 0o755)
	dbPath := filepath.Join(tapesDir, "tapes.db")
	os.WriteFile(dbPath, []byte("fake"), 0o644)

	result := FindDB(dir)
	if result == "" {
		t.Fatal("expected to find tapes.db")
	}
	if result != dbPath {
		t.Errorf("expected %s, got %s", dbPath, result)
	}
}

func TestFindDBNotFound(t *testing.T) {
	dir := t.TempDir()
	result := FindDB(dir)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestFindDBHomeFallback(t *testing.T) {
	dir := t.TempDir()
	// TestMain sets HOME to an empty temp dir, so no home fallback should be found
	result := FindDB(dir)
	if result != "" {
		t.Errorf("expected empty string with no home tapes, got %s", result)
	}
}

func TestCheckInstallation(t *testing.T) {
	status := CheckInstallation("")
	if status.Available {
		if status.DBPath == "" {
			t.Error("available but no DB path")
		}
	}
}
