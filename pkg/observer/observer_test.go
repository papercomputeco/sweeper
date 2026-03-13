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
