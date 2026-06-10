package slack

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

// OAuthSource describes where Slack OAuth credentials were resolved from.
type OAuthSource string

const (
	OAuthSourceNone    OAuthSource = "none"
	OAuthSourceBundled OAuthSource = "bundled"
	OAuthSourceUser    OAuthSource = "user"
	OAuthSourceEnv     OAuthSource = "env"
	OAuthSourceConfig  OAuthSource = "config"
)

// bundledVendorCredentials is the shape of vendor/oauth.json (and .example).
type bundledVendorCredentials struct {
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	AppToken       string `json:"app_token"`
	OAuthRelayBase string `json:"oauth_relay_base,omitempty"`
}

var (
	hubBaseMu        sync.RWMutex
	hubPublicBaseURL string
)

// SetHubPublicBaseURL sets the public HTTP base URL for OAuth redirects (e.g. http://localhost:18765).
func SetHubPublicBaseURL(base string) {
	hubBaseMu.Lock()
	defer hubBaseMu.Unlock()
	hubPublicBaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
}

// HubPublicBaseURL returns the configured hub base URL.
func HubPublicBaseURL() string {
	hubBaseMu.RLock()
	defer hubBaseMu.RUnlock()
	return hubPublicBaseURL
}

// OAuthCallbackPath is the Slack OAuth redirect path on the hub.
const OAuthCallbackPath = "/api/slack/oauth/callback"

// UserDMOAuthCallbackPath is the redirect path for human-DM user token OAuth.
const UserDMOAuthCallbackPath = "/api/slack/oauth/user-dm/callback"

// HubOAuthRedirectURL builds the redirect_uri for Slack OAuth from the hub base URL.
func HubOAuthRedirectURL() string {
	base := HubPublicBaseURL()
	if base == "" {
		return ""
	}
	return base + OAuthCallbackPath
}

// HubUserDMOAuthRedirectURL builds the redirect_uri for user-scope OAuth.
func HubUserDMOAuthRedirectURL() string {
	base := HubPublicBaseURL()
	if base == "" {
		return ""
	}
	return base + UserDMOAuthCallbackPath
}

func parseBundledVendor() (*bundledVendorCredentials, bool) {
	if len(vendorOAuthJSON) == 0 {
		return nil, false
	}
	var v bundledVendorCredentials
	if err := json.Unmarshal(vendorOAuthJSON, &v); err != nil {
		return nil, false
	}
	if !bundledVendorValid(&v) {
		return nil, false
	}
	return &v, true
}

func bundledVendorValid(v *bundledVendorCredentials) bool {
	if v == nil {
		return false
	}
	cid := strings.TrimSpace(v.ClientID)
	secret := strings.TrimSpace(v.ClientSecret)
	if cid == "" || secret == "" {
		return false
	}
	if strings.Contains(cid, "YOUR_") || strings.Contains(secret, "YOUR_") {
		return false
	}
	return true
}

func oauthFromBundled() (*OAuthAppCredentials, OAuthSource) {
	v, ok := parseBundledVendor()
	if !ok {
		return nil, OAuthSourceNone
	}
	return &OAuthAppCredentials{
		ClientID:     strings.TrimSpace(v.ClientID),
		ClientSecret: strings.TrimSpace(v.ClientSecret),
		RedirectURL:  resolveBundledBotRedirectURL(),
	}, OAuthSourceBundled
}

func resolveBundledBotRedirectURL() string {
	return ResolveBotOAuthRedirectURL(slackRedirectHints{})
}

func oauthFromEnv() (*OAuthAppCredentials, OAuthSource) {
	cid := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_CLIENT_SECRET"))
	if cid == "" || secret == "" {
		return nil, OAuthSourceNone
	}
	return &OAuthAppCredentials{
		ClientID:     cid,
		ClientSecret: secret,
		RedirectURL:  ResolveBotOAuthRedirectURL(slackRedirectHints{envRedirect: os.Getenv("NEURAL_JUNKIE_SLACK_REDIRECT_URL")}),
	}, OAuthSourceEnv
}

