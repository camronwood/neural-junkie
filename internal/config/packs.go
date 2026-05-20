package config

import (
	"fmt"
	"strings"
)

// Pack IDs for domain packs (toggle in Settings).
const (
	PackLifeSciences         = "life-sciences"
	PackSoftwareDevelopment  = "software-development"
)

// DevOllamaCodeModel is the recommended local model for software-development specialists.
const DevOllamaCodeModel = "qwen2.5-coder:14b"

// devSpecialistTypes are in-process engineering agent types owned by the software-development pack.
var devSpecialistTypes = []string{"backend", "frontend", "devops", "database", "security", "rust"}

// PacksConfig stores which optional domain packs are enabled.
type PacksConfig struct {
	Enabled map[string]bool `json:"enabled"`
}

// DomainPack describes an optional add-on: agents, models, and expert UI slugs.
type DomainPack struct {
	ID             string
	Title          string
	Description    string
	ExpertSlug     string
	ExpertLabel    string
	Agents         []AgentConfig
	ModelsToEnsure []string
	// OllamaModel, when non-empty, is applied to the first ollama-local provider when the pack is enabled.
	OllamaModel string
}

// PackCatalog returns all installable domain packs.
func PackCatalog() []DomainPack {
	return []DomainPack{
		{
			ID:          PackLifeSciences,
			Title:       "Life sciences",
			Description: "OpenBioLLM chat (koesn), BiologyExpert, sequence analysis and ESMFold structure prediction (research use only).",
			ExpertSlug:  "biology",
			ExpertLabel: "Biology / Life sciences",
			OllamaModel: BioOllamaChatModel,
			ModelsToEnsure: []string{
				BioOllamaChatModel,
				BioOllamaToolModel,
				BioOllamaTag,
			},
			Agents: []AgentConfig{
				{Type: "biology", Name: "BiologyExpert", Enabled: true, ProviderID: "ollama-local"},
			},
		},
		{
			ID:          PackSoftwareDevelopment,
			Title:       "Software development",
			Description: "Go, React, Rust, DevOps, database, and security specialists with MCP analysis tools and Qwen Coder models.",
			OllamaModel: DevOllamaCodeModel,
			ModelsToEnsure: []string{
				DevOllamaCodeModel,
				UtilityOllamaModel,
			},
			Agents: []AgentConfig{
				{Type: "backend", Name: "GoExpert", Enabled: true, ProviderID: "ollama-local"},
				{Type: "frontend", Name: "ReactExpert", Enabled: true, ProviderID: "ollama-local"},
				{Type: "devops", Name: "DevOpsPro", Enabled: true, ProviderID: "ollama-local"},
				{Type: "database", Name: "SQLMaster", Enabled: true, ProviderID: "ollama-local"},
				{Type: "security", Name: "SecurityExpert", Enabled: true, ProviderID: "ollama-local"},
				{Type: "rust", Name: "RustExpert", Enabled: true, ProviderID: "ollama-local"},
			},
		},
	}
}

// PackByID returns a catalog pack or nil.
func PackByID(id string) *DomainPack {
	id = strings.TrimSpace(id)
	for _, p := range PackCatalog() {
		if p.ID == id {
			cp := p
			return &cp
		}
	}
	return nil
}

// DefaultPacksConfig returns default pack toggles (all off).
func DefaultPacksConfig() PacksConfig {
	enabled := make(map[string]bool)
	for _, p := range PackCatalog() {
		enabled[p.ID] = false
	}
	return PacksConfig{Enabled: enabled}
}

// IsPackEnabled reports whether a pack is on (missing key = false).
func (c *Config) IsPackEnabled(packID string) bool {
	if c == nil || c.Packs.Enabled == nil {
		return false
	}
	return c.Packs.Enabled[packID]
}

// SetPackEnabled updates pack toggle and syncs agents/models.
func (c *Config) SetPackEnabled(packID string, enabled bool) error {
	pack := PackByID(packID)
	if pack == nil {
		return fmt.Errorf("unknown pack %q", packID)
	}
	c.mu.Lock()
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	c.Packs.Enabled[packID] = enabled
	c.mu.Unlock()
	c.SyncAgentsFromPacks()
	return nil
}

