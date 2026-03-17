package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"
)

type spyPublisher struct {
	events      []Event
	closed      bool
	publishErr  error
	closeErr    error
}

func (s *spyPublisher) Publish(_ context.Context, event Event) error {
	s.events = append(s.events, event)
	return s.publishErr
}

func (s *spyPublisher) Close() error {
	s.closed = true
	return s.closeErr
}

func TestMultiPublisherFansOut(t *testing.T) {
	a := &spyPublisher{}
	b := &spyPublisher{}
	mp := NewMultiPublisher(a, b)

	event := Event{Timestamp: time.Now(), Type: "test", Data: map[string]any{"x": 1}}
	if err := mp.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(a.events) != 1 {
		t.Errorf("expected 1 event in a, got %d", len(a.events))
	}
	if len(b.events) != 1 {
		t.Errorf("expected 1 event in b, got %d", len(b.events))
	}
	if err := mp.Close(); err != nil {
		t.Fatal(err)
	}
	if !a.closed || !b.closed {
		t.Error("expected both publishers closed")
	}
}

func TestMultiPublisherSingleBackend(t *testing.T) {
	a := &spyPublisher{}
	mp := NewMultiPublisher(a)
	event := Event{Timestamp: time.Now(), Type: "test"}
	if err := mp.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if len(a.events) != 1 {
		t.Errorf("expected 1 event, got %d", len(a.events))
	}
}

func TestMultiPublisherPublishError(t *testing.T) {
	sentinel := errors.New("publish failure")
	a := &spyPublisher{publishErr: sentinel}
	b := &spyPublisher{}
	mp := NewMultiPublisher(a, b)

	err := mp.Publish(context.Background(), Event{Timestamp: time.Now(), Type: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error in joined error, got %v", err)
	}
	// b still receives the event despite a's failure
	if len(b.events) != 1 {
		t.Errorf("expected b to receive event despite a's error, got %d events", len(b.events))
	}
}

func TestMultiPublisherCloseError(t *testing.T) {
	sentinel := errors.New("close failure")
	a := &spyPublisher{closeErr: sentinel}
	b := &spyPublisher{}
	mp := NewMultiPublisher(a, b)

	err := mp.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error in joined error, got %v", err)
	}
	// b is still closed despite a's failure
	if !b.closed {
		t.Error("expected b to be closed despite a's error")
	}
}
