package slack

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
)

// DefaultOAuthRelayBase is the public HTTPS OAuth relay for distributed Slack installs.
// Override with NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE or oauth_relay_base in vendor/oauth.json.
// DefaultOAuthRelayBase is the Cloudflare Workers origin after `make slack-oauth-relay-deploy-cf`.
// Format: https://nj-slack-oauth-relay.<your-cf-subdomain>.workers.dev
// Override via SLACK_VENDOR_OAUTH_RELAY_BASE / oauth_relay_base in vendor/oauth.json.
const DefaultOAuthRelayBase = "https://nj-slack-oauth-relay.neuraljunkie.workers.dev"

// ErrRelayDisallowedCallback is returned when a relay redirect targets a non-loopback URL.
var ErrRelayDisallowedCallback = errors.New("slack oauth relay: callback must be loopback http")

var allowedLocalCallbackPaths = map[string]bool{
	OAuthCallbackPath:       true,
	UserDMOAuthCallbackPath: true,
}

type slackRedirectHints struct {
	userRedirect   string
	configRedirect string
	envRedirect    string
}

// OAuthRelayBase returns the configured public HTTPS relay origin (no trailing slash).
func OAuthRelayBase() string {
	if v := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v, ok := bundledOAuthRelayBase(); ok {
		return v
	}
	return DefaultOAuthRelayBase
}

func bundledOAuthRelayBase() (string, bool) {
	v, ok := parseBundledVendor()
	if !ok || v == nil {
		return "", false
	}
	base := strings.TrimRight(strings.TrimSpace(v.OAuthRelayBase), "/")
	if base == "" || !strings.HasPrefix(base, "https://") {
		return "", false
	}
	return base, true
}

// PublicRelayBotCallbackURL is the HTTPS redirect_uri registered in the Slack app for bot OAuth.
func PublicRelayBotCallbackURL() string {
	return OAuthRelayBase() + OAuthCallbackPath
}

// PublicRelayUserDMCallbackURL is the HTTPS redirect_uri for user-scope (human DM) OAuth.
func PublicRelayUserDMCallbackURL() string {
	return OAuthRelayBase() + UserDMOAuthCallbackPath
}

func resolveExplicitRedirect(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return ""
	}
	// Saved loopback URLs (oauth_app.json, config) must upgrade to HTTPS relay for public Slack distribution.
	if relayPreferred() && isLoopbackURL(r) {
		return PublicRelayBotCallbackURL()
	}
	return r
}

func resolveExplicitUserDMRedirect(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return ""
	}
	if relayPreferred() && isLoopbackURL(r) {
		return PublicRelayUserDMCallbackURL()
	}
	return r
}

// ResolveBotOAuthRedirectURL picks redirect_uri for bot OAuth (Slack authorize + token exchange).
func ResolveBotOAuthRedirectURL(hints slackRedirectHints) string {
	for _, raw := range []string{hints.userRedirect, hints.envRedirect, hints.configRedirect} {
		if r := resolveExplicitRedirect(raw); r != "" {
			return r
		}
	}
	if hub := HubOAuthRedirectURL(); hub != "" && isLoopbackURL(hub) && !relayPreferred() {
		return hub
	}
	return PublicRelayBotCallbackURL()
}

// ResolveUserDMOAuthRedirectURL picks redirect_uri for user-scope OAuth.
func ResolveUserDMOAuthRedirectURL(hints slackRedirectHints) string {
	for _, raw := range []string{hints.userRedirect, hints.envRedirect, hints.configRedirect} {
		if r := resolveExplicitUserDMRedirect(raw); r != "" {
			return r
		}
	}
	if hub := HubUserDMOAuthRedirectURL(); hub != "" && isLoopbackURL(hub) && !relayPreferred() {
		return hub
	}
	return PublicRelayUserDMCallbackURL()
}

// LocalBotOAuthCallbackURL is where the hub receives the OAuth code after the relay forwards the browser.
func LocalBotOAuthCallbackURL() string {
	if u := HubOAuthRedirectURL(); u != "" {
		return u
	}
	return "http://127.0.0.1:18765" + OAuthCallbackPath
}

// LocalUserDMOAuthCallbackURL is the loopback callback for user-scope OAuth.
func LocalUserDMOAuthCallbackURL() string {
	if u := HubUserDMOAuthRedirectURL(); u != "" {
		return u
	}
	return "http://127.0.0.1:18765" + UserDMOAuthCallbackPath
}

// IsRelayRedirectURL reports whether redirect_uri is the public HTTPS relay (not loopback).
func IsRelayRedirectURL(redirect string) bool {
	redirect = strings.TrimRight(strings.TrimSpace(redirect), "/")
	base := strings.TrimRight(OAuthRelayBase(), "/")
	return redirect == base+OAuthCallbackPath || redirect == base+UserDMOAuthCallbackPath
}

// IsAllowedLocalOAuthCallback reports whether u is a safe loopback target for the relay.
func IsAllowedLocalOAuthCallback(u *url.URL) bool {
	if u == nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host != "127.0.0.1" && host != "localhost" {
		return false
	}
	if port := u.Port(); port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return false
		}
	}
	path := u.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return allowedLocalCallbackPaths[path]
}

func relayPreferred() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_USE_OAUTH_RELAY")), "0") {
		return false
	}
	if strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE")) != "" {
		return true
	}
	if _, ok := bundledOAuthRelayBase(); ok {
		return true
	}
	if v, ok := parseBundledVendor(); ok && bundledVendorValid(v) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_USE_OAUTH_RELAY")), "1") {
		return true
	}
	// Default on: public HTTPS relay is required for Slack public distribution.
	return strings.HasPrefix(DefaultOAuthRelayBase, "https://")
}

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
