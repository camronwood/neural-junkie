package config

import "strings"

const DefaultAWSRegion = "us-east-2"

// AWSConfig holds AWS SSO profile integration settings (AWS pack).
type AWSConfig struct {
	DefaultRegion   string   `json:"default_region,omitempty"`
	Profile         string   `json:"profile,omitempty"`
	AllowedProfiles []string `json:"allowed_profiles,omitempty"`
	SSOStartURL     string   `json:"sso_start_url,omitempty"`
	ReadOnly        *bool    `json:"read_only,omitempty"`
}

func (c *Config) AWSSettings() AWSConfig {
	if c == nil {
		return AWSConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AWS
}

func (a AWSConfig) DefaultRegionOrDefault() string {
	if r := strings.TrimSpace(a.DefaultRegion); r != "" {
		return r
	}
	return DefaultAWSRegion
}

func (a AWSConfig) ProfileOrDefault() string {
	return strings.TrimSpace(a.Profile)
}

func (a AWSConfig) ReadOnlyEnabled() bool {
	if a.ReadOnly == nil {
		return true
	}
	return *a.ReadOnly
}

func (a AWSConfig) ProfileAllowed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if len(a.AllowedProfiles) == 0 {
		return true
	}
	for _, p := range a.AllowedProfiles {
		if strings.TrimSpace(p) == name {
			return true
		}
	}
	return false
}
