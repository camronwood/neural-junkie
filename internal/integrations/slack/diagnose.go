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

// DiagnoseResult is returned by GET /api/slack/diagnose (no secrets).
type DiagnoseResult struct {
	Ready              bool   `json:"ready"`
	AppTokenOK         bool   `json:"app_token_format_ok"`
	BotTokenOK         bool   `json:"bot_token_format_ok"`
	AppTokenHint       string `json:"app_token_hint,omitempty"`
	BotTokenHint       string `json:"bot_token_hint,omitempty"`
	AuthTestOK         bool   `json:"auth_test_ok"`
	SocketOpenOK       bool   `json:"socket_open_ok"`
	SocketOpenError    string `json:"socket_open_error,omitempty"`
	InboundEventSubs   []string `json:"inbound_event_subscriptions_required,omitempty"`
	Recommendations    []string `json:"recommendations,omitempty"`
}

// Bot event subscriptions required for inbound (Socket Mode still needs these enabled in the app UI).
var inboundEventSubscriptionsRequired = []string{
	"message.channels",
	"message.groups",
	"app_mention",
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
		out.Recommendations = append(out.Recommendations,
			"Inbound: api.slack.com → Your App → Event Subscriptions → Enable Events → Subscribe to bot events: message.channels, message.groups (required for private #channels), app_mention. Then reinstall app to workspace if you added scopes.",
			"Invite @neural_junkie to the channel; binding channel ID must match Slack → channel → About (C…).",
			"Hub debug: NEURAL_JUNKIE_SLACK_DEBUG=1 then post in Slack; logs show [slack] event message or unhandled event type.",
		)
	}
	return out
}
