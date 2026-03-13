package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPublishWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	pub := NewPublisher(dir)
	defer pub.Close()
	event := Event{
		Timestamp: time.Now(),
		Type:      "fix_attempt",
		Data: map[string]any{
			"file":    "server.go",
			"success": true,
			"issues":  3,
		},
	}
	if err := pub.Publish(event); err != nil {
		t.Fatal(err)
	}
	pub.Close()
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("expected 1 jsonl file, got %d", len(files))
	}
	data, _ := os.ReadFile(files[0])
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded.Type != "fix_attempt" {
		t.Errorf("expected type fix_attempt, got %s", decoded.Type)
	}
}
