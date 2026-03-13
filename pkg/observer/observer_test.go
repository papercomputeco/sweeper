package observer

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/papercomputeco/sweeper/pkg/tapes"
	"github.com/papercomputeco/sweeper/pkg/telemetry"
	_ "modernc.org/sqlite"
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

func setupTapesDB(t *testing.T) *tapes.Reader {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE nodes (
			hash TEXT PRIMARY KEY,
			parent_hash TEXT,
			role TEXT,
			content JSON,
			model TEXT,
			provider TEXT,
			agent_name TEXT,
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			cache_creation_input_tokens INTEGER,
			cache_read_input_tokens INTEGER,
			project TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	// Insert some sessions with token usage.
	_, err = db.Exec(
		`INSERT INTO nodes (hash, parent_hash, role, content, model, prompt_tokens, completion_tokens, total_tokens, created_at)
		 VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?)`,
		"root1", "user", `[{"type":"text","text":"fix"}]`, "claude-sonnet-4-20250514", 100, 50, 150, time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return tapes.NewReaderFromDB(db)
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

func TestObserveWithTapesEnabledNilReader(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir, WithTapesEnabled(true))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) == 0 {
		t.Fatal("expected at least one insight")
	}
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 when tapesReader is nil, got %d", ins.TotalTokens)
		}
	}
}

func TestObserveWithoutTapes(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": false}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir, WithTapesEnabled(false))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) == 0 {
		t.Fatal("expected at least one insight")
	}
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 when tapesEnabled=false, got %d", ins.TotalTokens)
		}
	}
}

func TestObserveWithTapesEnrichment(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": false}},
	}
	writeEvents(t, dir, events)
	reader := setupTapesDB(t)
	obs := New(dir, WithTapesReader(reader))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) == 0 {
		t.Fatal("expected at least one insight")
	}
	// With tapes enrichment, TotalTokens should be set.
	found := false
	for _, ins := range insights {
		if ins.TotalTokens > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one insight with TotalTokens > 0 after tapes enrichment")
	}
}

func TestObserveWithTapesZeroTokens(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)

	// Create a tapes reader with sessions that have zero tokens.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE nodes (
			hash TEXT PRIMARY KEY, parent_hash TEXT, role TEXT, content JSON,
			model TEXT, provider TEXT, agent_name TEXT, prompt_tokens INTEGER,
			completion_tokens INTEGER, total_tokens INTEGER,
			cache_creation_input_tokens INTEGER, cache_read_input_tokens INTEGER,
			project TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		`INSERT INTO nodes (hash, parent_hash, role, content, model, prompt_tokens, completion_tokens, total_tokens, created_at)
		 VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?)`,
		"root1", "user", `[]`, "model", 0, 0, 0, time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reader := tapes.NewReaderFromDB(db)
	obs := New(dir, WithTapesReader(reader))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 when sessions have zero tokens, got %d", ins.TotalTokens)
		}
	}
}

func TestComputeInsightsUnknownLinter(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(insights))
	}
	if insights[0].Linter != "unknown" {
		t.Errorf("expected linter 'unknown', got %s", insights[0].Linter)
	}
}

func TestComputeInsightsNonFixEvent(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "other_event", Data: map[string]any{"linter": "revive"}},
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight (non-fix event skipped), got %d", len(insights))
	}
}

func TestReadFileBadJSON(t *testing.T) {
	dir := t.TempDir()
	// Write a file with invalid JSON lines.
	f, _ := os.Create(filepath.Join(dir, "bad.jsonl"))
	f.WriteString("not valid json\n")
	f.WriteString(`{"timestamp":"2026-03-13T00:00:00Z","type":"fix_attempt","data":{"linter":"revive","success":true}}` + "\n")
	f.Close()
	obs := New(dir)
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	// Bad line is skipped, valid line is parsed.
	if len(insights) != 1 {
		t.Fatalf("expected 1 insight from valid line, got %d", len(insights))
	}
}

