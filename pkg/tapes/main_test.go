package tapes

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Override HOME so FindDB's home fallback doesn't find real tapes installations
	// on developer machines, keeping tests hermetic.
	dir, err := os.MkdirTemp("", "tapes-test-home-*")
	if err == nil {
		_ = os.Setenv("HOME", dir)
	}
	code := m.Run()
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	os.Exit(code)
}
