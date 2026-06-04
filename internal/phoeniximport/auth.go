package phoeniximport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type storedCredentials struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken *string    `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	TokenType    *string    `json:"token_type,omitempty"`
}

func (c storedCredentials) isExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().Add(60 * time.Second).After(c.ExpiresAt.UTC())
}

func defaultCredentialsPath(env string) string {
	env = normalizeEnvironment(env)
	name := fmt.Sprintf("credentials-%s.json", env)
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "com.BrightestBio.bbio", name)
	case "windows":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Roaming", "com.BrightestBio.bbio", "config", name)
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "com.BrightestBio.bbio", name)
	}
}

func resolveCredentialsPath(settings Settings) string {
	if p := strings.TrimSpace(settings.CredentialsPath); p != "" {
		return expandHome(p)
	}
	return defaultCredentialsPath(settings.EnvironmentOrDefault())
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func loadStoredCredentials(path string) (storedCredentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return storedCredentials{}, fmt.Errorf("read credentials %s: %w", path, err)
	}
	var creds storedCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return storedCredentials{}, fmt.Errorf("parse credentials %s: %w", path, err)
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return storedCredentials{}, fmt.Errorf("credentials file missing access_token")
	}
	return creds, nil
}

func saveStoredCredentials(path string, creds storedCredentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func clientIDForEnvironment(settings Settings, profile environmentProfile) (string, error) {
	if p := strings.TrimSpace(settings.AuthConfigPath); p != "" {
		cfg, err := ParseAuthConfigFile(expandHome(p))
		if err != nil {
			return "", err
		}
		if id := strings.TrimSpace(cfg.ClientID); id != "" {
			return id, nil
		}
	}
	envKey := strings.ToUpper(normalizeEnvironment(settings.EnvironmentOrDefault()))
	if id := strings.TrimSpace(os.Getenv("BBIO_" + envKey + "_CLIENT_ID")); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("Auth0 client_id not configured; set phoenix_auth_config_path or BBIO_%s_CLIENT_ID", envKey)
}

func refreshAccessToken(ctx context.Context, profile environmentProfile, clientID string, creds storedCredentials) (storedCredentials, error) {
	if creds.RefreshToken == nil || strings.TrimSpace(*creds.RefreshToken) == "" {
		return storedCredentials{}, fmt.Errorf("no refresh_token; authenticate with bbio login or device flow")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", strings.TrimSpace(*creds.RefreshToken))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+profile.Auth0Domain+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return storedCredentials{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return storedCredentials{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return storedCredentials{}, fmt.Errorf("token refresh HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var tok struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken *string `json:"refresh_token"`
		ExpiresIn    *int    `json:"expires_in"`
		TokenType    *string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return storedCredentials{}, err
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		return storedCredentials{}, fmt.Errorf("refresh response missing access_token")
	}
	out := storedCredentials{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
	}
	if out.RefreshToken == nil {
		out.RefreshToken = creds.RefreshToken
	}
	if tok.ExpiresIn != nil && *tok.ExpiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(*tok.ExpiresIn) * time.Second)
		out.ExpiresAt = &exp
	}
	return out, nil
}

func ensureCredentials(ctx context.Context, settings Settings) (storedCredentials, string, error) {
	path := resolveCredentialsPath(settings)
	creds, err := loadStoredCredentials(path)
	if err != nil {
		return storedCredentials{}, path, err
	}
	if !creds.isExpired() {
		return creds, path, nil
	}
	profile, err := profileForEnvironment(settings.EnvironmentOrDefault())
	if err != nil {
		return storedCredentials{}, path, err
	}
	clientID, err := clientIDForEnvironment(settings, profile)
	if err != nil {
		return storedCredentials{}, path, err
	}
	refreshed, err := refreshAccessToken(ctx, profile, clientID, creds)
	if err != nil {
		return storedCredentials{}, path, err
	}
	if err := saveStoredCredentials(path, refreshed); err != nil {
		return refreshed, path, fmt.Errorf("save refreshed credentials: %w", err)
	}
	return refreshed, path, nil
}

type jwtClaims struct {
	Email            string `json:"email"`
	BrightestBioEmail string `json:"brightestbio.com/email"`
	Organization     string `json:"brightestbio.com/organization"`
	UserType         string `json:"brightestbio.com/user-type"`
	Sub              string `json:"sub"`
}

func identityFromToken(accessToken string) string {
	claims, err := decodeJWTClaims(accessToken)
	if err != nil {
		return ""
	}
	email := strings.TrimSpace(claims.BrightestBioEmail)
	if email == "" {
		email = strings.TrimSpace(claims.Email)
	}
	var parts []string
	if email != "" {
		parts = append(parts, "email: "+email)
	}
	if org := strings.TrimSpace(claims.Organization); org != "" {
		parts = append(parts, "organization: "+org)
	}
	if ut := strings.TrimSpace(claims.UserType); ut != "" {
		parts = append(parts, "user-type: "+ut)
	}
	return strings.Join(parts, "\n")
}

func decodeJWTClaims(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return jwtClaims{}, fmt.Errorf("invalid JWT")
	}
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	dec, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		dec, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			dec, err = base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return jwtClaims{}, err
			}
		}
	}
	dec = bytes.TrimRight(dec, "\x00")
	var claims jwtClaims
	if err := json.Unmarshal(dec, &claims); err != nil {
		return jwtClaims{}, err
	}
	return claims, nil
}
