// Package remotetokens persists nj-remote sidecar bearer tokens per workspace.
package remotetokens

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type store struct {
	path string
	mu   sync.Mutex
	data map[string]string
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "remote-tokens.json"), nil
}

func load() (*store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	s := &store{path: path, data: make(map[string]string)}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	_ = json.Unmarshal(raw, &s.data)
	if s.data == nil {
		s.data = make(map[string]string)
	}
	return s, nil
}

// Save stores sidecar bearer token for a workspace ID.
func Save(workspaceID, token string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == "" {
		delete(s.data, workspaceID)
	} else {
		s.data[workspaceID] = token
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

// Get returns stored sidecar token.
func Get(workspaceID string) (string, error) {
	s, err := load()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[workspaceID], nil
}

// Delete removes token for workspace.
func Delete(workspaceID string) error {
	return Save(workspaceID, "")
}

// List returns workspace ID → token map copy.
func List() (map[string]string, error) {
	s, err := load()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out, nil
}

// Path returns store file path for diagnostics.
func Path() (string, error) {
	return storePath()
}
