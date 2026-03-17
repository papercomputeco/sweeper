package confluent

import (
	"context"
	"testing"
	"time"

	"github.com/papercomputeco/sweeper/pkg/telemetry"
)

type mockWriter struct {
	messages [][]byte
	closed   bool
}

func (m *mockWriter) WriteMessages(ctx context.Context, msgs ...message) error {
	for _, msg := range msgs {
		m.messages = append(m.messages, msg.Value)
	}
	return nil
}

func (m *mockWriter) Close() error {
	m.closed = true
	return nil
}

func TestPublishWritesToKafka(t *testing.T) {
	w := &mockWriter{}
	pub, err := newPublisherWithWriter(Config{Topic: "test"}, w)
	if err != nil {
		t.Fatal(err)
	}

	event := telemetry.Event{
		Timestamp: time.Now(),
		Type:      "fix_attempt",
		Data:      map[string]any{"file": "main.go"},
	}
	if err := pub.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(w.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(w.messages))
	}
}

func TestNewPublisherWithNilWriter(t *testing.T) {
	_, err := newPublisherWithWriter(Config{Topic: "test"}, nil)
	if err == nil {
		t.Error("expected error with nil writer")
	}
}

func TestNewPublisherValidation(t *testing.T) {
	_, err := NewPublisher(Config{})
	if err == nil {
		t.Error("expected error with empty config")
	}

	_, err = NewPublisher(Config{Brokers: []string{"b:9092"}})
	if err == nil {
		t.Error("expected error with missing topic")
	}
}

func TestCloseClosesWriter(t *testing.T) {
	w := &mockWriter{}
	pub, err := newPublisherWithWriter(Config{Topic: "test"}, w)
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.Close(); err != nil {
		t.Fatal(err)
	}
	if !w.closed {
		t.Error("expected writer to be closed")
	}
}
