package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store persists agent-scoped learnings to learnings.json.
type Store struct {
	mu   sync.RWMutex
	path string
	data struct {
		Entries []Entry `json:"entries"`
	}
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "learnings.json"
	}
	return filepath.Join(home, ".neural-junkie", "learnings.json")
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Entries = nil
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

// List returns active learnings, optionally filtered by agent_id.
func (s *Store) List(agentID string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.data.Entries {
		if !e.Active {
			continue
		}
		if agentID != "" && e.AgentID != agentID {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Add saves a confirmed learning entry.
func (s *Store) Add(e Entry) (Entry, error) {
	if e.AgentID == "" {
		return Entry{}, fmt.Errorf("agent_id is required")
	}
	content := trim(e.Content)
	if content == "" {
		return Entry{}, fmt.Errorf("content is required")
	}
	if e.Category == "" {
		e.Category = CategoryPreference
	}
	now := time.Now().UTC()
	e.ID = uuid.NewString()
	e.Content = content
	e.Active = true
	e.CreatedAt = now
	if e.ConfirmedAt.IsZero() {
		e.ConfirmedAt = now
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Entries = append(s.data.Entries, e)
	if err := s.saveLocked(); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Forget soft-deletes a learning by id.
func (s *Store) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.data.Entries {
		if s.data.Entries[i].ID == id {
			s.data.Entries[i].Active = false
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("learning not found")
	}
	return s.saveLocked()
}

// CountForAgent returns active learning count for an agent.
func (s *Store) CountForAgent(agentID string) int {
	return len(s.List(agentID))
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
