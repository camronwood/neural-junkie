package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// VectorRecord is a persisted embedding row.
type VectorRecord struct {
	ID     string
	Vector []float64
}

// SQLite persists chunk vectors under a codeindex directory.
type SQLite struct {
	mu sync.RWMutex
	db *sql.DB
}

// Open opens or creates vectors.db in indexDir.
func Open(indexDir string) (*SQLite, error) {
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(indexDir, "vectors.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &SQLite{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS vectors (
  chunk_id TEXT PRIMARY KEY,
  vector_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
CREATE INDEX IF NOT EXISTS idx_vectors_updated ON vectors(updated_at);
`)
	return err
}

// Put stores or replaces a vector.
func (s *SQLite) Put(id string, vec []float64) error {
	raw, err := json.Marshal(vec)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.Exec(`INSERT INTO vectors(chunk_id, vector_json) VALUES(?, ?)
ON CONFLICT(chunk_id) DO UPDATE SET vector_json=excluded.vector_json, updated_at=strftime('%s','now')`, id, string(raw))
	return err
}

// Get loads a vector by chunk id.
func (s *SQLite) Get(id string) (VectorRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var raw string
	err := s.db.QueryRow(`SELECT vector_json FROM vectors WHERE chunk_id=?`, id).Scan(&raw)
	if err != nil {
		return VectorRecord{}, false
	}
	var vec []float64
	if json.Unmarshal([]byte(raw), &vec) != nil {
		return VectorRecord{}, false
	}
	return VectorRecord{ID: id, Vector: vec}, true
}

// DeleteMissing removes vectors whose chunk ids are not in keep.
func (s *SQLite) DeleteMissing(keep map[string]struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT chunk_id FROM vectors`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		if _, ok := keep[id]; !ok {
			stale = append(stale, id)
		}
	}
	for _, id := range stale {
		if _, err := s.db.Exec(`DELETE FROM vectors WHERE chunk_id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database.
func (s *SQLite) Close() error {
	return s.db.Close()
}

// Count returns stored vector rows.
func (s *SQLite) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&n)
	return n, err
}

// Stats returns human-readable index stats.
func (s *SQLite) Stats() (string, error) {
	n, err := s.Count()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d vectors in sqlite store", n), nil
}
