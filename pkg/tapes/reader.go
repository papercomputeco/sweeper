package tapes

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

type Node struct {
	Hash             string
	ParentHash       string
	Role             string
	Content          []ContentBlock
	Model            string
	PromptTokens     int
	CompletionTokens int
	CreatedAt        time.Time
}

type ContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolInput any    `json:"tool_input,omitempty"`
}

type Session struct {
	RootHash              string
	Nodes                 []Node
	TotalPromptTokens     int
	TotalCompletionTokens int
	StartTime             time.Time
	EndTime               time.Time
}

type Reader struct {
	db *sql.DB
}

func NewReader(dbPath string) (*Reader, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	return &Reader{db: db}, nil
}

func NewReaderFromDB(db *sql.DB) *Reader {
	return &Reader{db: db}
}

func (r *Reader) Close() error {
	return r.db.Close()
}

func (r *Reader) ListSessions() ([]string, error) {
	rows, err := r.db.Query(
		`SELECT hash FROM nodes WHERE parent_hash IS NULL ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		sessions = append(sessions, hash)
	}
	return sessions, rows.Err()
}

func (r *Reader) RecentSessions(n int) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT hash FROM nodes WHERE parent_hash IS NULL ORDER BY created_at DESC LIMIT ?`, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		sessions = append(sessions, hash)
	}
	return sessions, rows.Err()
}

func (r *Reader) GetSession(rootHash string) (Session, error) {
	rows, err := r.db.Query(`
		WITH RECURSIVE chain(h) AS (
			SELECT ?
			UNION ALL
			SELECT n.hash FROM nodes n JOIN chain ON n.parent_hash = chain.h
		)
		SELECT n.hash, COALESCE(n.parent_hash, ''), n.role, COALESCE(n.content, '[]'),
			   COALESCE(n.model, ''), COALESCE(n.prompt_tokens, 0), COALESCE(n.completion_tokens, 0),
			   n.created_at
		FROM chain JOIN nodes n ON n.hash = chain.h
		ORDER BY n.created_at
	`, rootHash)
	if err != nil {
		return Session{RootHash: rootHash}, err
	}
	defer rows.Close()

	var session Session
	session.RootHash = rootHash
	for rows.Next() {
		var node Node
		var contentJSON string
		if err := rows.Scan(&node.Hash, &node.ParentHash, &node.Role, &contentJSON,
			&node.Model, &node.PromptTokens, &node.CompletionTokens, &node.CreatedAt); err != nil {
			return session, err
		}
		json.Unmarshal([]byte(contentJSON), &node.Content) //nolint:errcheck
		session.Nodes = append(session.Nodes, node)
		session.TotalPromptTokens += node.PromptTokens
		session.TotalCompletionTokens += node.CompletionTokens
	}

	if len(session.Nodes) > 0 {
		session.StartTime = session.Nodes[0].CreatedAt
		session.EndTime = session.Nodes[len(session.Nodes)-1].CreatedAt
	}
	return session, rows.Err()
}
