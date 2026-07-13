package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/google/uuid"
)

// ProfileType identifies connector kind.
type ProfileType string

const (
	TypeSlack    ProfileType = "slack"
	TypeWebhook  ProfileType = "webhook"
	TypeHTTPAuth ProfileType = "http_auth"
	TypeSMS      ProfileType = "sms"
	TypeMQTT     ProfileType = "mqtt"
	TypeKafka    ProfileType = "kafka"
)

// Profile is a named integration credential.
type Profile struct {
	ID        string            `json:"id"`
	Type      ProfileType       `json:"type"`
	Label     string            `json:"label"`
	Config    map[string]string `json:"config,omitempty"`
	Secret    string            `json:"secret,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
}

// ProfilePublic is returned from GET (secret redacted).
type ProfilePublic struct {
	ID        string            `json:"id"`
	Type      ProfileType       `json:"type"`
	Label     string            `json:"label"`
	Config    map[string]string `json:"config,omitempty"`
	SecretSet bool              `json:"secret_set"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
}

type storeFile struct {
	Profiles []Profile `json:"profiles"`
}

var mu sync.Mutex

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "connectors.json"), nil
}

func loadStore() (storeFile, error) {
	path, err := storePath()
	if err != nil {
		return storeFile{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storeFile{Profiles: []Profile{}}, nil
		}
		return storeFile{}, err
	}
	var s storeFile
	if err := json.Unmarshal(data, &s); err != nil {
		return storeFile{}, err
	}
	if s.Profiles == nil {
		s.Profiles = []Profile{}
	}
	for i := range s.Profiles {
		if s.Profiles[i].Secret != "" {
			if dec, err := config.DecryptSecret(s.Profiles[i].Secret); err == nil {
				s.Profiles[i].Secret = dec
			}
		}
	}
	return s, nil
}

func saveStore(s storeFile) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	copy := storeFile{Profiles: make([]Profile, len(s.Profiles))}
	for i, p := range s.Profiles {
		copy.Profiles[i] = p
		if copy.Profiles[i].Secret != "" {
			enc, err := config.EncryptSecret(copy.Profiles[i].Secret)
			if err != nil {
				return err
			}
			copy.Profiles[i].Secret = enc
		}
	}
	data, err := json.MarshalIndent(copy, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "connectors-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// List returns public profiles.
func List() ([]ProfilePublic, error) {
	mu.Lock()
	defer mu.Unlock()
	s, err := loadStore()
	if err != nil {
		return nil, err
	}
	out := make([]ProfilePublic, len(s.Profiles))
	for i, p := range s.Profiles {
		out[i] = toPublic(p)
	}
	return out, nil
}

// Get returns a profile with secret (internal use).
func Get(id string) (*Profile, error) {
	mu.Lock()
	defer mu.Unlock()
	s, err := loadStore()
	if err != nil {
		return nil, err
	}
	for i := range s.Profiles {
		if s.Profiles[i].ID == id {
			p := s.Profiles[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("connector %q not found", id)
}

// Create adds a profile.
func Create(p Profile) (*ProfilePublic, error) {
	mu.Lock()
	defer mu.Unlock()
	s, err := loadStore()
	if err != nil {
		return nil, err
	}
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	s.Profiles = append(s.Profiles, p)
	if err := saveStore(s); err != nil {
		return nil, err
	}
	pub := toPublic(p)
	return &pub, nil
}

// Update replaces a profile; empty secret keeps existing.
func Update(id string, p Profile) (*ProfilePublic, error) {
	mu.Lock()
	defer mu.Unlock()
	s, err := loadStore()
	if err != nil {
		return nil, err
	}
	for i := range s.Profiles {
		if s.Profiles[i].ID != id {
			continue
		}
		p.ID = id
		p.CreatedAt = s.Profiles[i].CreatedAt
		p.UpdatedAt = time.Now().UTC()
		if p.Secret == "" {
			p.Secret = s.Profiles[i].Secret
		}
		s.Profiles[i] = p
		if err := saveStore(s); err != nil {
			return nil, err
		}
		pub := toPublic(p)
		return &pub, nil
	}
	return nil, fmt.Errorf("connector %q not found", id)
}

// Delete removes a profile.
func Delete(id string) error {
	mu.Lock()
	defer mu.Unlock()
	s, err := loadStore()
	if err != nil {
		return err
	}
	var kept []Profile
	found := false
	for _, p := range s.Profiles {
		if p.ID == id {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return fmt.Errorf("connector %q not found", id)
	}
	s.Profiles = kept
	return saveStore(s)
}

func toPublic(p Profile) ProfilePublic {
	return ProfilePublic{
		ID: p.ID, Type: p.Type, Label: p.Label, Config: p.Config,
		SecretSet: p.Secret != "", CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ApplyToHTTPConfig merges connector credentials into action config headers.
func ApplyToHTTPConfig(cfg map[string]interface{}, profile *Profile) map[string]interface{} {
	if profile == nil || cfg == nil {
		return cfg
	}
	out := map[string]interface{}{}
	for k, v := range cfg {
		out[k] = v
	}
	switch profile.Type {
	case TypeHTTPAuth, TypeWebhook:
		headers, _ := out["headers"].(map[string]interface{})
		if headers == nil {
			headers = map[string]interface{}{}
		}
		if auth := profile.Config["header_name"]; auth != "" && profile.Secret != "" {
			headers[auth] = profile.Secret
		} else if profile.Secret != "" {
			headers["Authorization"] = profile.Secret
		}
		out["headers"] = headers
	}
	return out
}