func TestReadFileOpenError(t *testing.T) {
	dir := t.TempDir()
	// Create a .jsonl file that cannot be read.
	path := filepath.Join(dir, "unreadable.jsonl")
	os.WriteFile(path, []byte("data"), 0o644)
	os.Chmod(path, 0o000)
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	obs := New(dir)
	_, err := obs.Analyze()
	if err == nil {
		t.Error("expected error when reading unreadable file")
	}
}

func TestAnalyzeHistoryEmpty(t *testing.T) {
	dir := t.TempDir()
	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.TotalRuns != 0 {
		t.Errorf("expected 0 runs, got %d", hist.TotalRuns)
	}
}

func TestAnalyzeHistoryLegacyEvents(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	events := []telemetry.Event{
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": false}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.TotalRuns != 1 {
		t.Errorf("expected 1 run, got %d", hist.TotalRuns)
	}
	if len(hist.SuccessRateTrend) != 1 {
		t.Fatalf("expected 1 trend point, got %d", len(hist.SuccessRateTrend))
	}
	if hist.SuccessRateTrend[0] != 0.5 {
		t.Errorf("expected 0.5 success rate, got %f", hist.SuccessRateTrend[0])
	}
	// Legacy events default to round=1, strategy=standard
	if rate, ok := hist.RoundEffectiveness[1]; !ok || rate != 1.0 {
		t.Errorf("expected round 1 = 1.0, got %v", hist.RoundEffectiveness)
	}
	if rate, ok := hist.StrategyEffectiveness["standard"]; !ok || rate != 0.5 {
		t.Errorf("expected standard = 0.5, got %v", hist.StrategyEffectiveness)
	}
}

func TestAnalyzeHistoryWithRounds(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	events := []telemetry.Event{
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": true, "round": float64(1), "strategy": "standard"}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": false, "round": float64(1), "strategy": "standard"}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": true, "round": float64(2), "strategy": "retry"}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": false, "round": float64(3), "strategy": "exploration"}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	// 2 successes total: 1 in round 1, 1 in round 2
	if hist.RoundEffectiveness[1] != 0.5 {
		t.Errorf("expected round 1 = 0.5, got %f", hist.RoundEffectiveness[1])
	}
	if hist.RoundEffectiveness[2] != 0.5 {
		t.Errorf("expected round 2 = 0.5, got %f", hist.RoundEffectiveness[2])
	}
	// Strategy: standard 1/2=0.5, retry 1/1=1.0, exploration 0/1=0.0
	if hist.StrategyEffectiveness["standard"] != 0.5 {
		t.Errorf("expected standard = 0.5, got %f", hist.StrategyEffectiveness["standard"])
	}
	if hist.StrategyEffectiveness["retry"] != 1.0 {
		t.Errorf("expected retry = 1.0, got %f", hist.StrategyEffectiveness["retry"])
	}
	if hist.StrategyEffectiveness["exploration"] != 0.0 {
		t.Errorf("expected exploration = 0.0, got %f", hist.StrategyEffectiveness["exploration"])
	}
}

func TestAnalyzeHistoryMultipleRuns(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir, 0o755)
	// Write two date files to simulate two runs
	ts1 := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)

	f1, _ := os.Create(filepath.Join(dir, "2026-03-12.jsonl"))
	e1 := telemetry.Event{Timestamp: ts1, Type: "fix_attempt", Data: map[string]any{"success": false}}
	data1, _ := json.Marshal(e1)
	f1.Write(append(data1, '\n'))
	f1.Close()

	f2, _ := os.Create(filepath.Join(dir, "2026-03-13.jsonl"))
	e2 := telemetry.Event{Timestamp: ts2, Type: "fix_attempt", Data: map[string]any{"success": true}}
	data2, _ := json.Marshal(e2)
	f2.Write(append(data2, '\n'))
	f2.Close()

	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.TotalRuns != 2 {
		t.Errorf("expected 2 runs, got %d", hist.TotalRuns)
	}
	if len(hist.SuccessRateTrend) != 2 {
		t.Fatalf("expected 2 trend points, got %d", len(hist.SuccessRateTrend))
	}
}

