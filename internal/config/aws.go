package config

import "strings"

const DefaultAWSRegion = "us-east-2"

// AWSConfig holds AWS SSO profile integration settings (AWS pack).
type AWSConfig struct {
	DefaultRegion   string   `json:"default_region,omitempty"`
	Profile         string   `json:"profile,omitempty"`
	AllowedProfiles []string `json:"allowed_profiles,omitempty"`
	AllowedAccounts []string `json:"allowed_accounts,omitempty"`
	OrgRootID       string   `json:"org_root_id,omitempty"`
	SSOStartURL     string   `json:"sso_start_url,omitempty"`
	ReadOnly        *bool    `json:"read_only,omitempty"`
	WriteEnabled    *bool    `json:"write_enabled,omitempty"`
	WriteAuditPath  string   `json:"write_audit_path,omitempty"`
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

func (a AWSConfig) WriteEnabledFlag() bool {
	if a.WriteEnabled == nil {
		return false
	}
	return *a.WriteEnabled
}

func (a AWSConfig) WriteAuditPathOrDefault() string {
	if p := strings.TrimSpace(a.WriteAuditPath); p != "" {
		return p
	}
	return "~/.neural-junkie/aws-audit.log"
}

func (a AWSConfig) AccountAllowed(accountID string) bool {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return true
	}
	if len(a.AllowedAccounts) == 0 {
		return true
	}
	for _, id := range a.AllowedAccounts {
		if strings.TrimSpace(id) == accountID {
			return true
		}
	}
	return false
}
