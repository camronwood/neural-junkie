package slack

import (
	"net/url"
	"testing"
)

func TestFormatAndParseOAuthState(t *testing.T) {
	local := "http://127.0.0.1:18765/api/slack/oauth/callback"
	state := FormatOAuthState("abc123", local)
	nonce, ret, ok := ParseOAuthState(state)
	if !ok || nonce != "abc123" || ret != local {
		t.Fatalf("parse = (%q, %q, %v)", nonce, ret, ok)
	}
	nonce2, ok2 := OAuthStateNonce(state)
	if !ok2 || nonce2 != "abc123" {
		t.Fatalf("OAuthStateNonce = %q ok=%v", nonce2, ok2)
	}
}

func TestParseOAuthState_legacyHex(t *testing.T) {
	legacy := "deadbeefcafebabe0123456789abcdef"
	nonce, ret, ok := ParseOAuthState(legacy)
	if !ok || nonce != legacy || ret != "" {
		t.Fatalf("legacy parse = (%q, %q, %v)", nonce, ret, ok)
	}
}

func TestIsAllowedLocalOAuthCallback(t *testing.T) {
	okURL, _ := url.Parse("http://127.0.0.1:18765/api/slack/oauth/callback")
	if !IsAllowedLocalOAuthCallback(okURL) {
		t.Fatal("expected loopback callback allowed")
	}
	badURL, _ := url.Parse("http://evil.example/api/slack/oauth/callback")
	if IsAllowedLocalOAuthCallback(badURL) {
		t.Fatal("expected remote callback rejected")
	}
}

func TestBuildRelayRedirectURL(t *testing.T) {
	local := "http://localhost:18765/api/slack/oauth/callback"
	q := url.Values{}
	q.Set("code", "test-code")
	q.Set("state", "nj1.nonce.enc")
	got, err := BuildRelayRedirectURL(local, q)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	if u.Query().Get("code") != "test-code" {
		t.Fatalf("code = %q", u.Query().Get("code"))
	}
}
