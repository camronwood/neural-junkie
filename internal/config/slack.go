package config

// SlackPolicy controls when the assigned agent is triggered for Slack messages.
type SlackPolicy string

const (
	SlackPolicyMentionOnly SlackPolicy = "mention_only"
	SlackPolicyQuestions   SlackPolicy = "questions"
	SlackPolicyAlways      SlackPolicy = "always"
)

// SlackConfig configures the in-process Slack bridge.
type SlackConfig struct {
	Enabled        bool        `json:"enabled"`
	AppToken       string      `json:"app_token,omitempty"`  // xapp-... Socket Mode
	BotToken       string      `json:"bot_token,omitempty"`  // xoxb-...
	ClientID       string      `json:"client_id,omitempty"`  // OAuth app (optional)
	ClientSecret   string      `json:"client_secret,omitempty"`
	RedirectURL    string      `json:"redirect_url,omitempty"`
	DisplayName    string      `json:"display_name,omitempty"` // chat.postMessage username
	DisplayIconURL string      `json:"display_icon_url,omitempty"`
	DefaultPolicy  SlackPolicy `json:"default_policy,omitempty"`
	ForceDisabled  bool        `json:"force_disabled,omitempty"`
	Debug          bool        `json:"debug,omitempty"`
	OAuthRelayBase string      `json:"oauth_relay_base,omitempty"`
	UseOAuthRelay  *bool       `json:"use_oauth_relay,omitempty"`
}

// EffectiveDisplayName returns the Slack customize username.
func (s *SlackConfig) EffectiveDisplayName() string {
	if s == nil || s.DisplayName == "" {
		return "Camron"
	}
	return s.DisplayName
}

// EffectiveDefaultPolicy returns the binding default when unset.
func (s *SlackConfig) EffectiveDefaultPolicy() SlackPolicy {
	if s == nil || s.DefaultPolicy == "" {
		return SlackPolicyMentionOnly
	}
	return s.DefaultPolicy
}

// SlackDisabledByEnv reports whether live Slack (Socket Mode) must stay off.
func SlackDisabledByEnv() bool {
	return AppConfig().ResolvedSlackDisabled()
}

// SlackReady reports whether Socket Mode can start.
func (s *SlackConfig) SlackReady() bool {
	if SlackDisabledByEnv() {
		return false
	}
	return s != nil && s.Enabled && s.AppToken != "" && s.BotToken != ""
}
