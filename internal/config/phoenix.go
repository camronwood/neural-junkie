package config

import "strings"

// PhoenixConfig holds native TIM import settings (customer pack overlay).
type PhoenixConfig struct {
	Environment     string `json:"environment,omitempty"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	AuthConfigPath  string `json:"auth_config_path,omitempty"`
}

func (c *Config) PhoenixSettings() PhoenixConfig {
	if c == nil {
		return PhoenixConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Phoenix
}

func (p PhoenixConfig) EnvironmentOrDefault() string {
	if s := strings.TrimSpace(p.Environment); s != "" {
		return s
	}
	return "staging"
}

func (p PhoenixConfig) ToImportSettings() (environment, credentialsPath, authConfigPath string) {
	return p.EnvironmentOrDefault(), strings.TrimSpace(p.CredentialsPath), strings.TrimSpace(p.AuthConfigPath)
}
