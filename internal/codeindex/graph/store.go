package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// Store persists nodes and edges under a code-graph directory.
type Store struct {
	mu sync.RWMutex
	db *sql.DB
	dir string
}

// Open opens or creates graph.sqlite in graphDir.
func Open(graphDir string) (*Store, error) {
	if err := os.MkdirAll(graphDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(graphDir, "graph.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, dir: graphDir}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  label TEXT NOT NULL,
  path TEXT,
  line INTEGER DEFAULT 0,
  language TEXT,
  community TEXT,
  degree INTEGER DEFAULT 0,
  symbol_kind TEXT,
  json TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS edges (
  id TEXT PRIMARY KEY,
  from_id TEXT NOT NULL,
  to_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  provenance TEXT NOT NULL,
  path TEXT,
  line INTEGER DEFAULT 0,
  json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_nodes_label ON nodes(label);
CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes(path);
CREATE INDEX IF NOT EXISTS idx_nodes_community ON nodes(community);
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`)
	return err
}

// ReplaceAll clears and writes a full graph snapshot.
func (s *Store) ReplaceAll(nodes []Node, edges []Edge, meta Meta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM edges`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM nodes`); err != nil {
		return err
	}

	for _, n := range nodes {
		raw, err := json.Marshal(n)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO nodes(id, kind, label, path, line, language, community, degree, symbol_kind, json)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
			n.ID, string(n.Kind), n.Label, n.Path, n.Line, n.Language, n.Community, n.Degree, n.SymbolKind, string(raw),
		); err != nil {
			return err
		}
	}
	for _, e := range edges {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO edges(id, from_id, to_id, kind, provenance, path, line, json)
VALUES(?,?,?,?,?,?,?,?)`,
			e.ID, e.From, e.To, string(e.Kind), string(e.Provenance), e.Path, e.Line, string(raw),
		); err != nil {
			return err
		}
	}

	mb, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO meta(key, value) VALUES('meta', ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(mb)); err != nil {
		return err
	}
	return tx.Commit()
}

// LoadMeta returns persisted meta or empty.
func (s *Store) LoadMeta() (Meta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var raw string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key='meta'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return Meta{}, nil
	}
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return Meta{}, err
	}
	return m, nil
}

// AllNodes returns every node.
func (s *Store) AllNodes() ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT json FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var n Node
		if err := json.Unmarshal([]byte(raw), &n); err != nil {
			continue
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// AllEdges returns every edge.
func (s *Store) AllEdges() ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT json FROM edges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e Edge
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetNode loads one node by id.
func (s *Store) GetNode(id string) (Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var raw string
	err := s.db.QueryRow(`SELECT json FROM nodes WHERE id=?`, id).Scan(&raw)
	if err != nil {
		return Node{}, false
	}
	var n Node
	if json.Unmarshal([]byte(raw), &n) != nil {
		return Node{}, false
	}
	return n, true
}

// FindNodesByLabel returns nodes whose label contains q (case-insensitive).
func (s *Store) FindNodesByLabel(q string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(
		`SELECT json FROM nodes WHERE lower(label) LIKE '%' || lower(?) || '%' OR lower(path) LIKE '%' || lower(?) || '%' LIMIT ?`,
		q, q, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var n Node
		if json.Unmarshal([]byte(raw), &n) != nil {
			continue
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// EdgesForNode returns edges touching a node.
func (s *Store) EdgesForNode(id string) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT json FROM edges WHERE from_id=? OR to_id=?`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e Edge
		if json.Unmarshal([]byte(raw), &e) != nil {
			continue
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Adjacency returns undirected neighbors for BFS path finding.
func (s *Store) Adjacency() (map[string][]string, map[string]Edge, error) {
	edges, err := s.AllEdges()
	if err != nil {
		return nil, nil, err
	}
	adj := make(map[string][]string)
	byPair := make(map[string]Edge)
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
		byPair[e.From+"→"+e.To] = e
		byPair[e.To+"→"+e.From] = e
	}
	return adj, byPair, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Dir returns the graph directory.
func (s *Store) Dir() string { return s.dir }

func edgeID(from, to string, kind EdgeKind, line int) string {
	return fmt.Sprintf("%s|%s|%s|%d", from, kind, to, line)
}
