package config

import "strings"

// JiraConfig holds Jira Cloud integration settings (incident-management pack).
type JiraConfig struct {
	BaseURL           string `json:"base_url,omitempty"`
	Email             string `json:"email,omitempty"`
	APIToken          string `json:"api_token,omitempty"`
	DefaultProjectKey string `json:"default_project_key,omitempty"`
}

func (c *Config) JiraSettings() JiraConfig {
	if c == nil {
		return JiraConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Jira
}

func (j JiraConfig) BaseURLTrimmed() string {
	return strings.TrimRight(strings.TrimSpace(j.BaseURL), "/")
}

func (j JiraConfig) Configured() bool {
	return j.BaseURLTrimmed() != "" && strings.TrimSpace(j.Email) != "" && strings.TrimSpace(j.APIToken) != ""
}
