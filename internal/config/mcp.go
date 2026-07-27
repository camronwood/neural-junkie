package config

import (
	"strings"
)

// MCPConfig controls in-process MCP tool servers for specialist agents.
type MCPConfig struct {
	// Enabled is the master switch (default true in DefaultConfig).
	Enabled bool `json:"enabled"`
	// Agents overrides per agent type (backend, devops, database, biology, ...).
	// When a key is present it overrides pack-driven MCP enablement from mcp_agents in pack.yaml.
	Agents map[string]bool `json:"agents,omitempty"`
	// Ports optional per agent type (backend, devops, …).
	Ports map[string]int `json:"ports,omitempty"`
	// Biology holds life-sciences MCP tool limits (ESMFold, sequence analysis).
	Biology BiologyMCPConfig `json:"biology"`
	// CAD holds OpenSCAD render settings for the CAD pack.
	CAD CadMCPConfig `json:"cad"`
	// Music holds ACE-Step generation settings for the Music creation pack.
	Music MusicMCPConfig `json:"music"`
	// UserServers are user-connected remote MCP servers (URL/stdio registry entries).
	// Connection wiring is a future enhancement; today this is a persisted registry
	// that Settings can list/manage.
	UserServers []UserMCPServer `json:"user_servers,omitempty"`
	// UserTools are user-defined tools (HTTP-fetch template today) grantable to
	// custom expert agents by name — the "MCP Tool Wizard".
	UserTools []UserMCPTool `json:"user_tools,omitempty"`
	// ExternalMedia optionally wires media_submit/media_status/media_fetch
	// tools to a third-party media generation API. Disabled (BaseURL empty)
	// by default — see internal/mcp/externalmedia.
	ExternalMedia ExternalMediaConfig `json:"external_media,omitempty"`
}

// ExternalMediaConfig configures an optional external media-generation HTTP
// API (e.g. image/video/audio job submission) exposed to granted custom
// expert agents as media_submit / media_status / media_fetch MCP tools.
// BaseURL defaults to empty, which disables the feature entirely — no tools
// are attached until an operator configures a real endpoint.
type ExternalMediaConfig struct {
	// BaseURL is the media API root, e.g. "https://media.example.com/v1".
	// Empty (default) disables the tools.
	BaseURL string `json:"base_url,omitempty"`
	// APIKey is sent as "Authorization: Bearer <APIKey>" when set.
	APIKey string `json:"api_key,omitempty"`
	// GrantedAgents lists custom expert agent display names (case-insensitive)
	// allowed to call the media tools.
	GrantedAgents []string `json:"granted_agents,omitempty"`
}

// Enabled reports whether the external media tools should be attached at all.
func (e ExternalMediaConfig) Enabled() bool {
	return strings.TrimSpace(e.BaseURL) != ""
}

// GrantedTo reports whether agentName (case-insensitive) may use the media tools.
func (e ExternalMediaConfig) GrantedTo(agentName string) bool {
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return false
	}
	for _, g := range e.GrantedAgents {
		if strings.EqualFold(strings.TrimSpace(g), agentName) {
			return true
		}
	}
	return false
}

// ExternalMediaSettings returns a copy of the external media config (thread-safe).
func (c *Config) ExternalMediaSettings() ExternalMediaConfig {
	if c == nil {
		return ExternalMediaConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MCP.ExternalMedia
}

// UserMCPServer is a user-connected remote MCP server registry entry
// (e.g. "read this page/API from my website"). Connection transport is
// recorded but not yet dialed automatically — see MCP Tool Wizard phase 1
// in FUTURE_ENHANCEMENTS.md.
type UserMCPServer struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url,omitempty"`
	TransportType string `json:"transport_type,omitempty"` // "http" (default) | "stdio"
	CreatedAt     string `json:"created_at,omitempty"`
}