// SyncAgentsFromPacks merges enabled pack agents into cfg.Agents and updates models_to_ensure.
// Pack-owned agent types are enabled/disabled from pack toggles.
func (c *Config) SyncAgentsFromPacks() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}

	packTypes := make(map[string]struct{})
	for _, pack := range PackCatalog() {
		for _, a := range pack.Agents {
			packTypes[a.Type] = struct{}{}
		}
	}

	// Disable pack-owned agents when their pack is off.
	for i := range c.Agents {
		if _, owned := packTypes[c.Agents[i].Type]; !owned {
			continue
		}
		if !c.packEnabledLocked(packForAgentType(c.Agents[i].Type)) {
			c.Agents[i].Enabled = false
		}
	}

	// Upsert agents for enabled packs.
	for _, pack := range PackCatalog() {
		if !c.packEnabledLocked(pack.ID) {
			continue
		}
		for _, want := range pack.Agents {
			idx := agentIndexByType(c.Agents, want.Type)
			if idx < 0 {
				acfg := want
				acfg.Enabled = true
				c.Agents = append(c.Agents, acfg)
			} else {
				c.Agents[idx].Enabled = true
				if c.Agents[idx].Name == "" {
					c.Agents[idx].Name = want.Name
				}
			}
		}
		if pack.OllamaModel != "" && c.shouldApplyPackOllamaModelLocked(pack.ID) {
			c.applyPackOllamaModelLocked(pack.OllamaModel)
		}
	}

	c.mergeModelsToEnsureFromPacksLocked()
}

// shouldApplyPackOllamaModelLocked applies Ollama model only when exactly one pack with a model is enabled.
func (c *Config) shouldApplyPackOllamaModelLocked(packID string) bool {
	var enabledWithModel []string
	for _, p := range PackCatalog() {
		if p.OllamaModel != "" && c.packEnabledLocked(p.ID) {
			enabledWithModel = append(enabledWithModel, p.ID)
		}
	}
	if len(enabledWithModel) != 1 {
		return false
	}
	return enabledWithModel[0] == packID
}

func (c *Config) packEnabledLocked(packID string) bool {
	if c.Packs.Enabled == nil {
		return false
	}
	return c.Packs.Enabled[packID]
}

func packForAgentType(agentType string) string {
	for _, pack := range PackCatalog() {
		for _, a := range pack.Agents {
			if a.Type == agentType {
				return pack.ID
			}
		}
	}
	return ""
}

func agentIndexByType(agents []AgentConfig, agentType string) int {
	for i := range agents {
		if agents[i].Type == agentType {
			return i
		}
	}
	return -1
}

func (c *Config) applyPackOllamaModelLocked(model string) {
	for i := range c.AI.Providers {
		if c.AI.Providers[i].Type == "ollama" && (c.AI.Providers[i].ID == "ollama-local" || c.AI.Providers[i].Endpoint != "") {
			c.AI.Providers[i].Model = model
			return
		}
	}
}

func (c *Config) mergeModelsToEnsureFromPacksLocked() {
	seen := make(map[string]struct{})
	var merged []string
	for _, m := range c.Ollama.ModelsToEnsure {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; !ok {
			seen[m] = struct{}{}
			merged = append(merged, m)
		}
	}
	for _, pack := range PackCatalog() {
		if !c.packEnabledLocked(pack.ID) {
			continue
		}
		for _, m := range pack.ModelsToEnsure {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			if _, ok := seen[m]; !ok {
				seen[m] = struct{}{}
				merged = append(merged, m)
			}
		}
	}
	c.Ollama.ModelsToEnsure = merged
}

// ExpertPreset is one row for /create-expert and New DM persona dropdowns.
type ExpertPreset struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
	// FromPack is set when the preset comes from an enabled domain pack.
	FromPack string `json:"from_pack,omitempty"`
}

// Core expert slugs (always available).
var coreExpertPresets = []ExpertPreset{
	{Slug: "assistant", Label: "Assistant"},
}

// devPackExpertPresets are /create-expert and New DM presets when software-development is on.
var devPackExpertPresets = []ExpertPreset{
	{Slug: "rust", Label: "Rust"},
	{Slug: "backend", Label: "Backend"},
	{Slug: "frontend", Label: "Frontend"},
	{Slug: "devops", Label: "DevOps"},
	{Slug: "database", Label: "Database"},
	{Slug: "security", Label: "Security"},
}

