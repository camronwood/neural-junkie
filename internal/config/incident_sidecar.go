package config

import (
	"strconv"
	"strings"
)

// IncidentSidecarSettings returns overlay keys merged into the incident pack sidecar env.
func (c *Config) IncidentSidecarSettings() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.incidentSidecarSettingsLocked()
}

func (c *Config) incidentSidecarSettingsLocked() map[string]string {
	out := map[string]string{
		"incident_default_provider": c.Incident.DefaultProviderOr("jira"),
		"incident_write_mode":       strconv.FormatBool(c.Incident.WriteModeEnabled()),
		"incident_require_approval": strconv.FormatBool(c.Incident.RequireApprovalEnabled()),
	}
	j := c.Jira
	if v := j.BaseURLTrimmed(); v != "" {
		out["jira_base_url"] = v
	}
	if v := strings.TrimSpace(j.Email); v != "" {
		out["jira_email"] = v
	}
	if v := strings.TrimSpace(j.APIToken); v != "" {
		out["jira_api_token"] = v
	}
	if v := strings.TrimSpace(j.DefaultProjectKey); v != "" {
		out["jira_default_project_key"] = v
	}
	g := c.GitHubIssues
	if v := strings.TrimSpace(g.Token); v != "" {
		out["github_token"] = v
	}
	if v := strings.TrimSpace(g.DefaultRepo); v != "" {
		out["github_default_repo"] = v
	}
	l := c.Linear
	if v := strings.TrimSpace(l.APIKey); v != "" {
		out["linear_api_key"] = v
	}
	if v := strings.TrimSpace(l.DefaultTeamID); v != "" {
		out["linear_default_team_id"] = v
	}
	p := c.PagerDuty
	if v := strings.TrimSpace(p.APIKey); v != "" {
		out["pagerduty_api_key"] = v
	}
	if v := strings.TrimSpace(p.DefaultServiceID); v != "" {
		out["pagerduty_default_service_id"] = v
	}
	s := c.Sentry
	if v := strings.TrimSpace(s.AuthToken); v != "" {
		out["sentry_auth_token"] = v
	}
	if v := strings.TrimSpace(s.DefaultOrg); v != "" {
		out["sentry_default_org"] = v
	}
	if v := strings.TrimSpace(s.DefaultProject); v != "" {
		out["sentry_default_project"] = v
	}
	return out
}
