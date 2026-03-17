package telemetry

import (
	"context"
	"errors"
)

// MultiPublisher fans out events to multiple backends.
// All backends receive every event. Errors are joined; a single
// backend failure does not prevent delivery to the others.
type MultiPublisher struct {
	publishers []Publisher
}

var _ Publisher = (*MultiPublisher)(nil)

func NewMultiPublisher(publishers ...Publisher) *MultiPublisher {
	return &MultiPublisher{publishers: publishers}
}

func (m *MultiPublisher) Publish(ctx context.Context, event Event) error {
	var errs []error
	for _, p := range m.publishers {
		if err := p.Publish(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *MultiPublisher) Close() error {
	var errs []error
	for _, p := range m.publishers {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
