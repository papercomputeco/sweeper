package dotdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTargetOverride(t *testing.T) {
	dir := t.TempDir()
	m := NewManager()
	got, err := m.Target(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("expected %s, got %s", dir, got)
	}
}

func TestTargetLocalDir(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, ".sweeper")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	got, err := m.TargetIn(tmp, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != local {
		t.Errorf("expected %s, got %s", local, got)
	}
}

func TestTargetHomeDir(t *testing.T) {
	home := t.TempDir()
	homeDir := filepath.Join(home, ".sweeper")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	got, err := m.TargetWithHome(t.TempDir(), home, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != homeDir {
		t.Errorf("expected %s, got %s", homeDir, got)
	}
}

func TestTargetNoneFound(t *testing.T) {
	m := NewManager()
	got, err := m.TargetWithHome(t.TempDir(), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}
