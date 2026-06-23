package checkpoint

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// State is a persisted agent-runtime checkpoint.
type State struct {
	ID        string
	Channel   string
	ThreadID  string
	Workspace string
	Step      int
	Payload   map[string]interface{}
	UpdatedAt time.Time
}

// Store persists agent runtime checkpoints in SQLite.
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

// DefaultPath returns ~/.neural-junkie/agent-checkpoints.db
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "agent-checkpoints.db"), nil
}

// Open opens the checkpoint database.
func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS agent_checkpoints (
  id TEXT PRIMARY KEY,
  channel TEXT NOT NULL,
  thread_id TEXT NOT NULL DEFAULT '',
  workspace TEXT NOT NULL DEFAULT '',
  step INTEGER NOT NULL DEFAULT 0,
  payload_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_ckpt_channel ON agent_checkpoints(channel, updated_at DESC);
`)
	return err
}

// Save upserts a checkpoint.
func (s *Store) Save(st State) error {
	raw, err := json.Marshal(st.Payload)
	if err != nil {
		return err
	}
	ts := st.UpdatedAt.Unix()
	if ts == 0 {
		ts = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.Exec(`INSERT INTO agent_checkpoints(id, channel, thread_id, workspace, step, payload_json, updated_at)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET step=excluded.step, payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
		st.ID, st.Channel, st.ThreadID, st.Workspace, st.Step, string(raw), ts)
	return err
}

// LoadLatest returns the newest checkpoint for a channel.
func (s *Store) LoadLatest(channel string) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var id, thread, workspace, payload string
	var step int
	var updated int64
	err := s.db.QueryRow(`SELECT id, thread_id, workspace, step, payload_json, updated_at
FROM agent_checkpoints WHERE channel=? ORDER BY updated_at DESC LIMIT 1`, channel).
		Scan(&id, &thread, &workspace, &step, &payload, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	_ = json.Unmarshal([]byte(payload), &m)
	return &State{
		ID:        id,
		Channel:   channel,
		ThreadID:  thread,
		Workspace: workspace,
		Step:      step,
		Payload:   m,
		UpdatedAt: time.Unix(updated, 0),
	}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}
