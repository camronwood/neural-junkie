package slack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// DiagnoseCheck is one actionable setup item for the Slack integration UI.
type DiagnoseCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"` // pass | warn | fail
	Label  string `json:"label"`
	Fix    string `json:"fix,omitempty"`
}

// DiagnoseResult is returned by GET /api/slack/diagnose (no secrets).
type DiagnoseResult struct {
	Ready              bool            `json:"ready"`
	AppTokenOK         bool            `json:"app_token_format_ok"`
	BotTokenOK         bool            `json:"bot_token_format_ok"`
	AppTokenHint       string          `json:"app_token_hint,omitempty"`
	BotTokenHint       string          `json:"bot_token_hint,omitempty"`
	AuthTestOK         bool            `json:"auth_test_ok"`
	SocketOpenOK       bool            `json:"socket_open_ok"`
	SocketOpenError    string          `json:"socket_open_error,omitempty"`
	InboxDMHistoryOK   bool            `json:"inbox_dm_history_ok,omitempty"`
	InboxDMHistoryHint string          `json:"inbox_dm_history_hint,omitempty"`
	InboundEventSubs   []string        `json:"inbound_event_subscriptions_required,omitempty"`
	Recommendations    []string        `json:"recommendations,omitempty"`
	Checks             []DiagnoseCheck `json:"checks,omitempty"`
	BridgeConnected    bool            `json:"bridge_connected,omitempty"`
	ChannelsFound      int             `json:"channels_found,omitempty"`
	BindingsCount      int             `json:"bindings_count,omitempty"`
}

// DiagnoseRuntimeContext adds live bridge/binding state to diagnose checks.
type DiagnoseRuntimeContext struct {
	BridgeConnected bool
	BindingsCount   int
}

// Bot event subscriptions required for inbound (Socket Mode still needs these enabled in the app UI).
var inboundEventSubscriptionsRequired = []string{
	"message.channels",
	"message.groups",
	"message.im",
	"app_mention",
	"reaction_added",
}

func tokenHint(tok, wantPrefix string) (ok bool, hint string) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false, "missing"
	}
	if strings.HasPrefix(tok, "enc:") || strings.HasPrefix(tok, "ENC:") {
		return false, "still encrypted — restart hub after saving tokens in Settings"
	}
	if !strings.HasPrefix(tok, wantPrefix) {
		if strings.HasPrefix(tok, "xoxb-") && wantPrefix == "xapp-" {
			return false, "looks like bot token (xoxb) — App token must be xapp- from Socket Mode"
		}
		if strings.HasPrefix(tok, "xapp-") && wantPrefix == "xoxb-" {
			return false, "looks like app token (xapp) — Bot token must be xoxb- from OAuth install"
		}
		return false, "unexpected format"
	}
	return true, "ok"
}

// Diagnose checks token shapes and whether Slack accepts Socket Mode open.
func Diagnose(cfg *config.Config) DiagnoseResult {
	return DiagnoseWithRuntime(cfg, DiagnoseRuntimeContext{})
}

// DiagnoseWithRuntime runs Diagnose and adds live bridge/binding checklist items.
func DiagnoseWithRuntime(cfg *config.Config, runtime DiagnoseRuntimeContext) DiagnoseResult {
	out := diagnoseCore(cfg)
	out.BridgeConnected = runtime.BridgeConnected
	out.BindingsCount = runtime.BindingsCount
	if out.AuthTestOK && cfg != nil && cfg.Slack.BotToken != "" {
		api := slackapi.New(cfg.Slack.BotToken, slackapi.OptionAppLevelToken(cfg.Slack.AppToken))
		if channels, err := ListChannels(api); err == nil {
			out.ChannelsFound = len(channels)
		}
	}
	out.Checks = buildDiagnoseChecks(out)
	return out
}

func diagnoseCore(cfg *config.Config) DiagnoseResult {
	out := DiagnoseResult{}
	if cfg == nil {
		out.Recommendations = []string{"Slack is not configured in ~/.neural-junkie/config.json"}
		return out
	}
	s := cfg.Slack
	out.Ready = s.SlackReady()
	var ok bool
	ok, out.AppTokenHint = tokenHint(s.AppToken, "xapp-")
	out.AppTokenOK = ok
	ok, out.BotTokenHint = tokenHint(s.BotToken, "xoxb-")
	out.BotTokenOK = ok

	if !out.AppTokenOK || !out.BotTokenOK {
		out.Recommendations = append(out.Recommendations,
			"Settings → Integrations → Slack: App token = xapp-… (Socket Mode, scope connections:write); Bot token = xoxb-…",
		)
		return out
	}

	api := slackapi.New(s.BotToken, slackapi.OptionAppLevelToken(s.AppToken))
	if auth, err := api.AuthTest(); err != nil {
		out.SocketOpenError = fmt.Sprintf("auth.test: %v", err)
		out.Recommendations = append(out.Recommendations, "Re-install the app to the workspace and paste a fresh Bot token (xoxb-).")
	} else {
		out.AuthTestOK = true
		if auth.UserID != "" {
			if _, err := api.GetUserInfo(auth.UserID); err != nil {
				out.Recommendations = append(out.Recommendations,
					"users:read — add Bot scope users:read and reinstall the app so NJ can show Slack display names (not \"Slack User\").",
				)
			}
		}
	}

	socket := socketmode.New(api)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	_, _, err := socket.OpenContext(ctx)
	if err != nil {
		out.SocketOpenError = err.Error()
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "missing_scope"):
			out.Recommendations = append(out.Recommendations,
				"App-Level Token is missing connections:write. api.slack.com → Your App → Socket Mode → Generate app-level token with scope connections:write only (xapp-…). Paste into Settings → App token, not the bot token.",
			)
		case strings.Contains(msg, "invalid_auth"), strings.Contains(msg, "not_authed"):
			out.Recommendations = append(out.Recommendations,
				"Regenerate the App-Level Token at api.slack.com → Socket Mode (scope connections:write) and save it as App token.",
			)
		case strings.Contains(msg, "account_inactive"):
			out.Recommendations = append(out.Recommendations, "Slack app or workspace is inactive — check the app is installed.")
		default:
			out.Recommendations = append(out.Recommendations,
				"Enable Socket Mode on the Slack app, enable Event Subscriptions (message.channels, message.groups, app_mention), then Save tokens & restart bridge.",
			)
		}
	} else {
		out.SocketOpenOK = true
		out.InboundEventSubs = inboundEventSubscriptionsRequired
		checkInboxDMHistory(api, &out)
		out.Recommendations = append(out.Recommendations,
			"Inbound: api.slack.com → Event Subscriptions → Subscribe to bot events: message.channels, message.groups, message.im (DM inbox), app_mention. Reinstall after scope changes.",
			"Personal inbox DMs require message.im (real-time) or im:history (NJ polls every 2s as fallback).",
			"Invite @neural_junkie to bound channels; binding channel ID must match Slack → channel → About (C…).",
			"Hub debug: NEURAL_JUNKIE_SLACK_DEBUG=1 then post in Slack; logs show [slack] im inbound or inbox DM poll.",
		)
	}
	return out
}

