package slack

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestHubOAuthRedirectURL(t *testing.T) {
	SetHubPublicBaseURL("http://localhost:18765")
	got := HubOAuthRedirectURL()
	want := "http://localhost:18765/api/slack/oauth/callback"
	if got != want {
		t.Fatalf("redirect = %q, want %q", got, want)
	}
}

func TestResolveOAuthAppBundledExampleNotUsed(t *testing.T) {
	SetHubPublicBaseURL("http://127.0.0.1:9999")
	cfg := &config.SlackConfig{}
	o, src := ResolveOAuthApp(cfg)
	// Default build embeds oauth.json.example with placeholders — not valid bundled creds.
	if src == OAuthSourceBundled && o != nil {
		if o.ClientID == "YOUR_SLACK_CLIENT_ID" {
			t.Fatal("placeholder bundled creds must not be used")
		}
	}
}

func TestResolveOAuthAppEnv(t *testing.T) {
	useTempHomeDir(t)
	t.Setenv("NEURAL_JUNKIE_SLACK_CLIENT_ID", "cid-test")
	t.Setenv("NEURAL_JUNKIE_SLACK_CLIENT_SECRET", "secret-test")
	SetHubPublicBaseURL("http://localhost:18765")
	o, src := ResolveOAuthApp(nil)
	if src != OAuthSourceEnv || o == nil || o.ClientID != "cid-test" {
		cid := ""
		if o != nil {
			cid = o.ClientID
		}
		t.Fatalf("env resolve: src=%s clientID=%q", src, cid)
	}
	if o.RedirectURL != "http://localhost:18765/api/slack/oauth/callback" {
		t.Fatalf("redirect = %q", o.RedirectURL)
	}
}

func TestBundledVendorValid(t *testing.T) {
	if bundledVendorValid(&bundledVendorCredentials{ClientID: "YOUR_SLACK_CLIENT_ID", ClientSecret: "x"}) {
		t.Fatal("placeholder client_id should be invalid")
	}
	if !bundledVendorValid(&bundledVendorCredentials{ClientID: "abc", ClientSecret: "def"}) {
		t.Fatal("real-looking creds should be valid")
	}
}
