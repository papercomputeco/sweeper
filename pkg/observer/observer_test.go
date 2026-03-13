package observer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/papercomputeco/sweeper/pkg/telemetry"
)

func writeEvents(t *testing.T, dir string, events []telemetry.Event) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	f, _ := os.Create(filepath.Join(dir, "2026-03-13.jsonl"))
	defer f.Close()
	for _, e := range events {
		data, _ := json.Marshal(e)
		f.Write(append(data, '\n'))
	}
}

func TestObserveSuccessRate(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": false}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "ineffassign", "success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) == 0 {
		t.Fatal("expected at least one insight")
	}
}

func TestObserveEmptyDir(t *testing.T) {
	dir := t.TempDir()
	obs := New(dir)
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) != 0 {
		t.Fatalf("expected 0 insights from empty dir, got %d", len(insights))
	}
}
