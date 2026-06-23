package authstore

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Role is hub authorization role.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// APIKeyRecord is a persisted service account key.
type APIKeyRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
}

// Store persists API keys in ~/.neural-junkie/auth.db
type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".neural-junkie", "auth.db")
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
CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'member',
  key_hash TEXT NOT NULL UNIQUE,
  prefix TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0
);
`)
	return err
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CreateAPIKey returns the raw key once (nj_...).
func (s *Store) CreateAPIKey(name string, role Role) (raw string, rec *APIKeyRecord, err error) {
	if role == "" {
		role = RoleMember
	}
	id := newID()
	secret := newID()
	raw = "nj_" + secret
	prefix := raw[:11] + "..."
	rec = &APIKeyRecord{
		ID:        id,
		Name:      strings.TrimSpace(name),
		Role:      role,
		Prefix:    prefix,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.Exec(`INSERT INTO api_keys(id,name,role,key_hash,prefix,created_at,revoked) VALUES (?,?,?,?,?,?,0)`,
		rec.ID, rec.Name, string(rec.Role), hashKey(raw), prefix, rec.CreatedAt.UnixMilli())
	return raw, rec, err
}

func (s *Store) ListAPIKeys() ([]APIKeyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.Query(`SELECT id,name,role,prefix,created_at,revoked FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyRecord
	for rows.Next() {
		var rec APIKeyRecord
		var role string
		var created int64
		var revoked int
		if err := rows.Scan(&rec.ID, &rec.Name, &role, &rec.Prefix, &created, &revoked); err != nil {
			return nil, err
		}
		rec.Role = Role(role)
		rec.CreatedAt = time.UnixMilli(created)
		rec.Revoked = revoked != 0
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) RevokeAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE api_keys SET revoked=1 WHERE id=?`, id)
	return err
}

// ValidateAPIKey returns role and key id when valid.
func (s *Store) ValidateAPIKey(raw string) (Role, string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "nj_") {
		return "", "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var role, id string
	var revoked int
	err := s.db.QueryRow(`SELECT id, role, revoked FROM api_keys WHERE key_hash=?`, hashKey(raw)).
		Scan(&id, &role, &revoked)
	if err != nil || revoked != 0 {
		return "", "", false
	}
	return Role(role), id, true
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) Close() error {
	return s.db.Close()
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "auth.db"), nil
}

func NormalizeRole(r string) Role {
	switch Role(strings.ToLower(strings.TrimSpace(r))) {
	case RoleAdmin:
		return RoleAdmin
	case RoleViewer:
		return RoleViewer
	default:
		return RoleMember
	}
}

func (r Role) CanMutate() bool  { return r == RoleAdmin || r == RoleMember }
func (r Role) CanAdmin() bool   { return r == RoleAdmin }
func (r Role) String() string   { return string(r) }
func (r Role) Valid() bool      { return r == RoleAdmin || r == RoleMember || r == RoleViewer }

func ErrInvalidRole() error { return fmt.Errorf("invalid role") }
