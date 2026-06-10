package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestPublicRelayBotCallbackURL_defaultBase(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE", "")
	got := PublicRelayBotCallbackURL()
	want := DefaultOAuthRelayBase + OAuthCallbackPath // Cloudflare Workers default
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveBotOAuthRedirectURL_devLoopback(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE", "")
	t.Setenv("NEURAL_JUNKIE_SLACK_USE_OAUTH_RELAY", "0")
	SetHubPublicBaseURL("http://127.0.0.1:19999")
	got := ResolveBotOAuthRedirectURL(slackRedirectHints{})
	want := "http://127.0.0.1:19999/api/slack/oauth/callback"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveBotOAuthRedirectURL_bundledUsesRelay(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_SLACK_USE_OAUTH_RELAY", "1")
	SetHubPublicBaseURL("http://127.0.0.1:18765")
	got := ResolveBotOAuthRedirectURL(slackRedirectHints{})
	if !IsRelayRedirectURL(got) {
		t.Fatalf("expected relay redirect, got %q", got)
	}
}

func TestResolveBotOAuthRedirectURL_explicitOverride(t *testing.T) {
	got := ResolveBotOAuthRedirectURL(slackRedirectHints{configRedirect: "https://custom.example/cb"})
	if got != "https://custom.example/cb" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveBotOAuthRedirectURL_loopbackUserFileUpgradesToRelay(t *testing.T) {
	got := ResolveBotOAuthRedirectURL(slackRedirectHints{
		userRedirect: "http://localhost:18765/api/slack/oauth/callback",
	})
	if !IsRelayRedirectURL(got) {
		t.Fatalf("expected relay redirect, got %q", got)
	}
}

func TestResolveOAuthAppEnvUsesExplicitRedirect(t *testing.T) {
	useTempHomeDir(t)
	t.Setenv("NEURAL_JUNKIE_SLACK_CLIENT_ID", "cid")
	t.Setenv("NEURAL_JUNKIE_SLACK_CLIENT_SECRET", "secret")
	t.Setenv("NEURAL_JUNKIE_SLACK_REDIRECT_URL", "https://relay.example/api/slack/oauth/callback")
	o, src := ResolveOAuthApp(&config.SlackConfig{})
	if src != OAuthSourceEnv || o.RedirectURL != "https://relay.example/api/slack/oauth/callback" {
		t.Fatalf("src=%s redirect=%q", src, o.RedirectURL)
	}
}
