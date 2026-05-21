package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

// Binding maps a Slack channel to one primary Neural Junkie agent.
type Binding struct {
	ID              string             `json:"id,omitempty"`
	WorkspaceID     string             `json:"workspace_id"`
	SlackChannelID  string             `json:"slack_channel_id"`
	SlackChannelName string            `json:"slack_channel_name,omitempty"`
	NJChannel       string             `json:"nj_channel"`
	AgentID         string             `json:"agent_id"`
	AgentName       string             `json:"agent_name,omitempty"`
	Policy          config.SlackPolicy `json:"policy"`
	ReplyInThread   bool               `json:"reply_in_thread"`
	Enabled         bool               `json:"enabled"`
}

// NJChannelName returns the hub channel name for a Slack channel ID.
func NJChannelName(slackChannelID string) string {
	return "slack:" + slackChannelID
}

// BindingStore persists channel bindings.
type BindingStore struct {
	mu       sync.RWMutex
	filePath string
	items    []Binding
}

// NewBindingStore loads bindings from disk.
func NewBindingStore() (*BindingStore, error) {
	p, err := bindingsPath()
	if err != nil {
		return nil, err
	}
	s := &BindingStore{filePath: p, items: []Binding{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload reads bindings from disk.
func (s *BindingStore) Reload() error {
	return s.load()
}

func (s *BindingStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []Binding
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	s.items = items
	return nil
}

func (s *BindingStore) saveLocked() error {
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// List returns a copy of all bindings.
func (s *BindingStore) List() []Binding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Binding, len(s.items))
	copy(out, s.items)
	return out
}

// GetBySlackChannel returns the binding for a Slack channel ID.
func (s *BindingStore) GetBySlackChannel(slackChannelID string) (*Binding, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.items {
		if s.items[i].SlackChannelID == slackChannelID && s.items[i].Enabled {
			b := s.items[i]
			return &b, true
		}
	}
	return nil, false
}

// GetByNJChannel returns binding for a hub channel name.
func (s *BindingStore) GetByNJChannel(njChannel string) (*Binding, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.items {
		if s.items[i].NJChannel == njChannel && s.items[i].Enabled {
			b := s.items[i]
			return &b, true
		}
	}
	return nil, false
}

// Upsert creates or updates a binding.
func (s *BindingStore) Upsert(b Binding) (Binding, error) {
	if b.SlackChannelID == "" {
		return Binding{}, fmt.Errorf("slack_channel_id required")
	}
	if b.AgentID == "" {
		return Binding{}, fmt.Errorf("agent_id required")
	}
	if b.NJChannel == "" {
		b.NJChannel = NJChannelName(b.SlackChannelID)
	}
	if b.Policy == "" {
		b.Policy = config.SlackPolicyMentionOnly
	}
	if b.ID == "" {
		b.ID = b.SlackChannelID
	}
	b.ReplyInThread = true

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].SlackChannelID == b.SlackChannelID {
			keep := s.items[i].ID
			if keep != "" {
				b.ID = keep
			}
			s.items[i] = b
			return b, s.saveLocked()
		}
	}
	s.items = append(s.items, b)
	return b, s.saveLocked()
}

// Delete removes a binding by Slack channel ID.
func (s *BindingStore) Delete(slackChannelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	found := false
	for _, b := range s.items {
		if b.SlackChannelID == slackChannelID {
			found = true
			continue
		}
		out = append(out, b)
	}
	if !found {
		return fmt.Errorf("binding not found")
	}
	s.items = out
	return s.saveLocked()
}
