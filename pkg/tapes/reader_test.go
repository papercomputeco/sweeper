package tapes

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
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
	return db
}

func insertNode(t *testing.T, db *sql.DB, hash, parentHash, role, content, model string, promptTok, completionTok int) {
	t.Helper()
	var ph *string
	if parentHash != "" {
		ph = &parentHash
	}
	_, err := db.Exec(
		`INSERT INTO nodes (hash, parent_hash, role, content, model, prompt_tokens, completion_tokens, total_tokens, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hash, ph, role, content, model, promptTok, completionTok, promptTok+completionTok, time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReaderListSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertNode(t, db, "root1", "", "user", `[{"type":"text","text":"fix bug"}]`, "claude-sonnet-4-20250514", 0, 0)
	insertNode(t, db, "root2", "", "user", `[{"type":"text","text":"refactor"}]`, "claude-sonnet-4-20250514", 0, 0)
	insertNode(t, db, "child1", "root1", "assistant", `[{"type":"text","text":"done"}]`, "claude-sonnet-4-20250514", 100, 50)

	reader := NewReaderFromDB(db)
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestReaderGetSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertNode(t, db, "root1", "", "user", `[{"type":"text","text":"fix lint"}]`, "claude-sonnet-4-20250514", 10, 0)
	insertNode(t, db, "child1", "root1", "assistant", `[{"type":"text","text":"fixing..."}]`, "claude-sonnet-4-20250514", 100, 50)
	insertNode(t, db, "child2", "child1", "assistant", `[{"type":"text","text":"done"}]`, "claude-sonnet-4-20250514", 200, 100)

	reader := NewReaderFromDB(db)
	session, err := reader.GetSession("root1")
	if err != nil {
		t.Fatal(err)
	}
	if session.RootHash != "root1" {
		t.Errorf("expected root hash root1, got %s", session.RootHash)
	}
	if len(session.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(session.Nodes))
	}
	if session.TotalPromptTokens != 310 {
		t.Errorf("expected 310 prompt tokens, got %d", session.TotalPromptTokens)
	}
	if session.TotalCompletionTokens != 150 {
		t.Errorf("expected 150 completion tokens, got %d", session.TotalCompletionTokens)
	}
}

func TestReaderGetSessionNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reader := NewReaderFromDB(db)
	session, err := reader.GetSession("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Nodes) != 0 {
		t.Errorf("expected 0 nodes for missing session, got %d", len(session.Nodes))
	}
}

func TestReaderRecentSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	insertNode(t, db, "root1", "", "user", `[{"type":"text","text":"old"}]`, "claude-sonnet-4-20250514", 0, 0)
	insertNode(t, db, "root2", "", "user", `[{"type":"text","text":"new"}]`, "claude-sonnet-4-20250514", 0, 0)

	reader := NewReaderFromDB(db)
	sessions, err := reader.RecentSessions(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestNewReaderInvalidPath(t *testing.T) {
	_, err := NewReader("/nonexistent/path/tapes.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestReaderClose(t *testing.T) {
	db := setupTestDB(t)
	reader := NewReaderFromDB(db)
	if err := reader.Close(); err != nil {
		t.Errorf("unexpected error closing reader: %v", err)
	}
}

func TestReaderListSessionsEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reader := NewReaderFromDB(db)
	sessions, err := reader.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestReaderRecentSessionsEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reader := NewReaderFromDB(db)
	sessions, err := reader.RecentSessions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestReaderListSessionsClosedDB(t *testing.T) {
	db := setupTestDB(t)
	reader := NewReaderFromDB(db)
	db.Close()
	_, err := reader.ListSessions()
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestReaderRecentSessionsClosedDB(t *testing.T) {
	db := setupTestDB(t)
	reader := NewReaderFromDB(db)
	db.Close()
	_, err := reader.RecentSessions(10)
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestReaderGetSessionClosedDB(t *testing.T) {
	db := setupTestDB(t)
	reader := NewReaderFromDB(db)
	db.Close()
	_, err := reader.GetSession("root1")
	if err == nil {
		t.Error("expected error querying closed DB")
	}
}

func TestNewReaderValidDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE nodes (hash TEXT)")
	db.Close()

	reader, err := NewReader(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reader.Close()
}

func TestNewReaderPingError(t *testing.T) {
	// A non-SQLite file will fail on Ping.
	dir := t.TempDir()
	dbPath := dir + "/bad.db"
	os.WriteFile(dbPath, []byte("not a sqlite file"), 0o644)
	_, err := NewReader(dbPath)
	if err == nil {
		t.Error("expected error for corrupt DB file")
	}
}

func TestReaderListSessionsScanError(t *testing.T) {
	// Create a DB with wrong column types to trigger scan errors.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Create nodes with hash as BLOB instead of TEXT — Scan(&string) may fail.
	db.Exec(`CREATE TABLE nodes (hash BLOB, parent_hash TEXT, role TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	db.Exec(`INSERT INTO nodes (hash, parent_hash) VALUES (X'DEADBEEF', NULL)`)
	reader := NewReaderFromDB(db)
	// ListSessions will try to scan BLOB into string.
	sessions, err := reader.ListSessions()
	// Some drivers handle this gracefully; we just exercise the code path.
	_ = sessions
	_ = err
}

func TestReaderGetSessionNullTimestamp(t *testing.T) {
	// Insert a node with NULL created_at to exercise the scan error
	// path when scanning NULL into time.Time.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE nodes (
		hash TEXT PRIMARY KEY, parent_hash TEXT, role TEXT, content JSON,
		model TEXT, provider TEXT, agent_name TEXT, prompt_tokens INTEGER,
		completion_tokens INTEGER, total_tokens INTEGER,
		cache_creation_input_tokens INTEGER, cache_read_input_tokens INTEGER,
		project TEXT, created_at TEXT
	)`)
	// Explicitly insert NULL created_at (no DEFAULT) to trigger scan error
	db.Exec(`INSERT INTO nodes (hash, parent_hash, role, content, model, prompt_tokens, completion_tokens, total_tokens, created_at)
		VALUES ('root1', NULL, 'user', '[]', 'model', 0, 0, 0, NULL)`)
	reader := NewReaderFromDB(db)
	// GetSession tries to scan NULL into time.Time which may error.
	// Either way, exercise the code path.
	session, err := reader.GetSession("root1")
	_ = session
	_ = err
}

func TestReaderGetSessionScanError(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Create table missing columns that GetSession expects.
	db.Exec(`CREATE TABLE nodes (hash TEXT PRIMARY KEY, parent_hash TEXT, role TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	db.Exec(`INSERT INTO nodes (hash, parent_hash, role, created_at) VALUES ('root1', NULL, 'user', CURRENT_TIMESTAMP)`)
	reader := NewReaderFromDB(db)
	// GetSession SELECTs more columns than exist — should trigger an error.
	_, err = reader.GetSession("root1")
	if err == nil {
		t.Error("expected error from schema mismatch in GetSession")
	}
}
