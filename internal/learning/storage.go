package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type storeFile struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Store persists learnings to learnings.json (v2).
type Store struct {
	mu   sync.RWMutex
	path string
	data storeFile
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
	s := &Store{path: path, data: storeFile{Version: StoreVersion}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = storeFile{Version: StoreVersion, Entries: nil}
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
	// v1: bare array or {entries:[]}
	var probe struct {
		Version *int    `json:"version"`
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		var legacy []Entry
		if err2 := json.Unmarshal(raw, &legacy); err2 != nil {
			return err
		}
		s.data.Entries = migrateEntries(legacy)
		s.data.Version = StoreVersion
		return s.saveLocked()
	}
	if probe.Version == nil && len(probe.Entries) == 0 {
		var legacy []Entry
		if err := json.Unmarshal(raw, &legacy); err == nil {
			s.data.Entries = migrateEntries(legacy)
			s.data.Version = StoreVersion
			return s.saveLocked()
		}
	}
	s.data.Entries = migrateEntries(probe.Entries)
	if probe.Version != nil {
		s.data.Version = *probe.Version
	} else {
		s.data.Version = StoreVersion
	}
	if s.data.Version < StoreVersion {
		s.data.Version = StoreVersion
		return s.saveLocked()
	}
	return nil
}

func migrateEntries(in []Entry) []Entry {
	out := make([]Entry, len(in))
	for i, e := range in {
		out[i] = e
		if out[i].Scope == "" {
			out[i].Scope = ScopeAgent
		}
		if out[i].ContentHash == "" && out[i].Content != "" {
			out[i].ContentHash = ContentHash(out[i].Content)
		}
	}
	return out
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	s.data.Version = StoreVersion
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func (s *Store) matchesUser(e Entry, userID string, includeLegacy bool) bool {
	if userID == "" {
		return true
	}
	if e.UserID == userID {
		return true
	}
	return includeLegacy && e.UserID == ""
}

// hasEntriesForUser reports whether any active learning is stored under userID.
func (s *Store) hasEntriesForUser(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.data.Entries {
		if e.Active && e.UserID == userID {
			return true
		}
	}
	return false
}

// entryMatchesAgent reports whether an agent-scoped learning applies to a runtime agent.
// Matches by ID first, then stable type+name (survives hub restarts that reissue UUIDs).
func entryMatchesAgent(e Entry, agentID, agentType, agentName string) bool {
	if e.Scope != ScopeAgent {
		return true
	}
	if agentID != "" && e.AgentID == agentID {
		return true
	}
	eType := strings.TrimSpace(strings.ToLower(e.AgentType))
	eName := strings.TrimSpace(strings.ToLower(e.AgentName))
	wantType := strings.TrimSpace(strings.ToLower(agentType))
	wantName := strings.TrimSpace(strings.ToLower(agentName))
	if wantType != "" && wantName != "" && eType == wantType && eName == wantName {
		return true
	}
	return false
}

// ResolveUserID maps a requested username slug to the stored user_id when they differ
// (e.g. login renamed from "Camron" to "camronwood" on a single-user hub).
func (s *Store) ResolveUserID(requested string) string {
	requested = SlugUserID(requested)
	if requested != "" && s.hasEntriesForUser(requested) {
		return requested
	}
	if sole := s.soleActiveUserID(); sole != "" {
		return sole
	}
	return requested
}

func (s *Store) soleActiveUserID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var found string
	for _, e := range s.data.Entries {
		if !e.Active {
			continue
		}
		uid := strings.TrimSpace(e.UserID)
		if uid == "" {
			continue
		}
		if found == "" {
			found = uid
			continue
		}
		if found != uid {
			return ""
		}
	}
	return found
}

func (s *Store) List(agentID string) []Entry {
	return s.ListFiltered(Filter{AgentID: agentID, IncludeLegacy: true})
}

func (s *Store) ListFiltered(f Filter) []Entry {
	if f.UserID != "" {
		f.UserID = s.ResolveUserID(f.UserID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.data.Entries {
		if !e.Active {
			continue
		}
		if f.Scope != "" && e.Scope != f.Scope {
			continue
		}
		if f.AgentID != "" || f.AgentType != "" || f.AgentName != "" {
			if e.Scope == ScopeAgent && !entryMatchesAgent(e, f.AgentID, f.AgentType, f.AgentName) {
				continue
			}
		}
		if f.CollaborationID != "" && e.CollaborationID != f.CollaborationID {
			continue
		}
		if !s.matchesUser(e, f.UserID, f.IncludeLegacy) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (s *Store) Get(id string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.data.Entries {
		if e.ID == id && e.Active {
			return e, true
		}
	}
	return Entry{}, false
}

func (s *Store) validateEntry(e *Entry) error {
	content := trim(e.Content)
	if content == "" {
		return fmt.Errorf("content is required")
	}
	e.Content = content
	e.Scope = NormalizeScope(e.Scope)
	switch e.Scope {
	case ScopeCollaboration:
		if strings.TrimSpace(e.CollaborationID) == "" {
			return fmt.Errorf("collaboration_id required for collaboration scope")
		}
	case ScopeAgent:
		if strings.TrimSpace(e.AgentID) == "" {
			return fmt.Errorf("agent_id is required for agent scope")
		}
	case ScopeGlobal:
		if strings.TrimSpace(e.AgentID) == "" {
			return fmt.Errorf("agent_id required as capture provenance")
		}
	}
	return nil
}

// Add saves a confirmed learning entry.
func (s *Store) Add(e Entry) (Entry, error) {
	if e.Category == "" {
		e.Category = CategoryPreference
	}
	if e.Scope == "" {
		e.Scope = ScopeAgent
	}
	if err := s.validateEntry(&e); err != nil {
		return Entry{}, err
	}
	now := time.Now().UTC()
	e.ID = uuid.NewString()
	e.Active = true
	e.ContentHash = ContentHash(e.Content)
	e.CreatedAt = now
	if e.ConfirmedAt.IsZero() {
		e.ConfirmedAt = now
	}
	e.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Entries = append(s.data.Entries, e)
	if err := s.saveLocked(); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Update edits an active entry.
func (s *Store) Update(id string, patch UpdatePatch) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Entries {
		if s.data.Entries[i].ID != id || !s.data.Entries[i].Active {
			continue
		}
		e := &s.data.Entries[i]
		if patch.Content != nil {
			e.Content = trim(*patch.Content)
			e.ContentHash = ContentHash(e.Content)
		}
		if patch.Category != nil {
			e.Category = *patch.Category
		}
		if patch.Scope != nil {
			e.Scope = NormalizeScope(*patch.Scope)
		}
		if patch.CollaborationID != nil {
			e.CollaborationID = strings.TrimSpace(*patch.CollaborationID)
		}
		if err := s.validateEntry(e); err != nil {
			return Entry{}, err
		}
		e.UpdatedAt = time.Now().UTC()
		if err := s.saveLocked(); err != nil {
			return Entry{}, err
		}
		return *e, nil
	}
	return Entry{}, fmt.Errorf("learning not found")
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

// FindSimilar returns active entry with same content hash for user.
func (s *Store) FindSimilar(userID, content string) (Entry, bool) {
	hash := ContentHash(content)
	for _, e := range s.ListFiltered(Filter{UserID: userID, IncludeLegacy: true}) {
		if e.ContentHash == hash {
			return e, true
		}
	}
	return Entry{}, false
}

// RecordUse increments use_count for injected ids.
func (s *Store) RecordUse(ids []string) {
	if len(ids) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	for i := range s.data.Entries {
		if _, ok := idSet[s.data.Entries[i].ID]; ok {
			s.data.Entries[i].UseCount++
			t := now
			s.data.Entries[i].LastUsedAt = &t
		}
	}
	_ = s.saveLocked()
}

func (s *Store) CountForAgent(agentID, agentType, agentName string) int {
	return len(s.ListFiltered(Filter{
		AgentID:       agentID,
		AgentType:     agentType,
		AgentName:     agentName,
		IncludeLegacy: true,
	}))
}

func (s *Store) CountByScope(userID string, scope Scope) int {
	return len(s.ListFiltered(Filter{UserID: userID, Scope: scope, IncludeLegacy: true}))
}

// ExportBundle returns all active entries for a user (including legacy).
func (s *Store) ExportBundle(userID string) []Entry {
	return s.ListFiltered(Filter{UserID: userID, IncludeLegacy: true})
}

// ImportMerge adds entries from bundle, skipping duplicate content_hash per user.
func (s *Store) ImportMerge(userID string, entries []Entry) (added int, skipped int) {
	for _, e := range entries {
		e.UserID = userID
		if _, dup := s.FindSimilar(userID, e.Content); dup {
			skipped++
			continue
		}
		if _, err := s.Add(e); err == nil {
			added++
		} else {
			skipped++
		}
	}
	return added, skipped
}

func trim(s string) string {
	return strings.TrimSpace(s)
}
