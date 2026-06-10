package slack

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strings"
)

const oauthStatePrefix = "nj1."

// FormatOAuthState encodes a hub nonce and local callback URL for the public HTTPS relay.
func FormatOAuthState(nonce, localCallbackURL string) string {
	nonce = strings.TrimSpace(nonce)
	localCallbackURL = strings.TrimSpace(localCallbackURL)
	if nonce == "" || localCallbackURL == "" {
		return nonce
	}
	return oauthStatePrefix + nonce + "." + base64.RawURLEncoding.EncodeToString([]byte(localCallbackURL))
}

// ParseOAuthState extracts the hub nonce and optional local callback URL from a Slack state value.
// Legacy plain hex states (no relay) return ok=true with localReturn="".
func ParseOAuthState(state string) (nonce string, localReturn string, ok bool) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", "", false
	}
	if !strings.HasPrefix(state, oauthStatePrefix) {
		return state, "", true
	}
	rest := strings.TrimPrefix(state, oauthStatePrefix)
	dot := strings.Index(rest, ".")
	if dot <= 0 || dot >= len(rest)-1 {
		return "", "", false
	}
	nonce = rest[:dot]
	enc := rest[dot+1:]
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return "", "", false
	}
	localReturn = strings.TrimSpace(string(raw))
	if nonce == "" || localReturn == "" {
		return "", "", false
	}
	return nonce, localReturn, true
}

// NewOAuthNonce returns a random hex nonce for OAuth state tracking on the hub.
func NewOAuthNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// OAuthStateNonce returns the hub validation key for a Slack state parameter.
func OAuthStateNonce(state string) (string, bool) {
	nonce, _, ok := ParseOAuthState(state)
	if !ok {
		return "", false
	}
	return nonce, true
}

// BuildRelayRedirectURL constructs the loopback callback URL with Slack query params preserved.
func BuildRelayRedirectURL(localCallback string, query url.Values) (string, error) {
	u, err := url.Parse(strings.TrimSpace(localCallback))
	if err != nil {
		return "", err
	}
	if !IsAllowedLocalOAuthCallback(u) {
		return "", ErrRelayDisallowedCallback
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}
