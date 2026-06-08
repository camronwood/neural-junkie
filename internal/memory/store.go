package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists memory chunks in SQLite.
type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

// DefaultDBPath returns ~/.neural-junkie/memory.db
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "memory.db"), nil
}

// Open creates or opens the memory database.
func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultDBPath()
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
CREATE TABLE IF NOT EXISTS memory_chunks (
  id TEXT PRIMARY KEY,
  source_type TEXT NOT NULL,
  source_id TEXT NOT NULL,
  channel TEXT NOT NULL DEFAULT '',
  thread_id TEXT NOT NULL DEFAULT '',
  collaboration_id TEXT NOT NULL DEFAULT '',
  rel_path TEXT NOT NULL DEFAULT '',
  sender_name TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  embedding_model TEXT NOT NULL DEFAULT '',
  vector_json TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memory_channel ON memory_chunks(channel, created_at);
CREATE INDEX IF NOT EXISTS idx_memory_collab ON memory_chunks(collaboration_id, created_at);
CREATE INDEX IF NOT EXISTS idx_memory_source ON memory_chunks(source_type, source_id);
`)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertChunk inserts or replaces a chunk row.
func (s *Store) UpsertChunk(ch Chunk) error {
	if ch.ID == "" {
		return fmt.Errorf("chunk id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	vecJSON, err := json.Marshal(ch.Vector)
	if err != nil {
		return err
	}
	created := ch.CreatedAt.UnixMilli()
	if ch.CreatedAt.IsZero() {
		created = time.Now().UTC().UnixMilli()
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO memory_chunks
(id, source_type, source_id, channel, thread_id, collaboration_id, rel_path, sender_name,
 content, content_hash, embedding_model, vector_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ch.ID, string(ch.SourceType), ch.SourceID, ch.Channel, ch.ThreadID, ch.CollaborationID,
		ch.RelPath, ch.SenderName, ch.Content, ch.ContentHash, ch.EmbeddingModel, string(vecJSON), created)
	return err
}

// DeleteBySource removes all chunks for a source id (message id or file path).
func (s *Store) DeleteBySource(sourceID string) error {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM memory_chunks WHERE source_id = ?`, sourceID)
	return err
}

// DeleteByChannel removes all chunks for a channel.
func (s *Store) DeleteByChannel(channel string) error {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM memory_chunks WHERE channel = ?`, channel)
	return err
}

// ListCandidates returns chunks scoped to channel and/or collaboration for search.
func (s *Store) ListCandidates(channel, collaborationID string, limit int) ([]Chunk, error) {
	channel = strings.TrimSpace(channel)
	collaborationID = strings.TrimSpace(collaborationID)
	if limit <= 0 {
		limit = DefaultSearchPrefilter
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []any
	switch {
	case channel != "" && collaborationID != "":
		query = `SELECT id, source_type, source_id, channel, thread_id, collaboration_id, rel_path,
sender_name, content, content_hash, embedding_model, vector_json, created_at
FROM memory_chunks
WHERE channel = ? OR (collaboration_id = ? AND source_type = ?)
ORDER BY created_at DESC LIMIT ?`
		args = []any{channel, collaborationID, string(SourceCollabArtifact), limit}
	case channel != "":
		query = `SELECT id, source_type, source_id, channel, thread_id, collaboration_id, rel_path,
sender_name, content, content_hash, embedding_model, vector_json, created_at
FROM memory_chunks WHERE channel = ? ORDER BY created_at DESC LIMIT ?`
		args = []any{channel, limit}
	case collaborationID != "":
		query = `SELECT id, source_type, source_id, channel, thread_id, collaboration_id, rel_path,
sender_name, content, content_hash, embedding_model, vector_json, created_at
FROM memory_chunks WHERE collaboration_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []any{collaborationID, limit}
	default:
		return nil, nil
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChunks(rows)
}

func scanChunks(rows *sql.Rows) ([]Chunk, error) {
	var out []Chunk
	for rows.Next() {
		var ch Chunk
		var sourceType, vecJSON string
		var created int64
		if err := rows.Scan(
			&ch.ID, &sourceType, &ch.SourceID, &ch.Channel, &ch.ThreadID, &ch.CollaborationID,
			&ch.RelPath, &ch.SenderName, &ch.Content, &ch.ContentHash, &ch.EmbeddingModel,
			&vecJSON, &created,
		); err != nil {
			return nil, err
		}
		ch.SourceType = SourceType(sourceType)
		ch.CreatedAt = time.UnixMilli(created)
		if vecJSON != "" {
			_ = json.Unmarshal([]byte(vecJSON), &ch.Vector)
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

// Stats returns aggregate index statistics.
func (s *Store) Stats() (Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var st Stats
	st.BySourceType = map[string]int{}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM memory_chunks`).Scan(&st.TotalChunks); err != nil {
		return st, err
	}
	rows, err := s.db.Query(`SELECT source_type, COUNT(*) FROM memory_chunks GROUP BY source_type`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return st, err
		}
		st.BySourceType[typ] = n
	}
	_ = s.db.QueryRow(`SELECT COUNT(DISTINCT channel) FROM memory_chunks WHERE channel != ''`).Scan(&st.ChannelsIndexed)
	_ = s.db.QueryRow(`SELECT COUNT(DISTINCT collaboration_id) FROM memory_chunks WHERE collaboration_id != ''`).Scan(&st.CollabsIndexed)
	return st, nil
}

// HasSource reports whether any chunk exists for source_id.
func (s *Store) HasSource(sourceID string) (bool, error) {
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM memory_chunks WHERE source_id = ?`, sourceID).Scan(&n)
	return n > 0, err
}
