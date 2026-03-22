package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type JSONLPublisher struct {
	dir  string
	file *os.File
	mu   sync.Mutex
}

var _ Publisher = (*JSONLPublisher)(nil)

func NewJSONLPublisher(dir string) *JSONLPublisher {
	_ = os.MkdirAll(dir, 0o755)
	return &JSONLPublisher{dir: dir}
}

func (p *JSONLPublisher) ensureFile() error {
	if p.file != nil {
		return nil
	}
	name := time.Now().Format("2006-01-02") + ".jsonl"
	f, err := os.OpenFile(filepath.Join(p.dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	p.file = f
	return nil
}

func (p *JSONLPublisher) Publish(_ context.Context, event Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureFile(); err != nil {
		return fmt.Errorf("opening telemetry file: %w", err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = p.file.Write(data)
	return err
}

func (p *JSONLPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file != nil {
		err := p.file.Close()
		p.file = nil
		return err
	}
	return nil
}

// NewPublisher is a backwards-compatible alias for NewJSONLPublisher.
func NewPublisher(dir string) *JSONLPublisher {
	return NewJSONLPublisher(dir)
}
