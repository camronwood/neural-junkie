package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BaseDir returns ~/.neural-junkie/slack for tokens and bindings.
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".neural-junkie", "slack")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func bindingsPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bindings.json"), nil
}

func threadsPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "threads.json"), nil
}

// OAuthAppCredentials holds Slack app OAuth client credentials.
type OAuthAppCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func oauthAppPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "oauth_app.json"), nil
}

// LoadOAuthApp reads optional OAuth app credentials.
func LoadOAuthApp() (*OAuthAppCredentials, error) {
	p, err := oauthAppPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c OAuthAppCredentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// SaveOAuthApp persists OAuth app credentials.
func SaveOAuthApp(c *OAuthAppCredentials) error {
	if c == nil {
		return fmt.Errorf("nil credentials")
	}
	p, err := oauthAppPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

// PublicOAuthConfig is returned to the desktop (no secret).
type PublicOAuthConfig struct {
	ClientID    string `json:"client_id"`
	RedirectURL string `json:"redirect_url"`
	SecretSet   bool   `json:"secret_set"`
	Configured  bool   `json:"configured"`
}

// PublicOAuthFromDir builds API-safe OAuth config.
func PublicOAuthFromDir() PublicOAuthConfig {
	c, _ := LoadOAuthApp()
	if c == nil || c.ClientID == "" {
		return PublicOAuthConfig{}
	}
	return PublicOAuthConfig{
		ClientID:    c.ClientID,
		RedirectURL: c.RedirectURL,
		SecretSet:   c.ClientSecret != "",
		Configured:  true,
	}
}
