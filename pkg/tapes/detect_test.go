package tapes

import (
	"errors"
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

func TestFindDBSqlite(t *testing.T) {
	dir := t.TempDir()
	tapesDir := filepath.Join(dir, ".tapes")
	os.MkdirAll(tapesDir, 0o755)
	dbPath := filepath.Join(tapesDir, "tapes.sqlite")
	os.WriteFile(dbPath, []byte("fake"), 0o644)

	result := FindDB(dir)
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
	result := FindDB(dir)
	if result != "" {
		t.Errorf("expected empty string with no home tapes, got %s", result)
	}
}

func TestCheckInstallationWithDBPath(t *testing.T) {
	dir := t.TempDir()
	tapesDir := filepath.Join(dir, ".tapes")
	os.MkdirAll(tapesDir, 0o755)
	dbPath := filepath.Join(tapesDir, "tapes.db")
	os.WriteFile(dbPath, []byte("fake"), 0o644)

	status := CheckInstallation(dbPath)
	if !status.Available {
		t.Error("expected Available when DB path provided")
	}
	if status.DBPath != dbPath {
		t.Errorf("expected DBPath=%s, got %s", dbPath, status.DBPath)
	}
}

func TestCheckInstallationCLIFoundNoDB(t *testing.T) {
	orig := lookPath
	lookPath = func(name string) (string, error) { return "/usr/bin/tapes", nil }
	t.Cleanup(func() { lookPath = orig })

	status := CheckInstallation("")
	if status.Available {
		t.Error("should not be available without DB")
	}
	if status.Message == "" {
		t.Error("expected a message about running tapes init")
	}
}

func TestCheckInstallationNotInstalled(t *testing.T) {
	orig := lookPath
	lookPath = func(name string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { lookPath = orig })

	status := CheckInstallation("")
	if status.Available {
		t.Error("should not be available")
	}
	if status.Message == "" {
		t.Error("expected a message about installing tapes")
	}
}
