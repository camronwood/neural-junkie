package config

import (
	"strconv"
	"strings"
)

// AWSSidecarSettings returns overlay key/value pairs merged into the AWS sidecar env.
func (c *Config) AWSSidecarSettings() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.awsSidecarSettingsLocked()
}

func (c *Config) awsSidecarSettingsLocked() map[string]string {
	a := c.AWS
	out := map[string]string{
		"aws_profile":         a.ProfileOrDefault(),
		"aws_default_region":  a.DefaultRegionOrDefault(),
		"aws_read_only":       strconv.FormatBool(a.ReadOnlyEnabled()),
		"aws_write_enabled":   strconv.FormatBool(a.WriteEnabledFlag()),
		"aws_write_audit_path": a.WriteAuditPathOrDefault(),
	}
	if u := strings.TrimSpace(a.SSOStartURL); u != "" {
		out["aws_sso_start_url"] = u
	}
	if r := strings.TrimSpace(a.OrgRootID); r != "" {
		out["aws_org_root_id"] = r
	}
	if len(a.AllowedProfiles) > 0 {
		out["aws_allowed_profiles"] = strings.Join(a.AllowedProfiles, ",")
	}
	if len(a.AllowedAccounts) > 0 {
		out["aws_allowed_accounts"] = strings.Join(a.AllowedAccounts, ",")
	}
	return out
}