// UserMCPTool is a user-created tool (HTTP-fetch template) that can be
// granted to chosen custom expert agents by name. Grants are keyed by agent
// display name (case-insensitive) rather than agent ID, since custom expert
// agent IDs are regenerated across hub restarts but names are stable.
type UserMCPTool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// URL is the HTTP endpoint template this tool fetches. Must be a public
	// http(s) URL — private/loopback hosts are rejected at call time (SSRF gate).
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"` // GET (default) or POST
	Headers map[string]string `json:"headers,omitempty"`
	// JSONPath optionally extracts a nested value from a JSON response using
	// dot-separated keys / numeric array indices (e.g. "data.items.0.title").
	JSONPath string `json:"json_path,omitempty"`
	// GrantedAgents lists custom expert agent display names allowed to call
	// this tool (case-insensitive).
	GrantedAgents []string `json:"granted_agents,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
}

// MethodOrDefault returns the configured HTTP method, defaulting to GET.
func (t UserMCPTool) MethodOrDefault() string {
	m := strings.ToUpper(strings.TrimSpace(t.Method))
	if m == "" {
		return "GET"
	}
	return m
}

// GrantedTo reports whether agentName (case-insensitive) is granted this tool.
func (t UserMCPTool) GrantedTo(agentName string) bool {
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return false
	}
	for _, g := range t.GrantedAgents {
		if strings.EqualFold(strings.TrimSpace(g), agentName) {
			return true
		}
	}
	return false
}

// UserToolsForAgent returns user tools granted to agentName (case-insensitive).
func (c *Config) UserToolsForAgent(agentName string) []UserMCPTool {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []UserMCPTool
	for _, t := range c.MCP.UserTools {
		if t.GrantedTo(agentName) {
			out = append(out, t)
		}
	}
	return out
}

// BiologyMCPConfig is persisted in config.json and edited in Settings.
type BiologyMCPConfig struct {
	ChatModel                  string `json:"chat_model,omitempty"`
	ToolModel                  string `json:"tool_model,omitempty"`
	ESMFoldModel               string `json:"esmfold_model,omitempty"`
	MaxAnalyzeLength           int    `json:"max_analyze_length,omitempty"`
	MaxFoldLength              int    `json:"max_fold_length,omitempty"`
	ArtifactsDir               string `json:"artifacts_dir,omitempty"`
	SecondaryAnalysisToolsPath string `json:"secondary_analysis_tools_path,omitempty"`
	PythonExecutable           string `json:"python_executable,omitempty"`
	CumulativeQCDir            string `json:"cumulative_qc_dir,omitempty"`
	DefaultPanelProfile        string `json:"default_panel_profile,omitempty"`
}

const (
	defaultESMFoldModel     = "facebook/esmfold_v1"
	defaultMaxAnalyzeLength = 10000
	defaultMaxFoldLength    = 400
)

// DefaultMCPConfig returns MCP defaults. Per-agent enablement is driven by enabled pack mcp_agents.
func DefaultMCPConfig() MCPConfig {
	return MCPConfig{
		Enabled: true,
		Biology: BiologyMCPConfig{},
		CAD:     CadMCPConfig{},
		Music:   MusicMCPConfig{ModelVariant: "sft"},
	}
}

func (b BiologyMCPConfig) ESMFoldModelOrDefault() string {
	if m := strings.TrimSpace(b.ESMFoldModel); m != "" {
		return m
	}
	return defaultESMFoldModel
}

func (b BiologyMCPConfig) MaxAnalyzeLengthOrDefault() int {
	if b.MaxAnalyzeLength > 0 {
		return b.MaxAnalyzeLength
	}
	return defaultMaxAnalyzeLength
}

func (b BiologyMCPConfig) MaxFoldLengthOrDefault() int {
	if b.MaxFoldLength > 0 {
		return b.MaxFoldLength
	}
	return defaultMaxFoldLength
}

func (b BiologyMCPConfig) ArtifactsDirOrDefault() string {
	return strings.TrimSpace(b.ArtifactsDir)
}

func (b BiologyMCPConfig) SecondaryAnalysisToolsPathOrDefault() string {
	return strings.TrimSpace(b.SecondaryAnalysisToolsPath)
}

func (b BiologyMCPConfig) PythonExecutableOrDefault() string {
	if p := strings.TrimSpace(b.PythonExecutable); p != "" {
		return p
	}
	return "python3"
}

func (b BiologyMCPConfig) CumulativeQCDirOrDefault() string {
	return strings.TrimSpace(b.CumulativeQCDir)
}

func (b BiologyMCPConfig) DefaultPanelProfileOrDefault() string {
	if p := strings.TrimSpace(b.DefaultPanelProfile); p != "" {
		return p
	}
	return "human-inflammatory-12plex-v1"
}

func (b BiologyMCPConfig) ChatModelOrDefault() string {
	if m := strings.TrimSpace(b.ChatModel); m != "" {
		return m
	}
	return BioOllamaChatModel
}

func (b BiologyMCPConfig) ToolModelOrDefault() string {
	if m := strings.TrimSpace(b.ToolModel); m != "" {
		return m
	}
	return BioOllamaToolModel
}

// BiologyChatModelOrDefault returns mcp.biology.chat_model, then legacy delegation.biology_chat_model.
func (c *Config) BiologyChatModelOrDefault() string {
	if c == nil {
		return BioOllamaChatModel
	}
	if m := strings.TrimSpace(c.BiologyMCPSettings().ChatModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(c.Delegation.Normalized().BiologyChatModel); m != "" {
		return m
	}
	return BioOllamaChatModel
}

// BiologyToolModelOrDefault returns mcp.biology.tool_model, then legacy delegation.biology_tool_model.
func (c *Config) BiologyToolModelOrDefault() string {
	if c == nil {
		return BioOllamaToolModel
	}
	if m := strings.TrimSpace(c.BiologyMCPSettings().ToolModel); m != "" {
		return m
	}
	if m := strings.TrimSpace(c.Delegation.Normalized().BiologyToolModel); m != "" {
		return m
	}
	return BioOllamaToolModel
}

// MigrateBiologyMCPModels copies legacy delegation biology model fields into mcp.biology when unset.
func (c *Config) MigrateBiologyMCPModels() {
	if c == nil {
		return
	}
	d := c.Delegation.Normalized()
	if strings.TrimSpace(c.MCP.Biology.ChatModel) == "" && strings.TrimSpace(c.Delegation.BiologyChatModel) != "" {
		c.MCP.Biology.ChatModel = strings.TrimSpace(d.BiologyChatModel)
	}
	if strings.TrimSpace(c.MCP.Biology.ToolModel) == "" && strings.TrimSpace(c.Delegation.BiologyToolModel) != "" {
		c.MCP.Biology.ToolModel = strings.TrimSpace(d.BiologyToolModel)
	}
}

// CadMCPConfig is persisted in config.json and edited in Settings.
type CadMCPConfig struct {
	OpenSCADPath     string `json:"openscad_path,omitempty"`
	FreeCADPath      string `json:"freecad_path,omitempty"`
	ArtifactsDir     string `json:"artifacts_dir,omitempty"`
	RenderTimeoutSec int    `json:"render_timeout_sec,omitempty"`
	ChatModel        string `json:"chat_model,omitempty"`
	ToolModel        string `json:"tool_model,omitempty"`
}

const defaultCADRenderTimeoutSec = 120

func (c CadMCPConfig) OpenSCADPathOrDefault() string {
	if p := strings.TrimSpace(c.OpenSCADPath); p != "" {
		return p
	}
	return "openscad"
}

func (c CadMCPConfig) FreeCADPathOrDefault() string {
	return strings.TrimSpace(c.FreeCADPath)
}

func (c CadMCPConfig) ArtifactsDirOrDefault() string {
	return strings.TrimSpace(c.ArtifactsDir)
}

func (c CadMCPConfig) RenderTimeoutOrDefault() int {
	if c.RenderTimeoutSec > 0 {
		return c.RenderTimeoutSec
	}
	return defaultCADRenderTimeoutSec
}

func (c CadMCPConfig) ChatModelOrDefault() string {
	if m := strings.TrimSpace(c.ChatModel); m != "" {
		return m
	}
	return CadOllamaChatModel
}

func (c CadMCPConfig) ToolModelOrDefault() string {
	if m := strings.TrimSpace(c.ToolModel); m != "" {
		return m
	}
	return CadOllamaToolModel
}

// mcpAgentConfigKey normalizes agent type strings for MCP config lookup (code-review, backend, …).
func mcpAgentConfigKey(agentType string) string {
	k := strings.ToLower(strings.TrimSpace(agentType))
	k = strings.ReplaceAll(k, "_", "-")
	return k
}

// MCPEnabledForAgent reports whether the MCP server for agentType (BACKEND, biology, …) should run.
func (c *Config) MCPEnabledForAgent(agentType string) bool {
	if c == nil || !c.MCP.Enabled {
		return false
	}
	key := mcpAgentConfigKey(agentType)
	if key == "" {
		return false
	}
	if !c.SpecialistShouldBeRunning(key) {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.MCP.Agents != nil {
		if v, ok := c.MCP.Agents[key]; ok {
			return v
		}
	}
	return c.mcpAgentEnabledByPacksLocked(key)
}

// MCPPort returns the configured port for an agent MCP server or 0 for default.
func (c *Config) MCPPort(agentType string) int {
	if c == nil || c.MCP.Ports == nil {
		return 0
	}
	key := mcpAgentConfigKey(agentType)
	if p, ok := c.MCP.Ports[key]; ok && p > 0 {
		return p
	}
	return 0
}

// BiologyMCPSettings returns a copy of biology MCP settings (thread-safe).
func (c *Config) BiologyMCPSettings() BiologyMCPConfig {
	if c == nil {
		return BiologyMCPConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MCP.Biology
}

// CadMCPSettings returns a copy of CAD MCP settings (thread-safe).
func (c *Config) CadMCPSettings() CadMCPConfig {
	if c == nil {
		return CadMCPConfig{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MCP.CAD
}

// SyncMCPFromPacks updates MCP agent defaults from enabled pack manifests.
func (c *Config) SyncMCPFromPacks() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncMCPFromPacksLocked()
}

func (c *Config) syncMCPFromPacksLocked() {
	if c.MCP.Agents == nil {
		c.MCP.Agents = make(map[string]bool)
	}
	enabled := c.enabledPackMCPAgentTypesLocked()
	for key := range enabled {
		if _, ok := c.MCP.Agents[key]; !ok {
			c.MCP.Agents[key] = true
		}
	}
	allInstalled := c.allInstalledPackMCPAgentTypesLocked()
	for key := range allInstalled {
		if _, stillEnabled := enabled[key]; stillEnabled {
			continue
		}
		if _, ok := c.MCP.Agents[key]; ok {
			delete(c.MCP.Agents, key)
		}
	}
}

func (c *Config) enabledPackMCPAgentTypesLocked() map[string]struct{} {
	out := make(map[string]struct{})
	manifests, _ := c.installedPackManifestsLocked()
	for _, m := range manifests {
		if !c.packEnabledLocked(m.ID) {
			continue
		}
		for _, agentType := range m.MCPAgents {
			key := strings.ToLower(strings.TrimSpace(agentType))
			if key != "" {
				out[key] = struct{}{}
			}
		}
	}
	return out
}

func (c *Config) allInstalledPackMCPAgentTypesLocked() map[string]struct{} {
	out := make(map[string]struct{})
	manifests, _ := c.installedPackManifestsLocked()
	for _, m := range manifests {
		for _, agentType := range m.MCPAgents {
			key := strings.ToLower(strings.TrimSpace(agentType))
			if key != "" {
				out[key] = struct{}{}
			}
		}
	}
	return out
}

func (c *Config) mcpAgentEnabledByPacksLocked(agentType string) bool {
	key := mcpAgentConfigKey(agentType)
	if key == "" {
		return false
	}
	enabled := c.enabledPackMCPAgentTypesLocked()
	if _, ok := enabled[key]; ok {
		return true
	}
	// Rust MCP follows expert creation, not pack mcp_agents.
	return key == "rust"
}
