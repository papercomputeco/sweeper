package agent

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Override HOME so tapes detection doesn't find real installations,
	// allowing us to test the "tapes not available" code paths.
	dir, err := os.MkdirTemp("", "agent-test-home-*")
	if err == nil {
		os.Setenv("HOME", dir)
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}
