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
}

// BiologyMCPConfig is persisted in config.json and edited in Settings.
type BiologyMCPConfig struct {
	ESMFoldModel     string `json:"esmfold_model,omitempty"`
	MaxAnalyzeLength int    `json:"max_analyze_length,omitempty"`
	MaxFoldLength    int    `json:"max_fold_length,omitempty"`
	ArtifactsDir     string `json:"artifacts_dir,omitempty"`
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