func oauthFromConfig(cfg *config.SlackConfig) (*OAuthAppCredentials, OAuthSource) {
	if cfg == nil {
		return nil, OAuthSourceNone
	}
	cid := strings.TrimSpace(cfg.ClientID)
	secret := strings.TrimSpace(cfg.ClientSecret)
	if cid == "" || secret == "" {
		return nil, OAuthSourceNone
	}
	return &OAuthAppCredentials{
		ClientID:     cid,
		ClientSecret: secret,
		RedirectURL:  ResolveBotOAuthRedirectURL(slackRedirectHints{configRedirect: strings.TrimSpace(cfg.RedirectURL)}),
	}, OAuthSourceConfig
}

// ResolveOAuthApp returns OAuth app credentials using: user file → bundled → env → config.
// Redirect URL defaults to the hub public base unless the user file sets redirect_url.
func ResolveOAuthApp(cfg *config.SlackConfig) (*OAuthAppCredentials, OAuthSource) {
	user, err := LoadOAuthApp()
	if err == nil && user != nil && strings.TrimSpace(user.ClientID) != "" && strings.TrimSpace(user.ClientSecret) != "" {
		out := *user
		out.RedirectURL = ResolveBotOAuthRedirectURL(slackRedirectHints{userRedirect: out.RedirectURL})
		return &out, OAuthSourceUser
	}
	if o, src := oauthFromBundled(); o != nil {
		return o, src
	}
	if o, src := oauthFromEnv(); o != nil {
		return o, src
	}
	if o, src := oauthFromConfig(cfg); o != nil {
		return o, src
	}
	return nil, OAuthSourceNone
}

// ResolveAppToken returns the Socket Mode app token: config → bundled → env.
func ResolveAppToken(cfg *config.SlackConfig) (token string, source OAuthSource) {
	if cfg != nil {
		if t := strings.TrimSpace(cfg.AppToken); t != "" {
			return t, OAuthSourceConfig
		}
	}
	if v, ok := parseBundledVendor(); ok {
		if t := strings.TrimSpace(v.AppToken); t != "" && !strings.Contains(t, "YOUR_") {
			return t, OAuthSourceBundled
		}
	}
	if t := strings.TrimSpace(os.Getenv("NEURAL_JUNKIE_SLACK_APP_TOKEN")); t != "" {
		return t, OAuthSourceEnv
	}
	return "", OAuthSourceNone
}

func oauthRedirectHints(cfg *config.SlackConfig) slackRedirectHints {
	hints := slackRedirectHints{envRedirect: os.Getenv("NEURAL_JUNKIE_SLACK_REDIRECT_URL")}
	if cfg != nil {
		hints.configRedirect = strings.TrimSpace(cfg.RedirectURL)
	}
	if user, err := LoadOAuthApp(); err == nil && user != nil {
		hints.userRedirect = strings.TrimSpace(user.RedirectURL)
	}
	return hints
}

// ResolveUserDMOAuthRedirectFromConfig picks user-scope redirect_uri using the same override order as bot OAuth.
func ResolveUserDMOAuthRedirectFromConfig(cfg *config.SlackConfig) string {
	return ResolveUserDMOAuthRedirectURL(oauthRedirectHints(cfg))
}

// OAuthReady reports whether Connect Slack can start OAuth.
func OAuthReady(cfg *config.SlackConfig) bool {
	o, _ := ResolveOAuthApp(cfg)
	return o != nil && o.ClientID != "" && o.ClientSecret != ""
}

// PublicOAuthFromResolved builds API-safe OAuth config for the desktop.
func PublicOAuthFromResolved(cfg *config.SlackConfig) PublicOAuthConfig {
	o, src := ResolveOAuthApp(cfg)
	if o == nil || o.ClientID == "" {
		return PublicOAuthConfig{OAuthSource: string(OAuthSourceNone)}
	}
	return PublicOAuthConfig{
		ClientID:       o.ClientID,
		RedirectURL:    o.RedirectURL,
		OAuthRelayBase: OAuthRelayBase(),
		UsesOAuthRelay: IsRelayRedirectURL(o.RedirectURL),
		SecretSet:      o.ClientSecret != "",
		Configured:     true,
		ConnectReady:   true,
		OAuthSource:    string(src),
	}
}

// SeedBundledAppToken copies bundled xapp into cfg when cfg has no app token yet.
func SeedBundledAppToken(cfg *config.SlackConfig) bool {
	if cfg == nil || strings.TrimSpace(cfg.AppToken) != "" {
		return false
	}
	tok, src := ResolveAppToken(cfg)
	if src != OAuthSourceBundled || tok == "" {
		return false
	}
	cfg.AppToken = tok
	return true
}
