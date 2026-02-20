package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CLIAgentRecord is a persisted record of a user-created CLI agent.
type CLIAgentRecord struct {
	Type    string `json:"type"`     // Registry key (e.g. "cursor", "gemini")
	Name    string `json:"name"`     // Display name
	WorkDir string `json:"work_dir"` // Working directory
	Created string `json:"created"`  // ISO timestamp
}

// CLIAgentStorage manages persistence of CLI agent records.
type CLIAgentStorage struct {
	path string
	mu   sync.Mutex
}

// NewCLIAgentStorage returns a storage backed by ~/.neural-junkie/cli-agents.json.
func NewCLIAgentStorage() (*CLIAgentStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &CLIAgentStorage{path: filepath.Join(dir, "cli-agents.json")}, nil
}

func (s *CLIAgentStorage) load() ([]CLIAgentRecord, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []CLIAgentRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *CLIAgentStorage) save(records []CLIAgentRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Save persists a CLI agent record, replacing any existing record with the same name.
func (s *CLIAgentStorage) Save(record CLIAgentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, _ := s.load()

	// Replace existing record with same name, or append
	found := false
	for i, r := range records {
		if r.Name == record.Name {
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

// ListWithMetadata returns CLI agent records formatted for the cached agents API.
func (s *CLIAgentStorage) ListWithMetadata() ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.load()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, r := range records {
		result = append(result, map[string]interface{}{
			"type":       "cli",
			"name":       r.Name,
			"path":       r.WorkDir,
			"last_used":  r.Created,
			"cache_size": 0,
			"metadata": map[string]interface{}{
				"cli_type": r.Type,
				"work_dir": r.WorkDir,
			},
		})
	}
	return result, nil
}

// SaveCLIAgent is a convenience function for saving from the command handler.
func SaveCLIAgent(cliType, name, workDir string) {
	storage, err := NewCLIAgentStorage()
	if err != nil {
		return
	}
	_ = storage.Save(CLIAgentRecord{
		Type:    cliType,
		Name:    name,
		WorkDir: workDir,
		Created: time.Now().UTC().Format(time.RFC3339),
	})
}
