package config

import "strings"

// IncidentConfig holds incident-management pack settings.
type IncidentConfig struct {
	DefaultProvider  string `json:"default_provider,omitempty"`
	WriteMode        *bool  `json:"write_mode,omitempty"`
	RequireApproval  *bool  `json:"require_approval,omitempty"`
}

func (c *Config) IncidentSettings() IncidentConfig {
	if c == nil {
		return IncidentConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Incident
}

func (i IncidentConfig) DefaultProviderOr(defaultVal string) string {
	if p := strings.TrimSpace(i.DefaultProvider); p != "" {
		return p
	}
	return defaultVal
}

func (i IncidentConfig) WriteModeEnabled() bool {
	if i.WriteMode == nil {
		return false
	}
	return *i.WriteMode
}

func (i IncidentConfig) RequireApprovalEnabled() bool {
	if i.RequireApproval == nil {
		return true
	}
	return *i.RequireApproval
}

// GitHubIssuesConfig holds GitHub Issues integration settings.
type GitHubIssuesConfig struct {
	Token       string `json:"token,omitempty"`
	DefaultRepo string `json:"default_repo,omitempty"`
}

func (c *Config) GitHubIssuesSettings() GitHubIssuesConfig {
	if c == nil {
		return GitHubIssuesConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GitHubIssues
}

func (g GitHubIssuesConfig) Configured() bool {
	return strings.TrimSpace(g.Token) != "" && strings.TrimSpace(g.DefaultRepo) != ""
}

// LinearConfig holds Linear integration settings.
type LinearConfig struct {
	APIKey        string `json:"api_key,omitempty"`
	DefaultTeamID string `json:"default_team_id,omitempty"`
}

func (c *Config) LinearSettings() LinearConfig {
	if c == nil {
		return LinearConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Linear
}

func (l LinearConfig) Configured() bool {
	return strings.TrimSpace(l.APIKey) != ""
}

// PagerDutyConfig holds PagerDuty integration settings.
type PagerDutyConfig struct {
	APIKey           string `json:"api_key,omitempty"`
	DefaultServiceID string `json:"default_service_id,omitempty"`
}

func (c *Config) PagerDutySettings() PagerDutyConfig {
	if c == nil {
		return PagerDutyConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.PagerDuty
}

func (p PagerDutyConfig) Configured() bool {
	return strings.TrimSpace(p.APIKey) != ""
}

// SentryConfig holds Sentry integration settings.
type SentryConfig struct {
	AuthToken      string `json:"auth_token,omitempty"`
	DefaultOrg     string `json:"default_org,omitempty"`
	DefaultProject string `json:"default_project,omitempty"`
}

func (c *Config) SentrySettings() SentryConfig {
	if c == nil {
		return SentryConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Sentry
}

func (s SentryConfig) Configured() bool {
	return strings.TrimSpace(s.AuthToken) != "" && strings.TrimSpace(s.DefaultOrg) != ""
}
