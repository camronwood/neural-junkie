package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ExpertAgentRecord is a persisted user-created expert (/create-expert or DM spawn).
type ExpertAgentRecord struct {
	AgentID         string   `json:"agent_id"`
	Name            string   `json:"name"`
	ExpertSlug      string   `json:"expert_slug"`
	Persona         string   `json:"persona,omitempty"`
	ProviderID      string   `json:"provider_id,omitempty"`
	ProviderName    string   `json:"provider_name,omitempty"`
	Model           string   `json:"model,omitempty"`
	CreatedBy       string   `json:"created_by"`
	DMChannel       string   `json:"dm_channel,omitempty"`
	Created         string   `json:"created"`
	CapabilityAllow []string `json:"capability_allow,omitempty"`
	CapabilityDeny  []string `json:"capability_deny,omitempty"`
}

// ExpertAgentStorage manages persistence of custom expert agents.
type ExpertAgentStorage struct {
	path string
	mu   sync.Mutex
}

// NewExpertAgentStorage returns storage backed by ~/.neural-junkie/expert-agents.json.
func NewExpertAgentStorage() (*ExpertAgentStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &ExpertAgentStorage{path: filepath.Join(dir, "expert-agents.json")}, nil
}

func (s *ExpertAgentStorage) load() ([]ExpertAgentRecord, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []ExpertAgentRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *ExpertAgentStorage) save(records []ExpertAgentRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Save persists an expert record, replacing any existing record with the same name.
func (s *ExpertAgentStorage) Save(record ExpertAgentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, _ := s.load()
	found := false
	for i, r := range records {
		if strings.EqualFold(r.Name, record.Name) {
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		records = append(records, record)
	}
	return s.save(records)
}

// List returns all persisted expert records.
func (s *ExpertAgentStorage) List() ([]ExpertAgentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// ListWithMetadata returns expert records formatted for the cached agents API.
func (s *ExpertAgentStorage) ListWithMetadata() ([]map[string]interface{}, error) {
	records, err := s.List()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		result = append(result, map[string]interface{}{
			"type":       "expert",
			"name":       r.Name,
			"path":       r.DMChannel,
			"last_used":  r.Created,
			"cache_size": 0,
			"metadata": map[string]interface{}{
				"expert_slug":   r.ExpertSlug,
				"provider_id":   r.ProviderID,
				"provider_name": r.ProviderName,
				"model":         r.Model,
				"dm_channel":    r.DMChannel,
				"created_by":    r.CreatedBy,
			},
		})
	}
	return result, nil
}

// DeleteByName removes a persisted expert record by display name.
func (s *ExpertAgentStorage) DeleteByName(name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.load()
	if err != nil {
		return false, err
	}
	var kept []ExpertAgentRecord
	found := false
	for _, r := range records {
		if strings.EqualFold(r.Name, name) {
			found = true
			continue
		}
		kept = append(kept, r)
	}
	if !found {
		return false, nil
	}
	return true, s.save(kept)
}

// SaveExpertAgent persists a user-created expert for startup restore.
func SaveExpertAgent(record ExpertAgentRecord) {
	storage, err := NewExpertAgentStorage()
	if err != nil {
		return
	}
	if record.Created == "" {
		record.Created = time.Now().UTC().Format(time.RFC3339)
	}
	_ = storage.Save(record)
}

// DeleteExpertAgent removes a persisted expert by name.
func DeleteExpertAgent(name string) {
	storage, err := NewExpertAgentStorage()
	if err != nil {
		return
	}
	_, _ = storage.DeleteByName(name)
}
