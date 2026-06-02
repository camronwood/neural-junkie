package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

type userTokenFile struct {
	AccessToken string `json:"access_token,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

// UserTokenStore persists the owner's Slack user OAuth token (encrypted).
type UserTokenStore struct {
	mu       sync.RWMutex
	filePath string
	data     userTokenFile
}

func userTokenPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return dir + "/user_token.json", nil
}

// NewUserTokenStore loads the encrypted user token from disk.
func NewUserTokenStore() (*UserTokenStore, error) {
	p, err := userTokenPath()
	if err != nil {
		return nil, err
	}
	s := &UserTokenStore{filePath: p}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload reads the encrypted user token from disk into memory.
func (s *UserTokenStore) Reload() error {
	return s.load()
}

func (s *UserTokenStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw userTokenFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.mu.Lock()
	s.data = raw
	s.mu.Unlock()
	return nil
}

func (s *UserTokenStore) saveLocked() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// HasToken reports whether a user token is stored.
func (s *UserTokenStore) HasToken() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.data.AccessToken) != ""
}

// SaveToken encrypts and persists the user OAuth token.
func (s *UserTokenStore) SaveToken(token, userID, scope string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("user token required")
	}
	enc, err := config.EncryptSecret(token)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.data = userTokenFile{
		AccessToken: enc,
		UserID:      strings.TrimSpace(userID),
		Scope:       strings.TrimSpace(scope),
	}
	err = s.saveLocked()
	s.mu.Unlock()
	return err
}

// AccessToken returns the decrypted user token, or empty if unset.
func (s *UserTokenStore) AccessToken() (string, error) {
	s.mu.RLock()
	enc := s.data.AccessToken
	s.mu.RUnlock()
	if strings.TrimSpace(enc) == "" {
		return "", nil
	}
	return config.DecryptSecret(enc)
}

// Clear removes the stored user token.
func (s *UserTokenStore) Clear() error {
	s.mu.Lock()
	s.data = userTokenFile{}
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	_ = os.Remove(s.filePath)
	return err
}
