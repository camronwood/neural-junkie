package meetnotes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

const (
	defaultRedirectURL = "http://localhost:18765/api/assistant/google/callback"
	tokenFileName      = "google_oauth.json"
)

var scopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/drive.readonly",
}

// TokenStore persists OAuth tokens under the assistant data directory.
type TokenStore struct {
	BaseDir string
}

// OAuthConfig builds oauth2 config from env (override) or saved app credentials.
func OAuthConfig(baseDir string) (*oauth2.Config, error) {
	creds, err := ResolveAppCredentials(baseDir)
	if err != nil {
		return nil, err
	}
	return creds.oauth2Config(), nil
}

// OAuthConfigured reports whether OAuth app credentials are available.
func OAuthConfigured(baseDir string) bool {
	_, err := OAuthConfig(baseDir)
	return err == nil
}

func (s *TokenStore) tokenPath() string {
	return filepath.Join(s.BaseDir, tokenFileName)
}

// LoadToken reads a saved token or returns nil if not connected.
func (s *TokenStore) LoadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(s.tokenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read google oauth token: %w", err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse google oauth token: %w", err)
	}
	return &tok, nil
}

// SaveToken writes the token to disk.
func (s *TokenStore) SaveToken(tok *oauth2.Token) error {
	if err := os.MkdirAll(s.BaseDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.tokenPath(), data, 0600)
}

// ClearToken removes saved credentials and sync state.
func (s *TokenStore) ClearToken() error {
	_ = os.Remove(s.tokenPath())
	state := &StateStore{BaseDir: s.BaseDir}
	return state.Clear()
}

// HasValidToken returns true when a refresh or access token is stored.
func (s *TokenStore) HasValidToken() bool {
	tok, err := s.LoadToken()
	if err != nil || tok == nil {
		return false
	}
	return tok.AccessToken != "" || tok.RefreshToken != ""
}

// AuthCodeURL returns the Google consent URL for the given state nonce.
func AuthCodeURL(baseDir, state string) (string, error) {
	cfg, err := OAuthConfig(baseDir)
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce), nil
}

// ExchangeCode exchanges an authorization code for tokens and saves them.
func (s *TokenStore) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	cfg, err := OAuthConfig(s.BaseDir)
	if err != nil {
		return nil, err
	}
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange oauth code: %w", err)
	}
	if err := s.SaveToken(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// ClientSource returns a token source that auto-refreshes and persists updates.
func (s *TokenStore) ClientSource(ctx context.Context) (oauth2.TokenSource, error) {
	cfg, err := OAuthConfig(s.BaseDir)
	if err != nil {
		return nil, err
	}
	tok, err := s.LoadToken()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, fmt.Errorf("google account not connected")
	}
	base := cfg.TokenSource(ctx, tok)
	return &persistingTokenSource{store: s, base: oauth2.ReuseTokenSource(tok, base)}, nil
}

type persistingTokenSource struct {
	store *TokenStore
	base  oauth2.TokenSource
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.base.Token()
	if err != nil {
		return nil, err
	}
	if tok != nil && tok.AccessToken != "" {
		_ = p.store.SaveToken(tok)
	}
	return tok, nil
}

// ConnectedEmail returns the Gmail profile address when connected.
func (s *TokenStore) ConnectedEmail(ctx context.Context) (string, error) {
	ts, err := s.ClientSource(ctx)
	if err != nil {
		return "", err
	}
	svc, err := newGmailService(ctx, ts)
	if err != nil {
		return "", err
	}
	prof, err := svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return prof.EmailAddress, nil
}

// TokenExpiry returns token expiry if known.
func (s *TokenStore) TokenExpiry() *time.Time {
	tok, err := s.LoadToken()
	if err != nil || tok == nil || tok.Expiry.IsZero() {
		return nil
	}
	t := tok.Expiry
	return &t
}
