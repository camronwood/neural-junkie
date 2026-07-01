package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const StoreVersion = 1

// Status is adapter lifecycle state within the registry.
type Status string

const (
	StatusActive     Status = "active"
	StatusSuperseded Status = "superseded"
)

// Entry is one composed LoRA adapter version.
type Entry struct {
	ID            string    `json:"id"`
	OllamaTag     string    `json:"ollama_tag"`
	Version       int       `json:"version"`
	BaseOllamaTag string    `json:"base_ollama_tag"`
	AgentID       string    `json:"agent_id,omitempty"`
	Source        string    `json:"source,omitempty"`
	SourceID      string    `json:"source_id,omitempty"`
	JobID         string    `json:"job_id,omitempty"`
	RowCount      int       `json:"row_count,omitempty"`
	DatasetHash   string    `json:"dataset_hash,omitempty"`
	ExportedAt    time.Time `json:"exported_at"`
	Status        Status    `json:"status"`
	ArtifactDir   string    `json:"artifact_dir,omitempty"`
	EvalScore     float64   `json:"eval_score,omitempty"`
}

type storeFile struct {
	Version  int     `json:"version"`
	Adapters []Entry `json:"adapters"`
}

// RegisterInput describes a new adapter after successful training.
type RegisterInput struct {
	OllamaTag     string
	BaseOllamaTag string
	AgentID       string
	Source        string
	SourceID      string
	JobID         string
	RowCount      int
	DatasetHash   string
	ArtifactDir   string
	EvalScore     float64
}

// Store persists LoRA adapter versions.
type Store struct {
	mu   sync.RWMutex
	path string
	data storeFile
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "lora-adapters.json"
	}
	return filepath.Join(home, ".neural-junkie", "lora-adapters.json")
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
	s.data = storeFile{Version: StoreVersion, Adapters: nil}
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
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	s.data.Version = StoreVersion
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// Register adds a new adapter version and supersedes prior active entries for the same tag.
func (s *Store) Register(in RegisterInput) (*Entry, error) {
	tag := strings.TrimSpace(in.OllamaTag)
	if tag == "" {
		return nil, fmt.Errorf("ollama_tag is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	version := 1
	for _, e := range s.data.Adapters {
		if strings.EqualFold(e.OllamaTag, tag) && e.Version >= version {
			version = e.Version + 1
		}
	}
	for i := range s.data.Adapters {
		if strings.EqualFold(s.data.Adapters[i].OllamaTag, tag) && s.data.Adapters[i].Status == StatusActive {
			s.data.Adapters[i].Status = StatusSuperseded
		}
	}
	entry := Entry{
		ID:            uuid.NewString(),
		OllamaTag:     tag,
		Version:       version,
		BaseOllamaTag: strings.TrimSpace(in.BaseOllamaTag),
		AgentID:       strings.TrimSpace(in.AgentID),
		Source:        strings.TrimSpace(in.Source),
		SourceID:      strings.TrimSpace(in.SourceID),
		JobID:         strings.TrimSpace(in.JobID),
		RowCount:      in.RowCount,
		DatasetHash:   strings.TrimSpace(in.DatasetHash),
		ExportedAt:    time.Now().UTC(),
		Status:        StatusActive,
		ArtifactDir:   strings.TrimSpace(in.ArtifactDir),
		EvalScore:     in.EvalScore,
	}
	s.data.Adapters = append(s.data.Adapters, entry)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	out := entry
	return &out, nil
}

// Get returns an entry by id.
func (s *Store) Get(id string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.data.Adapters {
		if e.ID == id {
			out := e
			return &out, true
		}
	}
	return nil, false
}

// List returns adapters sorted by exported_at descending.
func (s *Store) List(agentID, ollamaTag string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	agentID = strings.TrimSpace(agentID)
	tag := strings.TrimSpace(ollamaTag)
	for _, e := range s.data.Adapters {
		if agentID != "" && e.AgentID != agentID {
			continue
		}
		if tag != "" && !strings.EqualFold(e.OllamaTag, tag) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExportedAt.After(out[j].ExportedAt)
	})
	return out
}

// ActiveForAgent returns the active adapter for an agent.
func (s *Store) ActiveForAgent(agentID string) (*Entry, bool) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *Entry
	for i := range s.data.Adapters {
		e := &s.data.Adapters[i]
		if e.AgentID != agentID || e.Status != StatusActive {
			continue
		}
		if best == nil || e.Version > best.Version {
			best = e
		}
	}
	if best == nil {
		return nil, false
	}
	out := *best
	return &out, true
}

// ActiveForTag returns the active adapter for an Ollama tag.
func (s *Store) ActiveForTag(ollamaTag string) (*Entry, bool) {
	tag := strings.TrimSpace(ollamaTag)
	if tag == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Adapters {
		e := &s.data.Adapters[i]
		if strings.EqualFold(e.OllamaTag, tag) && e.Status == StatusActive {
			out := *e
			return &out, true
		}
	}
	return nil, false
}

// Activate marks an entry active and supersedes other versions of the same tag.
func (s *Store) Activate(id string) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, e := range s.data.Adapters {
		if e.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("adapter not found")
	}
	tag := s.data.Adapters[idx].OllamaTag
	for i := range s.data.Adapters {
		if strings.EqualFold(s.data.Adapters[i].OllamaTag, tag) {
			if s.data.Adapters[i].ID == id {
				s.data.Adapters[i].Status = StatusActive
			} else if s.data.Adapters[i].Status == StatusActive {
				s.data.Adapters[i].Status = StatusSuperseded
			}
		}
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	out := s.data.Adapters[idx]
	return &out, nil
}

// SetEvalScore updates eval score on an entry.
func (s *Store) SetEvalScore(id string, score float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Adapters {
		if s.data.Adapters[i].ID == id {
			s.data.Adapters[i].EvalScore = score
			return s.saveLocked()
		}
	}
	return fmt.Errorf("adapter not found")
}

// DatasetHashFile computes sha256 of a JSONL dataset file.
func DatasetHashFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:8]), nil
}
