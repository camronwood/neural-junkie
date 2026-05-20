package collaboration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunbookTemplate is a reusable runbook definition stored on disk.
type RunbookTemplate struct {
	Name             string          `json:"name"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	ExecutionPolicy  ExecutionPolicy `json:"execution_policy,omitempty"`
	Tasks            []CollaborationTask `json:"tasks"`
}

// ListRunbookTemplates reads JSON templates from dir (*.json).
func ListRunbookTemplates(dir string) ([]RunbookTemplate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []RunbookTemplate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var t RunbookTemplate
		if err := json.Unmarshal(b, &t); err != nil {
			continue
		}
		if t.Name == "" {
			t.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, t)
	}
	return out, nil
}

// LoadRunbookTemplate loads one template by name.
func LoadRunbookTemplate(dir, name string) (*RunbookTemplate, error) {
	path := filepath.Join(dir, name+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", name, err)
	}
	var t RunbookTemplate
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	if t.Name == "" {
		t.Name = name
	}
	return &t, nil
}