func TestAnalyzeHistoryNoSuccesses(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	events := []telemetry.Event{
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": false}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": false}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	// No successes means no round effectiveness entries
	if len(hist.RoundEffectiveness) != 0 {
		t.Errorf("expected empty round effectiveness, got %v", hist.RoundEffectiveness)
	}
}

func TestAnalyzeHistorySkipsNonFixEvents(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	events := []telemetry.Event{
		{Timestamp: ts, Type: "round_complete", Data: map[string]any{"round": float64(1)}},
		{Timestamp: ts, Type: "fix_attempt", Data: map[string]any{"success": true}},
	}
	writeEvents(t, dir, events)
	obs := New(dir)
	hist, err := obs.AnalyzeHistory()
	if err != nil {
		t.Fatal(err)
	}
	if hist.TotalRuns != 1 {
		t.Errorf("expected 1 run (round_complete skipped), got %d", hist.TotalRuns)
	}
}

func TestAnalyzeHistoryReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unreadable.jsonl")
	os.WriteFile(path, []byte("data"), 0o644)
	os.Chmod(path, 0o000)
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	obs := New(dir)
	_, err := obs.AnalyzeHistory()
	if err == nil {
		t.Error("expected error reading unreadable file")
	}
}

func TestEnrichWithTapesZeroAttempts(t *testing.T) {
	dir := t.TempDir()
	// Write non-fix_attempt events so computeInsights returns empty insights.
	// This means enrichWithTapes receives insights with totalAttempts == 0.
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "round_complete", Data: map[string]any{"round": float64(1)}},
	}
	writeEvents(t, dir, events)
	reader := setupTapesDB(t) // has sessions with tokens
	obs := New(dir, WithTapesReader(reader))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	// No fix_attempt events → no insights → enrichWithTapes hits totalAttempts == 0 return.
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 with zero attempts, got %d", ins.TotalTokens)
		}
	}
}

func TestEnrichWithTapesGetSessionError(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)

	// Create a DB with a minimal schema that satisfies RecentSessions
	// (SELECT hash FROM nodes WHERE parent_hash IS NULL) but causes
	// GetSession to fail because it queries columns that don't exist
	// (role, content, model, prompt_tokens, etc.).
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE nodes (
		hash TEXT PRIMARY KEY,
		parent_hash TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`INSERT INTO nodes (hash, parent_hash) VALUES ('root1', NULL)`)
	reader := tapes.NewReaderFromDB(db)

	obs := New(dir, WithTapesReader(reader))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	// enrichWithTapes continues past GetSession errors; tokens stay at zero.
	if len(insights) == 0 {
		t.Fatal("expected insights even with GetSession errors")
	}
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 when GetSession errors, got %d", ins.TotalTokens)
		}
	}
}

func TestEnrichWithTapesReaderError(t *testing.T) {
	dir := t.TempDir()
	events := []telemetry.Event{
		{Timestamp: time.Now(), Type: "fix_attempt", Data: map[string]any{"linter": "revive", "success": true}},
	}
	writeEvents(t, dir, events)

	// Create a tapes reader with a closed DB to trigger errors.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE nodes (hash TEXT PRIMARY KEY, parent_hash TEXT, role TEXT, content JSON,
		model TEXT, provider TEXT, agent_name TEXT, prompt_tokens INTEGER,
		completion_tokens INTEGER, total_tokens INTEGER,
		cache_creation_input_tokens INTEGER, cache_read_input_tokens INTEGER,
		project TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	reader := tapes.NewReaderFromDB(db)
	db.Close() // Close to force errors

	obs := New(dir, WithTapesReader(reader))
	insights, err := obs.Analyze()
	if err != nil {
		t.Fatal(err)
	}
	// enrichWithTapes errors are swallowed; insights should still be returned.
	if len(insights) == 0 {
		t.Fatal("expected insights even with tapes errors")
	}
	for _, ins := range insights {
		if ins.TotalTokens != 0 {
			t.Errorf("expected TotalTokens=0 when tapes errors, got %d", ins.TotalTokens)
		}
	}
}
