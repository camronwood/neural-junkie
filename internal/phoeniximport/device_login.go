package phoeniximport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const readScopes = "openid offline_access read:scanResults read:scanResultsAttachments read:analyses read:analysesAttachments read:readers"

// DeviceLoginStart is returned to the UI when device authorization begins.
type DeviceLoginStart struct {
	SessionID       string `json:"session_id"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Environment     string `json:"environment"`
}

// DeviceLoginPollResult is the poll response while waiting for user authorization.
type DeviceLoginPollResult struct {
	Status     string `json:"status"` // pending | success | expired | denied | error
	Identity   string `json:"identity,omitempty"`
	Hint       string `json:"hint,omitempty"`
	ExpiresIn  int    `json:"expires_in,omitempty"`
}

type deviceLoginSession struct {
	id              string
	userCode        string
	verificationURL string
	deviceCode      string
	expiresAt       time.Time
	interval        time.Duration
	settings        Settings
	profile         environmentProfile
	clientID        string
}

var (
	deviceLoginMu       sync.Mutex
	deviceLoginSessions = map[string]*deviceLoginSession{}
)

// StartDeviceLogin begins Auth0 device authorization flow.
func StartDeviceLogin(ctx context.Context, settings Settings) (*DeviceLoginStart, error) {
	profile, err := profileForEnvironment(settings.EnvironmentOrDefault())
	if err != nil {
		return nil, err
	}
	clientID, err := clientIDForEnvironment(settings, profile)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("audience", profile.Audience)
	form.Set("scope", readScopes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+profile.Auth0Domain+"/oauth/device/code", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("device code HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var device struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		VerificationURI         string `json:"verification_uri"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(body, &device); err != nil {
		return nil, err
	}
	verify := strings.TrimSpace(device.VerificationURIComplete)
	if verify == "" {
		verify = strings.TrimSpace(device.VerificationURI)
	}
	if verify == "" || strings.TrimSpace(device.DeviceCode) == "" {
		return nil, fmt.Errorf("invalid device code response")
	}

	id, err := randomSessionID()
	if err != nil {
		return nil, err
	}
	expiresIn := device.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 600
	}
	interval := device.Interval
	if interval < 2 {
		interval = 2
	}

	sess := &deviceLoginSession{
		id:              id,
		userCode:        device.UserCode,
		verificationURL: verify,
		deviceCode:      device.DeviceCode,
		expiresAt:       time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
		interval:        time.Duration(interval) * time.Second,
		settings:        settings,
		profile:         profile,
		clientID:        clientID,
	}
	deviceLoginMu.Lock()
	pruneExpiredDeviceSessionsLocked()
	deviceLoginSessions[id] = sess
	deviceLoginMu.Unlock()

	return &DeviceLoginStart{
		SessionID:       id,
		UserCode:        sess.userCode,
		VerificationURL: sess.verificationURL,
		ExpiresIn:       expiresIn,
		Environment:     settings.EnvironmentOrDefault(),
	}, nil
}

// PollDeviceLogin checks whether the user completed device authorization.
func PollDeviceLogin(ctx context.Context, sessionID string) (*DeviceLoginPollResult, error) {
	sess := getDeviceLoginSession(sessionID)
	if sess == nil {
		return &DeviceLoginPollResult{Status: "expired", Hint: "login session expired; start again"}, nil
	}
	if time.Now().UTC().After(sess.expiresAt) {
		removeDeviceLoginSession(sessionID)
		return &DeviceLoginPollResult{Status: "expired", Hint: "device code expired"}, nil
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("device_code", sess.deviceCode)
	form.Set("client_id", sess.clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+sess.profile.Auth0Domain+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	var tok struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken *string `json:"refresh_token"`
		ExpiresIn    *int    `json:"expires_in"`
		TokenType    *string `json:"token_type"`
		Error        string  `json:"error"`
		ErrorDesc    string  `json:"error_description"`
	}
	_ = json.Unmarshal(body, &tok)

	if res.StatusCode >= 200 && res.StatusCode < 300 && strings.TrimSpace(tok.AccessToken) != "" {
		creds := storedCredentials{
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			TokenType:    tok.TokenType,
		}
		if tok.ExpiresIn != nil && *tok.ExpiresIn > 0 {
			exp := time.Now().UTC().Add(time.Duration(*tok.ExpiresIn) * time.Second)
			creds.ExpiresAt = &exp
		}
		credPath := resolveCredentialsPath(sess.settings)
		if err := saveStoredCredentials(credPath, creds); err != nil {
			return &DeviceLoginPollResult{Status: "error", Hint: err.Error()}, nil
		}
		removeDeviceLoginSession(sessionID)
		return &DeviceLoginPollResult{
			Status:   "success",
			Identity: identityFromToken(tok.AccessToken),
		}, nil
	}

	switch tok.Error {
	case "authorization_pending", "":
		remaining := int(time.Until(sess.expiresAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		return &DeviceLoginPollResult{Status: "pending", ExpiresIn: remaining}, nil
	case "slow_down":
		sess.interval += 2 * time.Second
		return &DeviceLoginPollResult{Status: "pending", Hint: "slow down"}, nil
	case "expired_token":
		removeDeviceLoginSession(sessionID)
		return &DeviceLoginPollResult{Status: "expired", Hint: "device code expired"}, nil
	case "access_denied":
		removeDeviceLoginSession(sessionID)
		return &DeviceLoginPollResult{Status: "denied", Hint: "access denied"}, nil
	default:
		msg := strings.TrimSpace(tok.ErrorDesc)
		if msg == "" {
			msg = tok.Error
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		return &DeviceLoginPollResult{Status: "error", Hint: msg}, nil
	}
}

// Logout removes stored TIM credentials for the configured environment.
func Logout(settings Settings) error {
	path := resolveCredentialsPath(settings)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func randomSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func getDeviceLoginSession(id string) *deviceLoginSession {
	deviceLoginMu.Lock()
	defer deviceLoginMu.Unlock()
	pruneExpiredDeviceSessionsLocked()
	return deviceLoginSessions[id]
}

func removeDeviceLoginSession(id string) {
	deviceLoginMu.Lock()
	delete(deviceLoginSessions, id)
	deviceLoginMu.Unlock()
}

func pruneExpiredDeviceSessionsLocked() {
	now := time.Now().UTC()
	for id, s := range deviceLoginSessions {
		if now.After(s.expiresAt) {
			delete(deviceLoginSessions, id)
		}
	}
}
