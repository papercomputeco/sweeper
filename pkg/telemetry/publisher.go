package telemetry

import (
	"context"
	"time"
)

type Event struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}
