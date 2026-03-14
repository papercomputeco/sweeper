package observer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/papercomputeco/sweeper/pkg/tapes"
	"github.com/papercomputeco/sweeper/pkg/telemetry"
)

type Insight struct {
	Linter      string
	Attempts    int
	Successes   int
	SuccessRate float64
	TotalTokens int
}

type Observer struct {
	dir          string
	tapesReader  *tapes.Reader
	tapesEnabled bool
}

type ObserverOption func(*Observer)

func WithTapesReader(r *tapes.Reader) ObserverOption {
	return func(o *Observer) {
		o.tapesReader = r
		o.tapesEnabled = true
	}
}

func WithTapesEnabled(enabled bool) ObserverOption {
	return func(o *Observer) {
		o.tapesEnabled = enabled
	}
}

func New(dir string, opts ...ObserverOption) *Observer {
	o := &Observer{dir: dir}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *Observer) Analyze() ([]Insight, error) {
	events, err := o.readAll()
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	insights := o.computeInsights(events)
	if o.tapesEnabled && o.tapesReader != nil {
		o.enrichWithTapes(insights)
	}
	return insights, nil
}

const tapesSessionWindow = 50

func (o *Observer) enrichWithTapes(insights []Insight) {
	hashes, err := o.tapesReader.RecentSessions(tapesSessionWindow)
	if err != nil {
		return
	}

	totalTokens := 0
	for _, hash := range hashes {
		session, err := o.tapesReader.GetSession(hash)
		if err != nil {
			continue
		}
		totalTokens += session.TotalPromptTokens + session.TotalCompletionTokens
	}

	if totalTokens == 0 {
		return
	}

	totalAttempts := 0
	for _, ins := range insights {
		totalAttempts += ins.Attempts
	}
	if totalAttempts == 0 {
		return
	}

	for i := range insights {
		insights[i].TotalTokens = (totalTokens * insights[i].Attempts) / totalAttempts
	}
}

func (o *Observer) readAll() ([]telemetry.Event, error) {
	files, _ := filepath.Glob(filepath.Join(o.dir, "*.jsonl"))
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
	defer func() { _ = f.Close() }()
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

// HistoricalInsight provides cross-run trend data computed from telemetry history.
type HistoricalInsight struct {
	SuccessRateTrend      []float64
	RoundEffectiveness    map[int]float64
	StrategyEffectiveness map[string]float64
	TotalRuns             int
}

// AnalyzeHistory computes historical trends from all telemetry events.
func (o *Observer) AnalyzeHistory() (HistoricalInsight, error) {
	events, err := o.readAll()
	if err != nil {
		return HistoricalInsight{}, err
	}

	insight := HistoricalInsight{
		RoundEffectiveness:    make(map[int]float64),
		StrategyEffectiveness: make(map[string]float64),
	}

	if len(events) == 0 {
		return insight, nil
	}

	// Group fix_attempt events by date (proxy for "run")
	type runStats struct {
		attempts  int
		successes int
	}
	byDate := make(map[string]*runStats)
	var dateOrder []string

	type strategyStats struct {
		attempts  int
		successes int
	}
	roundFixed := make(map[int]int)
	totalFixed := 0
	byStrategy := make(map[string]*strategyStats)

	for _, e := range events {
		if e.Type != "fix_attempt" {
			continue
		}

		date := e.Timestamp.Format("2006-01-02")
		rs, ok := byDate[date]
		if !ok {
			rs = &runStats{}
			byDate[date] = rs
			dateOrder = append(dateOrder, date)
		}
		rs.attempts++
		success, _ := e.Data["success"].(bool)
		if success {
			rs.successes++
		}

		// Round effectiveness
		round := 1
		if r, ok := e.Data["round"].(float64); ok {
			round = int(r)
		}
		if success {
			roundFixed[round]++
			totalFixed++
		}

		// Strategy effectiveness
		strategy := "standard"
		if s, ok := e.Data["strategy"].(string); ok {
			strategy = s
		}
		ss, ok := byStrategy[strategy]
		if !ok {
			ss = &strategyStats{}
			byStrategy[strategy] = ss
		}
		ss.attempts++
		if success {
			ss.successes++
		}
	}

	insight.TotalRuns = len(byDate)

	// Success rate trend (chronological)
	for _, date := range dateOrder {
		rs := byDate[date]
		if rs.attempts > 0 {
			insight.SuccessRateTrend = append(insight.SuccessRateTrend, float64(rs.successes)/float64(rs.attempts))
		}
	}

	// Round effectiveness
	if totalFixed > 0 {
		for round, fixed := range roundFixed {
			insight.RoundEffectiveness[round] = float64(fixed) / float64(totalFixed)
		}
	}

	// Strategy effectiveness
	for strategy, ss := range byStrategy {
		if ss.attempts > 0 {
			insight.StrategyEffectiveness[strategy] = float64(ss.successes) / float64(ss.attempts)
		}
	}

	return insight, nil
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
