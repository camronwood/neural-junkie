package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// AgentCustomRulesFile is persisted rules keyed by agent ID.
type AgentCustomRulesFile struct {
	Rules map[string]string `json:"rules"`
}

// AgentCustomRulesStorage persists per-agent custom instructions (markdown).
type AgentCustomRulesStorage struct {
	path string
	mu   sync.Mutex
}

// NewAgentCustomRulesStorage returns storage at ~/.neural-junkie/agent-custom-rules.json.
func NewAgentCustomRulesStorage() (*AgentCustomRulesStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &AgentCustomRulesStorage{path: filepath.Join(dir, "agent-custom-rules.json")}, nil
}

func (s *AgentCustomRulesStorage) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var f AgentCustomRulesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Rules == nil {
		f.Rules = map[string]string{}
	}
	return f.Rules, nil
}

func (s *AgentCustomRulesStorage) save(rules map[string]string) error {
	f := AgentCustomRulesFile{Rules: rules}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Get returns stored markdown for an agent ID, if any.
func (s *AgentCustomRulesStorage) Get(agentID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := s.load()
	if err != nil {
		return "", false
	}
	v, ok := rules[agentID]
	return v, ok && v != ""
}

// Set replaces rules for one agent and persists the full map.
func (s *AgentCustomRulesStorage) Set(agentID, markdown string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, err := s.load()
	if err != nil {
		return err
	}
	if markdown == "" {
		delete(rules, agentID)
	} else {
		rules[agentID] = markdown
	}
	return s.save(rules)
}
