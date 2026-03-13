package observer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/papercomputeco/sweeper/pkg/telemetry"
)

type Insight struct {
	Linter      string
	Attempts    int
	Successes   int
	SuccessRate float64
}

type Observer struct {
	dir string
}

func New(dir string) *Observer {
	return &Observer{dir: dir}
}

func (o *Observer) Analyze() ([]Insight, error) {
	events, err := o.readAll()
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return o.computeInsights(events), nil
}

func (o *Observer) readAll() ([]telemetry.Event, error) {
	files, err := filepath.Glob(filepath.Join(o.dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var events []telemetry.Event
	for _, f := range files {
		fileEvents, err := o.readFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		events = append(events, fileEvents...)
	}
	return events, nil
}

func (o *Observer) readFile(path string) ([]telemetry.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []telemetry.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e telemetry.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, scanner.Err()
}

func (o *Observer) computeInsights(events []telemetry.Event) []Insight {
	type stats struct {
		attempts  int
		successes int
	}
	byLinter := make(map[string]*stats)
	for _, e := range events {
		if e.Type != "fix_attempt" {
			continue
		}
		linter, _ := e.Data["linter"].(string)
		if linter == "" {
			linter = "unknown"
		}
		s, ok := byLinter[linter]
		if !ok {
			s = &stats{}
			byLinter[linter] = s
		}
		s.attempts++
		if success, _ := e.Data["success"].(bool); success {
			s.successes++
		}
	}
	insights := make([]Insight, 0, len(byLinter))
	for linter, s := range byLinter {
		rate := 0.0
		if s.attempts > 0 {
			rate = float64(s.successes) / float64(s.attempts)
		}
		insights = append(insights, Insight{
			Linter: linter, Attempts: s.attempts,
			Successes: s.successes, SuccessRate: rate,
		})
	}
	return insights
}
