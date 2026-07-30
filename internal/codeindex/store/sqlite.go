package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// ChunkRecord is a persisted searchable source slice.
type ChunkRecord struct {
	ID      string
	Path    string
	Start   int
	End     int
	Content string
}

// VectorRecord is a persisted embedding row.
type VectorRecord struct {
	ID     string
	Vector []float64
}

// SQLite persists chunks and vectors under a codeindex directory (index.db).
type SQLite struct {
	mu         sync.RWMutex
	db         *sql.DB
	ftsEnabled bool
}

// Open opens or creates index.db in indexDir.
func Open(indexDir string) (*SQLite, error) {
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(indexDir, "index.db")
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

// Exists reports whether index.db is present under indexDir.
func Exists(indexDir string) bool {
	_, err := os.Stat(filepath.Join(indexDir, "index.db"))
	return err == nil
}

func (s *SQLite) migrate() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  content TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks(path);
CREATE TABLE IF NOT EXISTS vectors (
  chunk_id TEXT PRIMARY KEY,
  vector_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
CREATE INDEX IF NOT EXISTS idx_vectors_updated ON vectors(updated_at);
`); err != nil {
		return err
	}
	// FTS5 is optional; Search falls back to LIKE when unavailable.
	if _, err := s.db.Exec(`
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
  id UNINDEXED, path, content
);
`); err == nil {
		s.ftsEnabled = true
	}
	return nil
}

// ReplaceAllChunks replaces all chunk rows (and FTS) in one transaction.
func (s *SQLite) ReplaceAllChunks(chunks []ChunkRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM chunks`); err != nil {
		return err
	}
	if s.ftsEnabled {
		if _, err := tx.Exec(`DELETE FROM chunks_fts`); err != nil {
			return err
		}
	}
	ins, err := tx.Prepare(`INSERT INTO chunks(id, path, start_line, end_line, content) VALUES(?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer ins.Close()
	var ftsIns *sql.Stmt
	if s.ftsEnabled {
		ftsIns, err = tx.Prepare(`INSERT INTO chunks_fts(id, path, content) VALUES(?,?,?)`)
		if err != nil {
			return err
		}
		defer ftsIns.Close()
	}
	for _, ch := range chunks {
		if _, err := ins.Exec(ch.ID, ch.Path, ch.Start, ch.End, ch.Content); err != nil {
			return err
		}
		if ftsIns != nil {
			if _, err := ftsIns.Exec(ch.ID, ch.Path, ch.Content); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
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

// GetVectors loads vectors for the given chunk ids.
func (s *SQLite) GetVectors(ids []string) map[string][]float64 {
	out := make(map[string][]float64, len(ids))
	if len(ids) == 0 {
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range ids {
		var raw string
		if s.db.QueryRow(`SELECT vector_json FROM vectors WHERE chunk_id=?`, id).Scan(&raw) != nil {
			continue
		}
		var vec []float64
		if json.Unmarshal([]byte(raw), &vec) != nil {
			continue
		}
		out[id] = vec
	}
	return out
}

// GetChunks loads chunk rows by id (order follows ids).
func (s *SQLite) GetChunks(ids []string) []ChunkRecord {
	out := make([]ChunkRecord, 0, len(ids))
	if len(ids) == 0 {
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range ids {
		var ch ChunkRecord
		err := s.db.QueryRow(`SELECT id, path, start_line, end_line, content FROM chunks WHERE id=?`, id).
			Scan(&ch.ID, &ch.Path, &ch.Start, &ch.End, &ch.Content)
		if err != nil {
			continue
		}
		out = append(out, ch)
	}
	return out
}

// LexicalCandidates returns chunk ids matching query via FTS5 or LIKE fallback.
func (s *SQLite) LexicalCandidates(query string, limit int) []ChunkRecord {
	if limit <= 0 {
		limit = 80
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ftsEnabled {
		if match := ftsMatchQuery(query); match != "" {
			rows, err := s.db.Query(`
SELECT c.id, c.path, c.start_line, c.end_line, c.content
FROM chunks_fts f JOIN chunks c ON c.id = f.id
WHERE chunks_fts MATCH ?
ORDER BY bm25(chunks_fts, 0.0, 1.0, 1.0)
LIMIT ?`, match, limit)
			if err == nil {
				defer rows.Close()
				if out := scanChunkRows(rows); len(out) > 0 {
					return out
				}
			}
		}
	}
	like := "%" + escapeLike(strings.ToLower(strings.TrimSpace(query))) + "%"
	rows, err := s.db.Query(`
SELECT id, path, start_line, end_line, content FROM chunks
WHERE lower(path) LIKE ? ESCAPE '\' OR lower(content) LIKE ? ESCAPE '\'
LIMIT ?`, like, like, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanChunkRows(rows)
}

func scanChunkRows(rows *sql.Rows) []ChunkRecord {
	var out []ChunkRecord
	for rows.Next() {
		var ch ChunkRecord
		if rows.Scan(&ch.ID, &ch.Path, &ch.Start, &ch.End, &ch.Content) != nil {
			continue
		}
		out = append(out, ch)
	}
	return out
}

func ftsMatchQuery(query string) string {
	var terms []string
	for _, term := range strings.Fields(strings.ToLower(query)) {
		term = strings.Trim(term, " \t\r\n.,:;!?()[]{}<>\"'`")
		if len(term) < 2 {
			continue
		}
		term = strings.ReplaceAll(term, `"`, `""`)
		terms = append(terms, `"`+term+`"`)
		if len(terms) == 12 {
			break
		}
	}
	return strings.Join(terms, " OR ")
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// DeleteMissing removes vectors (and orphan chunk rows are replaced wholesale on rebuild).
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

// ChunkCount returns stored chunk rows.
func (s *SQLite) ChunkCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&n)
	return n, err
}

// Count returns stored vector rows.
func (s *SQLite) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&n)
	return n, err
}

// FTSEnabled reports whether FTS5 was created successfully.
func (s *SQLite) FTSEnabled() bool {
	return s.ftsEnabled
}

// Stats returns human-readable index stats.
func (s *SQLite) Stats() (string, error) {
	n, err := s.Count()
	if err != nil {
		return "", err
	}
	c, err := s.ChunkCount()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d chunks, %d vectors in sqlite store (fts=%v)", c, n, s.ftsEnabled), nil
}
