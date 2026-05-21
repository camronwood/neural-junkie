package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
	"github.com/camronwood/neural-junkie/internal/hub"
)

var slackBridge *slackint.Bridge

func slackEnsureAgent(ctx context.Context, agentID, channelName string) error {
	if chatHub == nil {
		return fmt.Errorf("hub not ready")
	}
	ch := chatHub.GetCommandHandler()
	if ch == nil {
		return fmt.Errorf("command handler not ready")
	}
	if h, ok := ch.(*hub.CommandHandler); ok {
		return h.EnsureAgentSubscribedToChannel(ctx, agentID, channelName)
	}
	return nil
}

func startSlackBridge(ctx context.Context) {
	if appConfig == nil || !appConfig.Slack.SlackReady() {
		return
	}
	adapter := slackint.HubAdapter{H: chatHub}
	b, err := slackint.NewBridge(appConfig, adapter, slackEnsureAgent)
	if err != nil {
		log.Printf("[slack] bridge init: %v", err)
		return
	}
	if err := b.Start(ctx); err != nil {
		log.Printf("[slack] bridge start: %v", err)
		return
	}
	slackBridge = b
	log.Println("[slack] bridge started (Socket Mode)")
}

func stopSlackBridge() {
	if slackBridge != nil {
		slackBridge.Stop()
		slackBridge = nil
	}
}

func handleSlackStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := map[string]interface{}{
		"enabled":          appConfig != nil && appConfig.Slack.Enabled,
		"configured":       appConfig != nil && appConfig.Slack.SlackReady(),
		"oauth_configured": slackint.PublicOAuthFromDir().Configured,
		"token_set":        appConfig != nil && appConfig.Slack.BotToken != "" && appConfig.Slack.AppToken != "",
	}
	if slackBridge != nil {
		for k, v := range slackBridge.Status() {
			resp[k] = v
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleSlackConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		pub := map[string]interface{}{
			"enabled":         appConfig.Slack.Enabled,
			"display_name":    appConfig.Slack.DisplayName,
			"display_icon_url": appConfig.Slack.DisplayIconURL,
			"default_policy":  appConfig.Slack.EffectiveDefaultPolicy(),
			"bot_token_set":   appConfig.Slack.BotToken != "",
			"app_token_set":   appConfig.Slack.AppToken != "",
			"oauth":           slackint.PublicOAuthFromDir(),
		}
		writeJSON(w, http.StatusOK, pub)
	case http.MethodPut, http.MethodPost:
		var body struct {
			Enabled        *bool   `json:"enabled"`
			AppToken       string  `json:"app_token"`
			BotToken       string  `json:"bot_token"`
			DisplayName    string  `json:"display_name"`
			DisplayIconURL string  `json:"display_icon_url"`
			DefaultPolicy  string  `json:"default_policy"`
			ClientID       string  `json:"client_id"`
			ClientSecret   string  `json:"client_secret"`
			RedirectURL    string  `json:"redirect_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Enabled != nil {
			appConfig.Slack.Enabled = *body.Enabled
		}
		if t := strings.TrimSpace(body.AppToken); t != "" {
			appConfig.Slack.AppToken = t
		}
		if t := strings.TrimSpace(body.BotToken); t != "" {
			appConfig.Slack.BotToken = t
		}
		if body.DisplayName != "" {
			appConfig.Slack.DisplayName = strings.TrimSpace(body.DisplayName)
		}
		if body.DisplayIconURL != "" {
			appConfig.Slack.DisplayIconURL = strings.TrimSpace(body.DisplayIconURL)
		}
		if body.DefaultPolicy != "" {
			appConfig.Slack.DefaultPolicy = config.SlackPolicy(body.DefaultPolicy)
		}
		if body.ClientID != "" || body.ClientSecret != "" || body.RedirectURL != "" {
			existing, _ := slackint.LoadOAuthApp()
			secret := strings.TrimSpace(body.ClientSecret)
			if secret == "" && existing != nil {
				secret = existing.ClientSecret
			}
			_ = slackint.SaveOAuthApp(&slackint.OAuthAppCredentials{
				ClientID:     strings.TrimSpace(body.ClientID),
				ClientSecret: secret,
				RedirectURL:  strings.TrimSpace(body.RedirectURL),
			})
		}
		if err := appConfig.Save(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSlackBindings(w http.ResponseWriter, r *http.Request) {
	store, err := slackint.NewBindingStore()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, store.List())
	case http.MethodPost:
		var body struct {
			SlackChannelID   string `json:"slack_channel_id"`
			SlackChannelName string `json:"slack_channel_name"`
			AgentID          string `json:"agent_id"`
			AgentName        string `json:"agent_name"`
			Policy           string `json:"policy"`
			Enabled          *bool  `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.SlackChannelID == "" || body.AgentID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slack_channel_id and agent_id required"})
			return
		}
		teamID := ""
		if slackBridge != nil {
			teamID = slackBridge.TeamID()
		}
		b := slackint.NewBindingFromRequest(
			teamID,
			body.SlackChannelID,
			body.SlackChannelName,
			body.AgentID,
			body.AgentName,
			config.SlackPolicy(body.Policy),
			appConfig,
		)
		if body.Enabled != nil {
			b.Enabled = *body.Enabled
		}
		saved, err := store.Upsert(b)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ctx := r.Context()
		if slackBridge != nil {
			_ = slackBridge.ApplyBinding(ctx, saved)
			_ = slackBridge.ReloadBindings(ctx)
		} else {
			adapter := slackint.HubAdapter{H: chatHub}
			_ = slackint.ApplyBinding(ctx, adapter, slackEnsureAgent, saved)
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("slack_channel_id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slack_channel_id query required"})
			return
		}
		if err := store.Delete(id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if slackBridge != nil {
			_ = slackBridge.ReloadBindings(r.Context())
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSlackTestPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if slackBridge == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "slack bridge not running"})
		return
	}
	var body struct {
		SlackChannelID string `json:"slack_channel_id"`
		Text           string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.SlackChannelID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slack_channel_id required"})
		return
	}
	if err := slackBridge.PostTestMessage(body.SlackChannelID, body.Text); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleSlackOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	oauth, err := slackint.LoadOAuthApp()
	if err != nil || oauth == nil || oauth.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Slack OAuth app not configured. Set client_id and secret in Settings → Slack.",
		})
		return
	}
	state := newOAuthState()
	scopes := "app_mentions:read,channels:history,groups:history,chat:write,chat:write.customize,users:read"
	u, _ := url.Parse("https://slack.com/oauth/v2/authorize")
	q := u.Query()
	q.Set("client_id", oauth.ClientID)
	q.Set("scope", scopes)
	q.Set("redirect_uri", oauth.RedirectURL)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	if r.URL.Query().Get("json") == "1" {
		writeJSON(w, http.StatusOK, map[string]string{"url": u.String()})
		return
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func handleSlackOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Error(w, "Slack OAuth error: "+errParam, http.StatusBadRequest)
		return
	}
	if !validOAuthState(r.URL.Query().Get("state")) {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}
	oauth, err := slackint.LoadOAuthApp()
	if err != nil || oauth == nil {
		http.Error(w, "OAuth app not configured", http.StatusBadRequest)
		return
	}
	form := url.Values{}
	form.Set("client_id", oauth.ClientID)
	form.Set("client_secret", oauth.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", oauth.RedirectURL)
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !result.OK {
		http.Error(w, "Slack OAuth: "+result.Error, http.StatusBadRequest)
		return
	}
	appConfig.Slack.BotToken = result.AccessToken
	appConfig.Slack.Enabled = true
	_ = appConfig.Save()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<html><body><p>Slack connected. You can close this window and return to Neural Junkie.</p>
<script>setTimeout(function(){window.close();},1500);</script></body></html>`)
}

func handleSlackDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stopSlackBridge()
	appConfig.Slack.BotToken = ""
	appConfig.Slack.AppToken = ""
	appConfig.Slack.Enabled = false
	_ = appConfig.Save()
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func handleSlackRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stopSlackBridge()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	startSlackBridge(ctx)
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}
