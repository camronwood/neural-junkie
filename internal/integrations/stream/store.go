package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type storeFile struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

// Store persists stream subscriptions under ~/.neural-junkie/stream/subscriptions.json.
type Store struct {
	mu       sync.RWMutex
	filePath string
	items    []Subscription
}

func subscriptionsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie", "stream")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "subscriptions.json"), nil
}

// NewStore loads subscriptions from disk.
func NewStore() (*Store, error) {
	p, err := subscriptionsPath()
	if err != nil {
		return nil, err
	}
	s := &Store{filePath: p, items: []Subscription{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload reads subscriptions from disk.
func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) loadLocked() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.items = nil
			return nil
		}
		return err
	}
	var f storeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	if f.Subscriptions == nil {
		f.Subscriptions = []Subscription{}
	}
	s.items = f.Subscriptions
	return nil
}

func (s *Store) saveLocked() error {
	f := storeFile{Subscriptions: s.items}
	if f.Subscriptions == nil {
		f.Subscriptions = []Subscription{}
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// List returns a copy of all subscriptions.
func (s *Store) List() []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Subscription, len(s.items))
	copy(out, s.items)
	return out
}

// Get returns a subscription by ID.
func (s *Store) Get(id string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.items {
		if s.items[i].ID == id {
			sub := s.items[i]
			return &sub, true
		}
	}
	return nil, false
}

// Upsert creates or updates a subscription.
func (s *Store) Upsert(sub Subscription) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateSubscription(sub); err != nil {
		return Subscription{}, err
	}
	now := time.Now().UTC()
	if sub.ID == "" {
		sub.ID = uuid.New().String()
		sub.CreatedAt = now
		sub.UpdatedAt = now
		s.items = append(s.items, sub)
		if err := s.saveLocked(); err != nil {
			return Subscription{}, err
		}
		return sub, nil
	}
	for i := range s.items {
		if s.items[i].ID != sub.ID {
			continue
		}
		sub.CreatedAt = s.items[i].CreatedAt
		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = now
		}
		sub.UpdatedAt = now
		s.items[i] = sub
		if err := s.saveLocked(); err != nil {
			return Subscription{}, err
		}
		return sub, nil
	}
	sub.CreatedAt = now
	sub.UpdatedAt = now
	s.items = append(s.items, sub)
	if err := s.saveLocked(); err != nil {
		return Subscription{}, err
	}
	return sub, nil
}

// Delete removes a subscription by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]Subscription, 0, len(s.items))
	found := false
	for _, item := range s.items {
		if item.ID == id {
			found = true
			continue
		}
		kept = append(kept, item)
	}
	if !found {
		return fmt.Errorf("subscription %q not found", id)
	}
	s.items = kept
	return s.saveLocked()
}

func validateSubscription(sub Subscription) error {
	if sub.Protocol != ProtocolMQTT && sub.Protocol != ProtocolKafka {
		return fmt.Errorf("protocol must be mqtt or kafka")
	}
	if sub.ConnectorID == "" {
		return fmt.Errorf("connector_id required")
	}
	if sub.Topic == "" {
		return fmt.Errorf("topic required")
	}
	switch sub.Action.Type {
	case ActionRunbook:
		if sub.Action.DefinitionID == "" {
			return fmt.Errorf("runbook action requires definition_id")
		}
		if len(sub.Action.AgentIDs) < 1 {
			return fmt.Errorf("runbook action requires agent_ids")
		}
	case ActionChannel:
		if sub.Action.HubChannel == "" {
			return fmt.Errorf("channel action requires hub_channel")
		}
	case ActionWebhook:
		if sub.Action.WebhookConnectorID == "" && sub.Action.URLOverride == "" {
			return fmt.Errorf("webhook action requires webhook_connector_id or url_override")
		}
	default:
		return fmt.Errorf("action.type must be runbook, channel, or webhook")
	}
	if sub.Match != nil && sub.Match.JSONPath != "" {
		op := sub.Match.Op
		if op == "" {
			op = MatchEquals
		}
		if op != MatchEquals && op != MatchContains {
			return fmt.Errorf("match.op must be equals or contains")
		}
	}
	return nil
}
