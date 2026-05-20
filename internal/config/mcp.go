package config

import (
	"strings"
)

// MCPConfig controls in-process MCP tool servers for specialist agents.
type MCPConfig struct {
	// Enabled is the master switch (default true in DefaultConfig).
	Enabled bool `json:"enabled"`
	// Agents overrides per agent type (backend, devops, database, biology, …).
	// Missing keys use defaults: backend/devops/database/biology follow agent enabled; frontend/security off.
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

// DefaultMCPConfig returns MCP defaults (tool servers on for core dev specialists).
func DefaultMCPConfig() MCPConfig {
	return MCPConfig{
		Enabled: true,
		Agents: map[string]bool{
			"backend":  true,
			"devops":   true,
			"database": true,
			"biology":  true,
			"frontend": false,
			"security": false,
		},
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

// MCPEnabledForAgent reports whether the MCP server for agentType (BACKEND, biology, …) should run.
func (c *Config) MCPEnabledForAgent(agentType string) bool {
	if c == nil || !c.MCP.Enabled {
		return false
	}
	key := strings.ToLower(strings.TrimSpace(agentType))
	if key == "" {
		return false
	}
	if !c.SpecialistShouldBeRunning(key) {
		return false
	}
	if c.MCP.Agents != nil {
		if v, ok := c.MCP.Agents[key]; ok {
			return v
		}
	}
	switch key {
	case "backend", "devops", "database", "biology":
		return true
	default:
		return false
	}
}

// MCPPort returns the configured port for an agent MCP server or 0 for default.
func (c *Config) MCPPort(agentType string) int {
	if c == nil || c.MCP.Ports == nil {
		return 0
	}
	key := strings.ToLower(strings.TrimSpace(agentType))
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
