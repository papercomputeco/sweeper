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
	defer func() { _ = pub.Close() }()
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
	if err := pub.Close(); err != nil {
		t.Fatal(err)
	}
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

func TestPublishMultipleReusesFile(t *testing.T) {
	dir := t.TempDir()
	pub := NewPublisher(dir)
	defer func() { _ = pub.Close() }()
	e1 := Event{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"file": "a.go"}}
	e2 := Event{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"file": "b.go"}}
	if err := pub.Publish(e1); err != nil {
		t.Fatal(err)
	}
	if err := pub.Publish(e2); err != nil {
		t.Fatal(err)
	}
	if err := pub.Close(); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("expected 1 jsonl file after multiple publishes, got %d", len(files))
	}
}

func TestPublishInvalidDir(t *testing.T) {
	pub := NewPublisher("/nonexistent/path/that/cannot/exist")
	defer func() { _ = pub.Close() }()
	err := pub.Publish(Event{Timestamp: time.Now(), Type: "test"})
	if err == nil {
		t.Error("expected error when publishing to invalid directory")
	}
}

func TestCloseWithoutPublish(t *testing.T) {
	dir := t.TempDir()
	pub := NewPublisher(dir)
	// Close without ever publishing — should not error.
	if err := pub.Close(); err != nil {
		t.Errorf("unexpected error closing without publish: %v", err)
	}
}

func TestPublishMarshalError(t *testing.T) {
	dir := t.TempDir()
	pub := NewPublisher(dir)
	defer func() { _ = pub.Close() }()
	// Channels are not JSON-serializable.
	event := Event{
		Timestamp: time.Now(),
		Type:      "test",
		Data:      map[string]any{"bad": make(chan int)},
	}
	err := pub.Publish(event)
	if err == nil {
		t.Error("expected error when marshaling channel")
	}
}