func buildDiagnoseChecks(out DiagnoseResult) []DiagnoseCheck {
	var checks []DiagnoseCheck
	add := func(id, status, label, fix string) {
		checks = append(checks, DiagnoseCheck{ID: id, Status: status, Label: label, Fix: fix})
	}

	if out.AppTokenOK && out.BotTokenOK {
		add("tokens", "pass", "App and bot tokens look valid", "")
	} else {
		add("tokens", "fail", "Slack tokens missing or wrong format", "Settings → Slack → Connect or Advanced tokens")
	}

	if out.AuthTestOK {
		add("auth_test", "pass", "Slack auth.test succeeded", "")
	} else {
		add("auth_test", "fail", "Slack auth.test failed", "Reinstall the app and reconnect Slack")
	}

	if out.SocketOpenOK {
		add("socket_open", "pass", "Socket Mode connection opens", "")
	} else if out.AppTokenOK && out.BotTokenOK {
		add("socket_open", "fail", "Socket Mode failed to open", out.SocketOpenError)
	}

	if out.BridgeConnected {
		add("bridge_connected", "pass", "Bridge is connected", "")
	} else if out.SocketOpenOK {
		add("bridge_connected", "warn", "Bridge not connected yet", "Restart Slack bridge in Settings")
	}

	switch {
	case out.ChannelsFound > 0:
		add("channels_found", "pass", fmt.Sprintf("Bot is in %d channel(s)", out.ChannelsFound), "")
	case out.AuthTestOK:
		add("channels_found", "warn", "Bot is not in any channels", "Run /invite @YourBot in each channel you want to use")
	}

	switch {
	case out.BindingsCount > 0:
		add("binding_exists", "pass", fmt.Sprintf("%d channel binding(s) saved", out.BindingsCount), "")
	case out.AuthTestOK:
		add("binding_exists", "warn", "No channel bindings yet", "Pick a channel and assign an agent in Settings → Slack")
	}

	if out.InboxDMHistoryOK {
		add("inbox_dm_history", "pass", "Personal inbox DM history OK", "")
	} else if out.InboxDMHistoryHint != "" {
		add("inbox_dm_history", "warn", "Personal inbox DM not verified", out.InboxDMHistoryHint)
	}

	return checks
}

func checkInboxDMHistory(api *slackapi.Client, out *DiagnoseResult) {
	store, err := NewInboxStore()
	if err != nil || store == nil {
		return
	}
	inbox := store.Get()
	if !inbox.Enabled {
		return
	}
	channelID := strings.TrimSpace(inbox.SlackDMChannelID)
	if channelID == "" {
		owner := strings.TrimSpace(inbox.OwnerSlackUserID)
		if owner == "" {
			out.InboxDMHistoryHint = "open a DM with the bot once, or save inbox settings after OAuth"
			return
		}
		auth, err := api.AuthTest()
		if err != nil {
			out.InboxDMHistoryHint = "auth.test: " + err.Error()
			return
		}
		ch, _, _, err := api.OpenConversation(&slackapi.OpenConversationParameters{
			Users: []string{owner, auth.UserID},
		})
		if err != nil {
			out.InboxDMHistoryHint = "im:read/im:history may be missing — " + err.Error()
			out.Recommendations = append(out.Recommendations,
				"Add bot scope im:history, reinstall the app, then DM the bot once.",
			)
			return
		}
		if ch != nil {
			channelID = ch.ID
		}
	}
	if channelID == "" {
		return
	}
	_, err = api.GetConversationHistory(&slackapi.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     1,
	})
	if err != nil {
		out.InboxDMHistoryHint = err.Error()
		if strings.Contains(strings.ToLower(err.Error()), "missing_scope") {
			out.Recommendations = append(out.Recommendations,
				"Add bot scope im:history for personal inbox DMs; reinstall the app to the workspace.",
			)
		}
		return
	}
	out.InboxDMHistoryOK = true
	out.InboxDMHistoryHint = "ok"
}
