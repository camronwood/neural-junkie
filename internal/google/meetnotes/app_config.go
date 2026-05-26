package meetnotes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const appConfigFileName = "google_oauth_app.json"

// OAuthSource describes where Google OAuth app credentials were resolved from.
type OAuthSource string

const (
	OAuthSourceNone   OAuthSource = "none"
	OAuthSourceEnv    OAuthSource = "env"
	OAuthSourceVendor OAuthSource = "vendor"
	OAuthSourceConfig OAuthSource = "config"
)

// AppOAuthCredentials holds the Google Cloud OAuth client (app) credentials.
type AppOAuthCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url,omitempty"`
}

// LoadAppCredentials reads saved app credentials from the assistant data directory.
func LoadAppCredentials(baseDir string) (*AppOAuthCredentials, error) {
	if baseDir == "" {
		return nil, nil
	}
	path := filepath.Join(baseDir, appConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read google oauth app config: %w", err)
	}
	var creds AppOAuthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse google oauth app config: %w", err)
	}
	return &creds, nil
}

// SaveAppCredentials writes app credentials (mode 0600).
func SaveAppCredentials(baseDir string, creds *AppOAuthCredentials) error {
	if baseDir == "" {
		return fmt.Errorf("assistant base directory required")
	}
	if creds == nil || strings.TrimSpace(creds.ClientID) == "" || strings.TrimSpace(creds.ClientSecret) == "" {
		return fmt.Errorf("client_id and client_secret are required")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}
	creds.ClientID = strings.TrimSpace(creds.ClientID)
	creds.ClientSecret = strings.TrimSpace(creds.ClientSecret)
	creds.RedirectURL = strings.TrimSpace(creds.RedirectURL)
	if creds.RedirectURL == "" {
		creds.RedirectURL = defaultRedirectURL
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baseDir, appConfigFileName), data, 0600)
}

// ResolveAppCredentials returns credentials from env, bundled vendor config, or saved file.
func ResolveAppCredentials(baseDir string) (*AppOAuthCredentials, error) {
	creds, _, err := ResolveAppCredentialsWithSource(baseDir)
	return creds, err
}

// ResolveAppCredentialsWithSource returns credentials plus their source.
func ResolveAppCredentialsWithSource(baseDir string) (*AppOAuthCredentials, OAuthSource, error) {
	clientID := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GOOGLE_OAUTH_CLIENT_SECRET"))
	if clientID != "" && clientSecret != "" {
		redirectURL := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_GOOGLE_OAUTH_REDIRECT_URL"))
		if redirectURL == "" {
			redirectURL = defaultRedirectURL
		}
		return &AppOAuthCredentials{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
		}, OAuthSourceEnv, nil
	}
	if creds, ok := oauthFromBundledVendor(); ok {
		return creds, OAuthSourceVendor, nil
	}
	fileCreds, err := LoadAppCredentials(baseDir)
	if err != nil {
		return nil, OAuthSourceNone, err
	}
	if fileCreds == nil || fileCreds.ClientID == "" || fileCreds.ClientSecret == "" {
		return nil, OAuthSourceNone, fmt.Errorf("google oauth app credentials not configured")
	}
	if fileCreds.RedirectURL == "" {
		fileCreds.RedirectURL = defaultRedirectURL
	}
	return fileCreds, OAuthSourceConfig, nil
}

func (c *AppOAuthCredentials) oauth2Config() *oauth2.Config {
	redirectURL := c.RedirectURL
	if redirectURL == "" {
		redirectURL = defaultRedirectURL
	}
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}
}

// PublicAppConfig is safe to return to clients (no secret).
type PublicAppConfig struct {
	ClientID     string `json:"client_id"`
	RedirectURL  string `json:"redirect_url"`
	SecretSet    bool   `json:"secret_set"`
	Configured   bool   `json:"configured"`
	ConnectReady bool   `json:"connect_ready"`
	OAuthSource  string `json:"oauth_source,omitempty"`
}

// PublicAppConfigFromDir returns non-sensitive OAuth app settings for the UI.
func PublicAppConfigFromDir(baseDir string) PublicAppConfig {
	creds, src, err := ResolveAppCredentialsWithSource(baseDir)
	if err != nil || creds == nil {
		fileOnly, _ := LoadAppCredentials(baseDir)
		out := PublicAppConfig{RedirectURL: defaultRedirectURL, OAuthSource: string(OAuthSourceNone)}
		if fileOnly != nil {
			out.ClientID = fileOnly.ClientID
			if fileOnly.RedirectURL != "" {
				out.RedirectURL = fileOnly.RedirectURL
			}
			out.SecretSet = fileOnly.ClientSecret != ""
			out.Configured = fileOnly.ClientID != "" && fileOnly.ClientSecret != ""
			if out.Configured {
				out.OAuthSource = string(OAuthSourceConfig)
			}
		}
		return out
	}
	return PublicAppConfig{
		ClientID:     creds.ClientID,
		RedirectURL:  creds.RedirectURL,
		SecretSet:    creds.ClientSecret != "",
		Configured:   true,
		ConnectReady: true,
		OAuthSource:  string(src),
	}
}
