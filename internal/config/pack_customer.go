package config

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// biologyOverlayFields maps settings_overlay keys to biology MCP config fields.
var biologyOverlayFields = map[string]func(*BiologyMCPConfig) *string{
	"secondary_analysis_tools_path": func(b *BiologyMCPConfig) *string { return &b.SecondaryAnalysisToolsPath },
	"python_executable":             func(b *BiologyMCPConfig) *string { return &b.PythonExecutable },
	"cumulative_qc_dir":             func(b *BiologyMCPConfig) *string { return &b.CumulativeQCDir },
	"default_panel_profile":         func(b *BiologyMCPConfig) *string { return &b.DefaultPanelProfile },
	"artifacts_dir":                 func(b *BiologyMCPConfig) *string { return &b.ArtifactsDir },
}

var awsOverlayFields = map[string]func(*AWSConfig) *string{
	"aws_default_region": func(a *AWSConfig) *string { return &a.DefaultRegion },
	"aws_profile":        func(a *AWSConfig) *string { return &a.Profile },
	"aws_sso_start_url":  func(a *AWSConfig) *string { return &a.SSOStartURL },
}

var jiraOverlayFields = map[string]func(*JiraConfig) *string{
	"jira_base_url":            func(j *JiraConfig) *string { return &j.BaseURL },
	"jira_email":               func(j *JiraConfig) *string { return &j.Email },
	"jira_api_token":           func(j *JiraConfig) *string { return &j.APIToken },
	"jira_default_project_key": func(j *JiraConfig) *string { return &j.DefaultProjectKey },
}

var incidentOverlayFields = map[string]func(*IncidentConfig) *string{
	"incident_default_provider": func(i *IncidentConfig) *string { return &i.DefaultProvider },
}

var phoenixOverlayFields = map[string]func(*PhoenixConfig) *string{
	"phoenix_environment":     func(p *PhoenixConfig) *string { return &p.Environment },
	"environment":             func(p *PhoenixConfig) *string { return &p.Environment },
	"phoenix_credentials_path": func(p *PhoenixConfig) *string { return &p.CredentialsPath },
	"credentials_path":        func(p *PhoenixConfig) *string { return &p.CredentialsPath },
	"phoenix_auth_config_path": func(p *PhoenixConfig) *string { return &p.AuthConfigPath },
	"auth_config_path":        func(p *PhoenixConfig) *string { return &p.AuthConfigPath },
}

func normalizeOverlayKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "mcp.biology.")
	return key
}

func (c *Config) validatePackRequirementsLocked(packID string) error {
	m, err := c.installedManifestLocked(packID)
	if err != nil {
		return err
	}
	for _, req := range m.RequiresPacks {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		if !c.packInstalledLocked(req) {
			return fmt.Errorf("install pack %q before enabling %q", req, packID)
		}
		if !c.packEnabledLocked(req) {
			return fmt.Errorf("enable pack %q before enabling %q", req, packID)
		}
	}
	return nil
}

func (c *Config) applyPackSettingsOverlayLocked(packID string) error {
	m, err := c.installedManifestLocked(packID)
	if err != nil {
		return err
	}
	if len(m.SettingsOverlay) == 0 {
		return nil
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		return err
	}
	resolved, err := packs.ResolveSettingsOverlay(m, dir)
	if err != nil {
		return err
	}
	if c.Packs.AppliedOverlays == nil {
		c.Packs.AppliedOverlays = make(map[string]map[string]string)
	}
	prev := make(map[string]string)
	for rawKey, val := range resolved {
		key := normalizeOverlayKey(rawKey)
		if setter, ok := biologyOverlayFields[key]; ok {
			field := setter(&c.MCP.Biology)
			prev["biology:"+key] = *field
			*field = val
			continue
		}
		if setter, ok := phoenixOverlayFields[key]; ok {
			field := setter(&c.Phoenix)
			prev["phoenix:"+key] = *field
			*field = val
			continue
		}
		if setter, ok := awsOverlayFields[key]; ok {
			field := setter(&c.AWS)
			prev["aws:"+key] = *field
			*field = val
			continue
		}
		if setter, ok := jiraOverlayFields[key]; ok {
			field := setter(&c.Jira)
			prev["jira:"+key] = *field
			*field = val
			continue
		}
		if setter, ok := incidentOverlayFields[key]; ok {
			field := setter(&c.Incident)
			prev["incident:"+key] = *field
			*field = val
			continue
		}
		switch key {
		case "incident_write_mode":
			prev["incident:write_mode"] = fmt.Sprintf("%v", c.Incident.WriteMode != nil && *c.Incident.WriteMode)
			b := parseOverlayBool(val, false)
			c.Incident.WriteMode = &b
		case "incident_require_approval":
			prev["incident:require_approval"] = fmt.Sprintf("%v", c.Incident.RequireApproval != nil && *c.Incident.RequireApproval)
			b := parseOverlayBool(val, true)
			c.Incident.RequireApproval = &b
		}
	}
	if len(prev) > 0 {
		c.Packs.AppliedOverlays[packID] = prev
	}
	return nil
}

