package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/google/meetnotes"
	"github.com/camronwood/neural-junkie/internal/hub"
	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
)

var (
	oauthStateMu sync.Mutex
	oauthStates  = map[string]time.Time{}
)

func meetnotesBaseDir() string {
	if a := getAssistantAgent(); a != nil {
		if dir := a.MeetNotesBaseDir(); dir != "" {
			return dir
		}
	}
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ""
	}
	return storage.BaseDir()
}

func getAssistantAgent() *agent.AssistantAgent {
	if chatHub == nil {
		return nil
	}
	ch := chatHub.GetCommandHandler()
	if ch == nil {
		return nil
	}
	h, ok := ch.(*hub.CommandHandler)
	if !ok {
		return nil
	}
	return h.GetAssistantAgent()
}

func pruneOAuthStates() {
	cutoff := time.Now().Add(-10 * time.Minute)
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	for k, t := range oauthStates {
		if t.Before(cutoff) {
			delete(oauthStates, k)
		}
	}
}

func registerOAuthNonce(nonce string) {
	oauthStateMu.Lock()
	oauthStates[nonce] = time.Now()
	oauthStateMu.Unlock()
	pruneOAuthStates()
}

func newOAuthState() string {
	nonce := slackint.NewOAuthNonce()
	registerOAuthNonce(nonce)
	return nonce
}

func newSlackOAuthState(localCallback, redirectURI string) string {
	nonce := slackint.NewOAuthNonce()
	registerOAuthNonce(nonce)
	if slackint.IsRelayRedirectURL(redirectURI) {
		return slackint.FormatOAuthState(nonce, localCallback)
	}
	return nonce
}

func validOAuthState(state string) bool {
	nonce, ok := slackint.OAuthStateNonce(state)
	if !ok {
		return false
	}
	oauthStateMu.Lock()
	defer oauthStateMu.Unlock()
	created, ok := oauthStates[nonce]
	if !ok {
		return false
	}
	delete(oauthStates, nonce)
	return time.Since(created) < 15*time.Minute
}

func handleAssistantGoogleConfig(w http.ResponseWriter, r *http.Request) {
	baseDir := meetnotesBaseDir()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, meetnotes.PublicAppConfigFromDir(baseDir))
	case http.MethodPut, http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var body struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			RedirectURL  string `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		existing, _ := meetnotes.LoadAppCredentials(baseDir)
		secret := strings.TrimSpace(body.ClientSecret)
		if secret == "" && existing != nil {
			secret = existing.ClientSecret
		}
		creds := &meetnotes.AppOAuthCredentials{
			ClientID:     strings.TrimSpace(body.ClientID),
			ClientSecret: secret,
			RedirectURL:  strings.TrimSpace(body.RedirectURL),
		}
		if err := meetnotes.SaveAppCredentials(baseDir, creds); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, meetnotes.PublicAppConfigFromDir(baseDir))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAssistantGoogleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a := getAssistantAgent()
	if a == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "assistant not available"})
		return
	}
	connected, email, lastSync, count, oauthConfigured, _ := a.GoogleMeetNotesStatus(r.Context())
	appConfig := meetnotes.PublicAppConfigFromDir(a.MeetNotesBaseDir())
	resp := map[string]interface{}{
		"connected":        connected,
		"email":            email,
		"notes_count":      count,
		"oauth_configured": oauthConfigured,
		"connect_ready":    appConfig.ConnectReady,
		"oauth_source":     appConfig.OAuthSource,
	}
	if lastSync != nil {
		resp["last_sync_at"] = lastSync.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleAssistantGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	baseDir := meetnotesBaseDir()
	if !meetnotes.OAuthConfigured(baseDir) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Google OAuth is not available. Configure Advanced Google OAuth or use a release build with bundled credentials.",
		})
		return
	}
	state := newOAuthState()
	url, err := meetnotes.AuthCodeURL(baseDir, state)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if r.URL.Query().Get("json") == "1" {
		writeJSON(w, http.StatusOK, map[string]string{"url": url})
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func handleAssistantGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Error(w, fmt.Sprintf("Google OAuth error: %s", errParam), http.StatusBadRequest)
		return
	}
	state := r.URL.Query().Get("state")
	if !validOAuthState(state) {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}
	a := getAssistantAgent()
	if a == nil || a.MeetNotesBaseDir() == "" {
		http.Error(w, "Assistant not available", http.StatusServiceUnavailable)
		return
	}
	store := &meetnotes.TokenStore{BaseDir: a.MeetNotesBaseDir()}
	if _, err := store.ExchangeCode(r.Context(), code); err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}
	if a := getAssistantAgent(); a != nil {
		a.EnsureGoogleMeetNotesSync(context.Background())
		go func() {
			if _, err := a.SyncGoogleMeetNotes(context.Background()); err != nil {
				log.Printf("[google] initial meet notes sync: %v", err)
			}
		}()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><body style="font-family:system-ui;padding:2rem">
<h1>Google connected</h1>
<p>Meeting notes sync is enabled. You can close this tab and return to Neural Junkie.</p>
<p>Use <strong>Sync now</strong> in Settings to pull notes immediately.</p>
</body></html>`)
}

func handleAssistantGoogleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a := getAssistantAgent()
	if a == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "assistant not available"})
		return
	}
	if err := a.DisconnectGoogle(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "disconnected"})
}

func handleAssistantGoogleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a := getAssistantAgent()
	if a == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "assistant not available"})
		return
	}
	n, err := a.SyncGoogleMeetNotes(context.Background())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"ingested": n})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
