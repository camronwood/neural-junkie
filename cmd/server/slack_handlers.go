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

	"github.com/camronwood/neural-junkie/internal/collaboration/actions"
	"github.com/camronwood/neural-junkie/internal/config"
	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
	"github.com/camronwood/neural-junkie/internal/hub"
	slackapi "github.com/slack-go/slack"
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
	applyCollabSlackActionConfig()
}

func applyCollabSlackActionConfig() {
	if chatHub == nil {
		return
	}
	cfg := actions.Config{
		AllowedHosts: nil,
		SMSEnabled:   false,
	}
	if slackBridge != nil {
		cfg.SlackEnabled = true
		cfg.ValidateSlackChannel = func(channelID string) error {
			return slackint.ValidateChannel(slackBridge.API(), channelID)
		}
		cfg.SlackPost = func(ctx context.Context, channelID, text, threadTS, username string) (string, error) {
			return slackBridge.PostRunbookMessage(channelID, text, threadTS, username)
		}
	}
	chatHub.SetCollabActionRunnerConfig(cfg)
}

func stopSlackBridge() {
	if slackBridge != nil {
		slackBridge.Stop()
		slackBridge = nil
	}
	applyCollabSlackActionConfig()
}

// slackBindingContext returns a long-lived context for agent channel subscriptions.
// HTTP request contexts must not be used — they cancel when the handler returns and
// stop agents from receiving Slack mirror channel messages.
func slackBindingContext() context.Context {
	if slackBridgeCtx != nil {
		return slackBridgeCtx
	}
	return context.Background()
}

func handleSlackStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pubOAuth := slackint.PublicOAuthFromResolved(&appConfig.Slack)
	resp := map[string]interface{}{
		"enabled":          appConfig != nil && appConfig.Slack.Enabled,
		"configured":       appConfig != nil && appConfig.Slack.SlackReady(),
		"oauth_configured": pubOAuth.Configured,
		"connect_ready":    pubOAuth.ConnectReady,
		"oauth_source":     pubOAuth.OAuthSource,
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
			"enabled":          appConfig.Slack.Enabled,
			"display_name":     appConfig.Slack.DisplayName,
			"display_icon_url": appConfig.Slack.DisplayIconURL,
			"default_policy":   appConfig.Slack.EffectiveDefaultPolicy(),
			"bot_token_set":    appConfig.Slack.BotToken != "",
			"app_token_set":    appConfig.Slack.AppToken != "",
			"connect_ready":    slackint.OAuthReady(&appConfig.Slack),
			"oauth":            slackint.PublicOAuthFromResolved(&appConfig.Slack),
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
		if strings.TrimSpace(body.SlackChannelName) == "" && slackBridge != nil {
			body.SlackChannelName = slackint.ResolveChannelName(slackBridge.API(), body.SlackChannelID)
		}
		if slackBridge != nil {
			if err := slackint.ValidateChannel(slackBridge.API(), body.SlackChannelID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
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
		ctx := slackBindingContext()
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
			_ = slackBridge.ReloadBindings(slackBindingContext())
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
	if err := slackint.ValidateChannel(slackBridge.API(), body.SlackChannelID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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
	oauth, _ := slackint.ResolveOAuthApp(&appConfig.Slack)
	if oauth == nil || oauth.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Slack OAuth is not available. Rebuild with vendor credentials or configure Advanced OAuth.",
		})
		return
	}
	redirectURI := oauth.RedirectURL
	state := newSlackOAuthState(slackint.LocalBotOAuthCallbackURL(), redirectURI)
	scopes := "app_mentions:read,channels:history,groups:history,im:history,channels:read,groups:read,chat:write,chat:write.customize,users:read,reactions:read"
	u, _ := url.Parse("https://slack.com/oauth/v2/authorize")
	q := u.Query()
	q.Set("client_id", oauth.ClientID)
	q.Set("scope", scopes)
	q.Set("redirect_uri", redirectURI)
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
	oauth, _ := slackint.ResolveOAuthApp(&appConfig.Slack)
	if oauth == nil {
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
	var result slackint.OAuthV2AccessResponse
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
	if appToken, src := slackint.ResolveAppToken(&appConfig.Slack); appToken != "" && strings.TrimSpace(appConfig.Slack.AppToken) == "" {
		appConfig.Slack.AppToken = appToken
		log.Printf("[slack] applied app token from %s after OAuth", src)
	}
	ownerID := strings.TrimSpace(result.AuthedUser.ID)
	ownerName := strings.TrimSpace(result.AuthedUser.Name)
	_ = slackint.SaveSlackInstall(&slackint.SlackInstallMetadata{
		TeamID:             result.Team.ID,
		TeamName:           result.Team.Name,
		BotUserID:          result.BotUserID,
		OwnerSlackUserID:   ownerID,
		OwnerSlackUserName: ownerName,
	})
	if ownerID != "" {
		_ = slackint.SeedInboxFromInstall(ownerID, ownerName)
	}
	_ = appConfig.Save()
	stopSlackBridge()
	startSlackBridge(slackBridgeCtx)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<html><body><p>Slack connected. You can close this window and return to Neural Junkie.</p>
<script>
try { if (window.opener) window.opener.postMessage({ type: 'nj-slack-connected' }, '*'); } catch (e) {}
setTimeout(function(){window.close();},1500);
</script></body></html>`)
}

const slackUserDMScopes = "im:history,im:read,chat:write,users:read"

func handleSlackOAuthUserDMStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil || !appConfig.Slack.SlackReady() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Connect Slack (bot) first"})
		return
	}
	oauth, _ := slackint.ResolveOAuthApp(&appConfig.Slack)
	if oauth == nil || oauth.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Slack OAuth is not available"})
		return
	}
	redirectURI := slackint.ResolveUserDMOAuthRedirectFromConfig(&appConfig.Slack)
	state := newSlackOAuthState(slackint.LocalUserDMOAuthCallbackURL(), redirectURI)
	u, _ := url.Parse("https://slack.com/oauth/v2/authorize")
	q := u.Query()
	q.Set("client_id", oauth.ClientID)
	q.Set("user_scope", slackUserDMScopes)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	if r.URL.Query().Get("json") == "1" {
		writeJSON(w, http.StatusOK, map[string]string{"url": u.String()})
		return
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func handleSlackOAuthUserDMCallback(w http.ResponseWriter, r *http.Request) {
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
	oauth, _ := slackint.ResolveOAuthApp(&appConfig.Slack)
	if oauth == nil {
		http.Error(w, "OAuth app not configured", http.StatusBadRequest)
		return
	}
	redirectURI := slackint.ResolveUserDMOAuthRedirectFromConfig(&appConfig.Slack)
	form := url.Values{}
	form.Set("client_id", oauth.ClientID)
	form.Set("client_secret", oauth.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result slackint.OAuthV2AccessResponse
	if err := json.Unmarshal(data, &result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !result.OK {
		http.Error(w, "Slack OAuth: "+result.Error, http.StatusBadRequest)
		return
	}
	userTok := strings.TrimSpace(result.AuthedUser.AccessToken)
	if userTok == "" {
		http.Error(w, "Slack did not return a user token — check User Token Scopes on the Slack app", http.StatusBadRequest)
		return
	}
	store, err := slackint.NewUserTokenStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := store.SaveToken(userTok, result.AuthedUser.ID, result.AuthedUser.Scope); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if slackBridge != nil {
		_ = slackBridge.ReloadUserTokens()
		stopSlackBridge()
		startSlackBridge(slackBridgeCtx)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<html><body><p>Slack DM access authorized. You can close this window and return to Neural Junkie.</p>
<script>
try { if (window.opener) window.opener.postMessage({ type: 'nj-slack-user-dm-connected' }, '*'); } catch (e) {}
setTimeout(function(){window.close();},1500);
</script></body></html>`)
}

func handleSlackDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stopSlackBridge()
	appConfig.Slack.BotToken = ""
	appConfig.Slack.Enabled = false
	_ = slackint.ClearSlackInstall()
	if store, err := slackint.NewUserTokenStore(); err == nil {
		_ = store.Clear()
	}
	_ = appConfig.Save()
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func handleSlackConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pub := slackint.PublicOAuthFromResolved(&appConfig.Slack)
	install, _ := slackint.LoadSlackInstall()
	bridgeConnected := false
	if slackBridge != nil {
		st := slackBridge.Status()
		if v, ok := st["connected"].(bool); ok {
			bridgeConnected = v
		}
	}
	teamID, teamName := "", ""
	ownerID, ownerName := "", ""
	if install != nil {
		teamID, teamName = install.TeamID, install.TeamName
		ownerID, ownerName = install.OwnerSlackUserID, install.OwnerSlackUserName
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"oauth_ready":          pub.ConnectReady,
		"oauth_source":         pub.OAuthSource,
		"bot_token_set":        appConfig.Slack.BotToken != "",
		"app_token_set":        appConfig.Slack.AppToken != "",
		"bridge_connected":     bridgeConnected,
		"team_id":              teamID,
		"team_name":            teamName,
		"owner_slack_user_id":  ownerID,
		"owner_slack_user_name": ownerName,
	})
}

func handleSlackRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stopSlackBridge()
	startSlackBridge(slackBridgeCtx)
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func handleSlackDiagnose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil {
		writeJSON(w, http.StatusOK, slackint.DiagnoseResult{
			Recommendations: []string{"Hub config not loaded"},
		})
		return
	}
	writeJSON(w, http.StatusOK, slackint.Diagnose(appConfig))
}

func slackAPIClient() *slackapi.Client {
	if slackBridge != nil {
		return slackBridge.API()
	}
	if appConfig == nil {
		return nil
	}
	botToken := strings.TrimSpace(appConfig.Slack.BotToken)
	if botToken == "" {
		return nil
	}
	opts := []slackapi.Option{}
	if appToken := strings.TrimSpace(appConfig.Slack.AppToken); appToken != "" {
		opts = append(opts, slackapi.OptionAppLevelToken(appToken))
	}
	return slackapi.New(botToken, opts...)
}

func handleSlackChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	api := slackAPIClient()
	if api == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Slack bot token required — connect Slack or save a bot token (xoxb-…) first",
		})
		return
	}
	channels, err := slackint.ListChannels(api)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "missing_scope") {
			msg = "missing_scope: add Bot scopes channels:read and groups:read, reinstall the app, then refresh"
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

func handleSlackInbox(w http.ResponseWriter, r *http.Request) {
	store, err := slackint.NewInboxStore()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	install, _ := slackint.LoadSlackInstall()
	switch r.Method {
	case http.MethodGet:
		cfg := store.Get()
		if cfg.OwnerSlackUserID == "" && install != nil {
			cfg.OwnerSlackUserID = install.OwnerSlackUserID
			cfg.OwnerSlackUserName = install.OwnerSlackUserName
		}
		userTok, _ := slackint.NewUserTokenStore()
		if userTok != nil {
			cfg.HumanDMAway.UserTokenSet = userTok.HasToken()
		}
		cfg.HumanDMAway.MonitoringStatus = slackint.HumanDMMonitoringStatus(cfg, cfg.HumanDMAway.UserTokenSet, time.Now())
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut, http.MethodPost:
		var body slackint.InboxConfig
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		existing := store.Get()
		if body.OwnerSlackUserID == "" {
			body.OwnerSlackUserID = existing.OwnerSlackUserID
		}
		if body.OwnerSlackUserID == "" && install != nil {
			body.OwnerSlackUserID = install.OwnerSlackUserID
			body.OwnerSlackUserName = install.OwnerSlackUserName
		}
		if body.OwnerSlackUserName == "" {
			body.OwnerSlackUserName = existing.OwnerSlackUserName
		}
		if body.SlackDMChannelID == "" {
			body.SlackDMChannelID = existing.SlackDMChannelID
		}
		if len(body.ForwardRules) == 0 && len(existing.ForwardRules) > 0 {
			body.ForwardRules = existing.ForwardRules
		}
		if len(body.ForwardRules) == 0 {
			body.ForwardRules = slackint.DefaultForwardRules()
		}
		if body.Enabled && body.AgentID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent_id required when inbox is enabled"})
			return
		}
		if !body.HumanDMAway.Enabled {
			if tokStore, err := slackint.NewUserTokenStore(); err == nil {
				_ = tokStore.Clear()
			}
		}
		saved, err := store.Save(body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ctx := slackBindingContext()
		if slackBridge != nil {
			_ = slackBridge.ReloadInbox(ctx)
		} else if saved.Enabled {
			adapter := slackint.HubAdapter{H: chatHub}
			_ = slackint.ApplyInbox(ctx, adapter, slackEnsureAgent, saved)
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSlackInboxTestDM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if slackBridge == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "slack bridge not running"})
		return
	}
	store, err := slackint.NewInboxStore()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	cfg := store.Get()
	var body struct {
		Text string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := slackBridge.PostInboxTestDM(cfg, body.Text); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleSlackInboxDMDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if slackBridge == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "slack bridge not running"})
		return
	}
	info, err := slackBridge.InboxDMDebugInfo()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func handleSlackInboxHumanDMDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if slackBridge == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "slack bridge not running"})
		return
	}
	info, err := slackBridge.HumanDMDebugInfo()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}