func (c *Config) revertPackSettingsOverlayLocked(packID string) {
	if c.Packs.AppliedOverlays == nil {
		return
	}
	prev, ok := c.Packs.AppliedOverlays[packID]
	if !ok {
		return
	}
	for key, val := range prev {
		if strings.HasPrefix(key, "biology:") {
			k := strings.TrimPrefix(key, "biology:")
			if setter, ok := biologyOverlayFields[k]; ok {
				field := setter(&c.MCP.Biology)
				*field = val
			}
			continue
		}
		if strings.HasPrefix(key, "phoenix:") {
			k := strings.TrimPrefix(key, "phoenix:")
			if setter, ok := phoenixOverlayFields[k]; ok {
				field := setter(&c.Phoenix)
				*field = val
			}
			continue
		}
		if strings.HasPrefix(key, "aws:") {
			k := strings.TrimPrefix(key, "aws:")
			if setter, ok := awsOverlayFields[k]; ok {
				field := setter(&c.AWS)
				*field = val
			}
			continue
		}
		if strings.HasPrefix(key, "jira:") {
			k := strings.TrimPrefix(key, "jira:")
			if setter, ok := jiraOverlayFields[k]; ok {
				field := setter(&c.Jira)
				*field = val
			}
			continue
		}
		if strings.HasPrefix(key, "incident:") {
			k := strings.TrimPrefix(key, "incident:")
			if setter, ok := incidentOverlayFields[k]; ok {
				field := setter(&c.Incident)
				*field = val
				continue
			}
			switch k {
			case "write_mode":
				b := parseOverlayBool(val, false)
				c.Incident.WriteMode = &b
			case "require_approval":
				b := parseOverlayBool(val, true)
				c.Incident.RequireApproval = &b
			}
		}
	}
	delete(c.Packs.AppliedOverlays, packID)
}

func (c *Config) installedManifestLocked(packID string) (*packs.Manifest, error) {
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		return nil, err
	}
	m, err := packs.LoadManifest(dir)
	if err != nil {
		return nil, fmt.Errorf("pack %q: %w", packID, err)
	}
	return m, nil
}

// InstallPackFromZip installs a customer pack from a zip archive (does not enable).
func (c *Config) InstallPackFromZip(data []byte) (*packs.Manifest, error) {
	m, err := packs.InstallFromZipBytes(data)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	if !c.packInstalledLocked(m.ID) {
		c.Packs.Installed = append(c.Packs.Installed, m.ID)
	}
	c.Packs.Enabled[m.ID] = false
	return m, nil
}

// CustomerPackContext is returned for enabled customer packs.
type CustomerPackContext struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Publisher       string            `json:"publisher,omitempty"`
	Version         string            `json:"version,omitempty"`
	RequiresPacks   []string          `json:"requires_packs,omitempty"`
	WorkspaceGuide  string            `json:"workspace_guide,omitempty"`
	SettingsOverlay map[string]string `json:"settings_overlay,omitempty"`
}

// EnabledCustomerPackContexts returns workspace guides and metadata for enabled customer packs.
func (c *Config) EnabledCustomerPackContexts() ([]CustomerPackContext, error) {
	if c == nil {
		return nil, nil
	}
	c.mu.RLock()
	ids := append([]string(nil), c.Packs.Installed...)
	c.mu.RUnlock()
	var out []CustomerPackContext
	for _, id := range ids {
		if !c.IsPackEnabled(id) {
			continue
		}
		m, err := c.InstalledPackManifestByID(id)
		if err != nil || m == nil || !m.IsCustomerPack() {
			continue
		}
		dir, err := packs.InstalledPackDir(id)
		if err != nil {
			continue
		}
		guide, _ := packs.ReadWorkspaceGuide(m, dir)
		overlay, _ := packs.ResolveSettingsOverlay(m, dir)
		out = append(out, CustomerPackContext{
			ID:              m.ID,
			Title:           m.Title,
			Publisher:       m.Publisher,
			Version:         m.Version,
			RequiresPacks:   append([]string(nil), m.RequiresPacks...),
			WorkspaceGuide:  guide,
			SettingsOverlay: overlay,
		})
	}
	return out, nil
}

// IsCatalogPackID reports whether id appears in the official pack catalog.
func IsCatalogPackID(id string) bool {
	cat, err := packs.FetchCatalog()
	if err != nil || cat == nil {
		return packs.IsOfficialPackID(id)
	}
	return cat.CatalogEntryByID(id) != nil
}

func parseOverlayBool(val string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}