// AvailableExpertPresets returns core presets plus slugs from enabled packs.
func (c *Config) AvailableExpertPresets() []ExpertPreset {
	out := append([]ExpertPreset(nil), coreExpertPresets...)
	if c == nil {
		return out
	}
	if c.IsPackEnabled(PackSoftwareDevelopment) {
		for _, p := range devPackExpertPresets {
			cp := p
			cp.FromPack = PackSoftwareDevelopment
			out = append(out, cp)
		}
	}
	for _, pack := range PackCatalog() {
		if !c.IsPackEnabled(pack.ID) {
			continue
		}
		if pack.ExpertSlug == "" {
			continue
		}
		out = append(out, ExpertPreset{
			Slug:     pack.ExpertSlug,
			Label:    pack.ExpertLabel,
			FromPack: pack.ID,
		})
	}
	return out
}

// PresetExpertDeniedMessage returns a user-facing error when a preset slug requires a disabled pack.
func (c *Config) PresetExpertDeniedMessage(slug string) string {
	if c == nil || c.PresetExpertAllowed(slug) {
		return ""
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	switch slug {
	case "biology":
		return "Biology experts require the **Life sciences** pack. Enable it in Settings → AI & providers → Domain packs."
	default:
		for _, p := range devPackExpertPresets {
			if p.Slug == slug {
				return "Software development specialists require the **Software development** pack. Enable it in Settings → AI & providers → Domain packs."
			}
		}
	}
	return ""
}

// PresetExpertAllowed reports whether a preset /create-expert slug may be spawned.
func (c *Config) PresetExpertAllowed(slug string) bool {
	if c == nil {
		return false
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return false
	}
	if slug == "assistant" {
		return true
	}
	if slug == "biology" {
		return c.IsPackEnabled(PackLifeSciences)
	}
	for _, p := range devPackExpertPresets {
		if p.Slug == slug {
			return c.IsPackEnabled(PackSoftwareDevelopment)
		}
	}
	return true // custom slugs
}

// IsDevSpecialistType reports whether agentType is owned by the software-development pack.
func IsDevSpecialistType(agentType string) bool {
	t := strings.ToLower(strings.TrimSpace(agentType))
	for _, d := range devSpecialistTypes {
		if d == t {
			return true
		}
	}
	return false
}

// PackStatus is returned by GET /api/packs.
type PackStatus struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ExpertSlug  string `json:"expert_slug,omitempty"`
	ExpertLabel string `json:"expert_label,omitempty"`
}

// ListPackStatus returns catalog with current enabled flags.
func (c *Config) ListPackStatus() []PackStatus {
	var out []PackStatus
	for _, pack := range PackCatalog() {
		out = append(out, PackStatus{
			ID:          pack.ID,
			Title:       pack.Title,
			Description: pack.Description,
			Enabled:     c.IsPackEnabled(pack.ID),
			ExpertSlug:  pack.ExpertSlug,
			ExpertLabel: pack.ExpertLabel,
		})
	}
	return out
}

// ConfigurableSpecialistTypes are agent types started from config.agents (not moderator/assistant/cli).
func ConfigurableSpecialistTypes() map[string]bool {
	types := make(map[string]bool)
	for _, p := range PackCatalog() {
		for _, a := range p.Agents {
			types[a.Type] = true
		}
	}
	return types
}

// migrateSoftwareDevelopmentPackIfNeeded enables the dev pack for legacy configs that had
// in-process specialists enabled before packs existed.
func (c *Config) migrateSoftwareDevelopmentPackIfNeeded() {
	if c == nil {
		return
	}
	if c.Packs.Enabled == nil {
		c.Packs = DefaultPacksConfig()
	}
	if _, explicit := c.Packs.Enabled[PackSoftwareDevelopment]; explicit {
		return
	}
	for _, a := range c.Agents {
		if a.Enabled && IsDevSpecialistType(a.Type) {
			c.Packs.Enabled[PackSoftwareDevelopment] = true
			return
		}
	}
}

// AgentTypeEnabled returns whether an agent type is enabled in config (ignores pack toggles).
func (c *Config) AgentTypeEnabled(agentType string) bool {
	if c == nil {
		return false
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, a := range c.Agents {
		if strings.ToLower(a.Type) == agentType {
			return a.Enabled
		}
	}
	return false
}

// SpecialistShouldBeRunning reports whether an in-process pack specialist should stay registered.
// Pack toggles take precedence over per-agent enabled flags in config.
func (c *Config) SpecialistShouldBeRunning(agentType string) bool {
	if c == nil {
		return false
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if !ConfigurableSpecialistTypes()[agentType] {
		return false
	}
	packID := packForAgentType(agentType)
	if packID != "" && !c.IsPackEnabled(packID) {
		return false
	}
	return c.AgentTypeEnabled(agentType)
}
