package config

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// Pack IDs for domain packs.
const (
	PackLifeSciences        = "life-sciences"
	PackSoftwareDevelopment = "software-development"
	PackSpecialistTuning    = "specialist-tuning"
)

// DevOllamaCodeModel is the recommended local model for software-development specialists.
const DevOllamaCodeModel = "qwen2.5-coder:14b"

// devSpecialistTypes are in-process engineering agent types owned by the software-development pack.
var devSpecialistTypes = []string{"backend", "frontend", "devops", "security", "architecture", "code-review", "database"}

// legacyDevSpecialistTypes are optional expert slugs gated by the software-development pack.
var legacyDevSpecialistTypes = []string{"rust"}

// PacksConfig stores installed packs, enable toggles, and layout ownership.
type PacksConfig struct {
	Installed   []string        `json:"installed,omitempty"`
	Enabled     map[string]bool `json:"enabled"`
	LayoutOwner string          `json:"layout_owner,omitempty"`
}

// DomainPack describes an installed pack merged from its manifest.
type DomainPack struct {
	ID             string
	Title          string
	Description    string
	LayoutProfile  string
	Capabilities   []string
	ExpertSlug     string
	ExpertLabel    string
	Agents         []AgentConfig
	ModelsToEnsure []string
	OllamaModel    string
	ExpertPresets  []packs.ExpertPreset
	LoRAAdapters   []packs.LoRAAdapterSpec
}

func manifestToDomainPack(m *packs.Manifest) DomainPack {
	if m == nil {
		return DomainPack{}
	}
	dp := DomainPack{
		ID:             m.ID,
		Title:          m.Title,
		Description:    m.Description,
		LayoutProfile:  m.DefaultLayoutProfile(),
		Capabilities:   append([]string(nil), m.Capabilities...),
		ExpertSlug:     m.ExpertSlug,
		ExpertLabel:    m.ExpertLabel,
		ModelsToEnsure: append([]string(nil), m.ModelsToEnsure...),
		OllamaModel:    m.OllamaModel,
		ExpertPresets:  append([]packs.ExpertPreset(nil), m.ExpertPresets...),
		LoRAAdapters:   append([]packs.LoRAAdapterSpec(nil), m.LoRAAdapters...),
	}
	for _, a := range m.Agents {
		ac := AgentConfig{
			Type:       a.Type,
			Name:       a.Name,
			Enabled:    true,
			ProviderID: "ollama-local",
		}
		if strings.TrimSpace(a.OllamaModel) != "" {
			ac.Model = strings.TrimSpace(a.OllamaModel)
		}
		dp.Agents = append(dp.Agents, ac)
	}
	return dp
}

// InstalledPackManifestByID returns the manifest for an installed pack id.
func (c *Config) InstalledPackManifestByID(id string) (*packs.Manifest, error) {
	if c == nil {
		return nil, fmt.Errorf("nil config")
	}
	id = strings.TrimSpace(id)
	for _, m := range c.installedPackManifests() {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, fmt.Errorf("pack %q not installed", id)
}

// InstalledPackManifests returns manifests for packs listed in config.installed.
func (c *Config) InstalledPackManifests() ([]*packs.Manifest, error) {
	if c == nil {
		return nil, nil
	}
	c.mu.RLock()
	ids := append([]string(nil), c.Packs.Installed...)
	c.mu.RUnlock()
	return c.installedPackManifestsFromIDs(ids)
}

func (c *Config) installedPackManifests() []*packs.Manifest {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	ids := append([]string(nil), c.Packs.Installed...)
	c.mu.RUnlock()
	out, _ := c.installedPackManifestsFromIDs(ids)
	return out
}

func (c *Config) installedPackManifestsFromIDs(ids []string) ([]*packs.Manifest, error) {
	var out []*packs.Manifest
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		dir, err := packs.InstalledPackDir(id)
		if err != nil {
			continue
		}
		m, err := packs.LoadManifest(dir)
		if err != nil {
			m, err = packs.LoadBuiltinManifest(id)
			if err != nil {
				continue
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// PackCatalog returns domain packs for installed pack ids.
func (c *Config) PackCatalog() []DomainPack {
	manifests, err := c.InstalledPackManifests()
	if err != nil || len(manifests) == 0 {
		return nil
	}
	var out []DomainPack
	for _, m := range manifests {
		out = append(out, manifestToDomainPack(m))
	}
	return out
}

// PackCatalogBuiltin returns metadata for all official packs (store listing).
func PackCatalogBuiltin() ([]DomainPack, error) {
	all, err := packs.AllBuiltinManifests()
	if err != nil {
		return nil, err
	}
	var out []DomainPack
	for _, m := range all {
		out = append(out, manifestToDomainPack(m))
	}
	return out, nil
}

// PackByID returns an installed pack or nil.
func (c *Config) PackByID(id string) *DomainPack {
	id = strings.TrimSpace(id)
	for _, p := range c.PackCatalog() {
		if p.ID == id {
			cp := p
			return &cp
		}
	}
	return nil
}

// DefaultPacksConfig returns default pack state (nothing installed, all off).
func DefaultPacksConfig() PacksConfig {
	return PacksConfig{
		Installed:   nil,
		Enabled:     make(map[string]bool),
		LayoutOwner: "",
	}
}

// IsPackInstalled reports whether packID is in packs.installed.
func (c *Config) IsPackInstalled(packID string) bool {
	if c == nil {
		return false
	}
	packID = strings.TrimSpace(packID)
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, id := range c.Packs.Installed {
		if id == packID {
			return true
		}
	}
	return false
}

// IsPackEnabled reports whether a pack is on (must be installed).
func (c *Config) IsPackEnabled(packID string) bool {
	if c == nil || !c.IsPackInstalled(packID) {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Packs.Enabled == nil {
		return false
	}
	return c.Packs.Enabled[packID]
}

// LayoutOwnerPackID returns the pack that owns UI layout profile.
func (c *Config) LayoutOwnerPackID() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return strings.TrimSpace(c.Packs.LayoutOwner)
}

// LayoutProfile returns team|ide from layout_owner pack manifest, or team if unset.
func (c *Config) LayoutProfile() string {
	owner := c.LayoutOwnerPackID()
	if owner == "" {
		return "team"
	}
	if p := c.PackByID(owner); p != nil {
		return p.LayoutProfile
	}
	return "team"
}

// EnabledCapabilities returns the union of capabilities from all enabled installed packs.
func (c *Config) EnabledCapabilities() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, pack := range c.PackCatalog() {
		if !c.IsPackEnabled(pack.ID) {
			continue
		}
		for _, cap := range pack.Capabilities {
			if _, ok := seen[cap]; ok {
				continue
			}
			seen[cap] = struct{}{}
			out = append(out, cap)
		}
	}
	return out
}

// AnyPackCapability reports whether any enabled pack declares cap.
func (c *Config) AnyPackCapability(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, ccap := range c.EnabledCapabilities() {
		if ccap == cap {
			return true
		}
	}
	return false
}

func (c *Config) countEnabledPacksLocked() int {
	n := 0
	if c.Packs.Enabled == nil {
		return 0
	}
	for _, id := range c.Packs.Installed {
		if c.Packs.Enabled[id] {
			n++
		}
	}
	return n
}

// InstallPack copies a builtin pack to ~/.neural-junkie/packs and adds to installed (does not enable).
func (c *Config) InstallPack(packID string) error {
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return fmt.Errorf("pack id required")
	}
	if err := packs.InstallOfficialPack(packID); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	if !c.packInstalledLocked(packID) {
		c.Packs.Installed = append(c.Packs.Installed, packID)
	}
	c.Packs.Enabled[packID] = false
	return nil
}

// UninstallPack disables, removes from disk and installed list.
func (c *Config) UninstallPack(packID string) error {
	packID = strings.TrimSpace(packID)
	if !c.IsPackInstalled(packID) {
		return fmt.Errorf("pack %q is not installed", packID)
	}
	if c.IsPackEnabled(packID) {
		if err := c.SetPackEnabled(packID, false); err != nil {
			return err
		}
	}
	c.mu.Lock()
	var kept []string
	for _, id := range c.Packs.Installed {
		if id != packID {
			kept = append(kept, id)
		}
	}
	c.Packs.Installed = kept
	delete(c.Packs.Enabled, packID)
	if c.Packs.LayoutOwner == packID {
		c.Packs.LayoutOwner = c.firstEnabledPackLocked()
	}
	c.mu.Unlock()
	if err := packs.UninstallPack(packID); err != nil {
		return err
	}
	c.SyncAgentsFromPacks()
	return nil
}

// SetPackEnabled updates pack toggle and syncs agents/models.
func (c *Config) SetPackEnabled(packID string, enabled bool) error {
	if !c.IsPackInstalled(packID) {
		return fmt.Errorf("pack %q is not installed — install it from Domain packs first", packID)
	}
	c.mu.Lock()
	err := c.setPackEnabledLocked(packID, enabled)
	c.mu.Unlock()
	if err != nil {
		return err
	}
	c.SyncAgentsFromPacks()
	return nil
}

func (c *Config) setPackEnabledLocked(packID string, enabled bool) error {
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	wasEnabled := c.packEnabledLocked(packID)
	c.Packs.Enabled[packID] = enabled
	if enabled && !wasEnabled {
		if c.countEnabledPacksLocked() == 1 {
			c.Packs.LayoutOwner = packID
		}
	}
	if !enabled {
		if c.Packs.LayoutOwner == packID {
			c.Packs.LayoutOwner = c.firstEnabledPackLocked()
		}
		if c.countEnabledPacksLocked() == 0 {
			c.Packs.LayoutOwner = ""
		}
	}
	return nil
}

func (c *Config) firstEnabledPackLocked() string {
	for _, id := range c.Packs.Installed {
		if c.Packs.Enabled[id] {
			return id
		}
	}
	return ""
}

func (c *Config) packInstalledLocked(packID string) bool {
	for _, id := range c.Packs.Installed {
		if id == packID {
			return true
		}
	}
	return false
}

// MigrateInstalledPacks upgrades legacy configs: mark enabled packs as installed, set layout_owner.
func (c *Config) MigrateInstalledPacks() {
	if c == nil {
		return
	}
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	// Legacy: packs.enabled keys without installed — treat as installed+enabled.
	for _, id := range []string{PackSoftwareDevelopment, PackLifeSciences} {
		if c.Packs.Enabled[id] {
			if !c.packInstalledLocked(id) {
				_ = packs.InstallOfficialPack(id)
				c.Packs.Installed = append(c.Packs.Installed, id)
			}
		}
	}
	if c.Packs.LayoutOwner == "" {
		if c.Packs.Enabled[PackSoftwareDevelopment] {
			c.Packs.LayoutOwner = PackSoftwareDevelopment
		} else if c.Packs.Enabled[PackLifeSciences] {
			c.Packs.LayoutOwner = PackLifeSciences
		}
	}
	c.MigrateSpecialistTuningForLoRAUsers()
}

// MigrateSpecialistTuningForLoRAUsers auto-enables the tuning pack when legacy configs used nj-* LoRA tags.
func (c *Config) MigrateSpecialistTuningForLoRAUsers() {
	if c == nil || c.IsPackEnabled(PackSpecialistTuning) {
		return
	}
	if !c.configUsesNJComposedTags() {
		return
	}
	if !c.IsPackInstalled(PackSpecialistTuning) {
		if err := c.InstallPack(PackSpecialistTuning); err != nil {
			return
		}
	}
	_ = c.SetPackEnabled(PackSpecialistTuning, true)
}

func (c *Config) configUsesNJComposedTags() bool {
	for _, a := range c.Agents {
		if isNJComposedModelTag(a.Model) {
			return true
		}
	}
	for _, m := range c.Ollama.ModelsToEnsure {
		if isNJComposedModelTag(m) {
			return true
		}
	}
	return false
}

func isNJComposedModelTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "nj-") && strings.Contains(tag, ":")
}

// SyncAgentsFromPacks merges enabled pack agents into cfg.Agents and updates models_to_ensure.
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
	for _, pack := range c.packCatalogLocked() {
		for _, a := range pack.Agents {
			packTypes[a.Type] = struct{}{}
		}
	}
	for _, t := range legacyDevSpecialistTypes {
		packTypes[t] = struct{}{}
	}

	for i := range c.Agents {
		if _, owned := packTypes[c.Agents[i].Type]; !owned {
			continue
		}
		if !c.packEnabledLocked(packForAgentTypeLocked(c, c.Agents[i].Type)) {
			c.Agents[i].Enabled = false
		}
	}

	for _, pack := range c.packCatalogLocked() {
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
				if strings.TrimSpace(want.Model) != "" {
					c.Agents[idx].Model = strings.TrimSpace(want.Model)
				}
			}
		}
		if pack.OllamaModel != "" && c.shouldApplyPackOllamaModelLocked(pack.ID) {
			c.applyPackOllamaModelLocked(pack.OllamaModel)
		}
	}

	c.mergeModelsToEnsureFromPacksLocked()
	c.syncMCPFromPacksLocked()
}

func (c *Config) packCatalogLocked() []DomainPack {
	manifests, _ := c.installedPackManifestsLocked()
	var out []DomainPack
	for _, m := range manifests {
		out = append(out, manifestToDomainPack(m))
	}
	return out
}

func (c *Config) installedPackManifestsLocked() ([]*packs.Manifest, error) {
	var out []*packs.Manifest
	for _, id := range c.Packs.Installed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		dir, err := packs.InstalledPackDir(id)
		if err != nil {
			continue
		}
		m, err := packs.LoadManifest(dir)
		if err != nil {
			m, err = packs.LoadBuiltinManifest(id)
			if err != nil {
				continue
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// shouldApplyPackOllamaModelLocked applies Ollama model from layout_owner pack when enabled.
func (c *Config) shouldApplyPackOllamaModelLocked(packID string) bool {
	owner := strings.TrimSpace(c.Packs.LayoutOwner)
	if owner == "" {
		return false
	}
	return owner == packID && c.packEnabledLocked(packID)
}

func (c *Config) packEnabledLocked(packID string) bool {
	if c.Packs.Enabled == nil {
		return false
	}
	if !c.packInstalledLocked(packID) {
		return false
	}
	return c.Packs.Enabled[packID]
}

func packForAgentTypeLocked(c *Config, agentType string) string {
	for _, pack := range c.packCatalogLocked() {
		for _, a := range pack.Agents {
			if a.Type == agentType {
				return pack.ID
			}
		}
	}
	for _, t := range legacyDevSpecialistTypes {
		if t == agentType {
			return PackSoftwareDevelopment
		}
	}
	return ""
}

func packForAgentType(agentType string) string {
	// Used where config is unavailable; check builtins only.
	for _, id := range packs.BuiltinIDs {
		m, err := packs.LoadBuiltinManifest(id)
		if err != nil {
			continue
		}
		for _, a := range m.Agents {
			if a.Type == agentType {
				return id
			}
		}
	}
	for _, t := range legacyDevSpecialistTypes {
		if t == agentType {
			return PackSoftwareDevelopment
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
	for _, pack := range c.packCatalogLocked() {
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
		for _, la := range pack.LoRAAdapters {
			if tag := strings.TrimSpace(la.BaseOllamaTag); tag != "" {
				if _, ok := seen[tag]; !ok {
					seen[tag] = struct{}{}
					merged = append(merged, tag)
				}
			}
			tag := strings.TrimSpace(la.OllamaTag)
			if tag == "" && la.AgentType != "" {
				tag = hfhubSpecialistTag(la.AgentType)
			}
			if tag != "" {
				if _, ok := seen[tag]; !ok {
					seen[tag] = struct{}{}
					merged = append(merged, tag)
				}
			}
		}
	}
	c.Ollama.ModelsToEnsure = merged
}

func hfhubSpecialistTag(agentType string) string {
	t := strings.ToLower(strings.TrimSpace(agentType))
	if t == "biology" {
		return "nj-biology:8b"
	}
	return "nj-" + t + ":14b"
}

// ExpertPreset is one row for /create-expert and New DM persona dropdowns.
type ExpertPreset struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	FromPack string `json:"from_pack,omitempty"`
}

var coreExpertPresets = []ExpertPreset{
	{Slug: "assistant", Label: "Assistant"},
}

var legacyDevPackExpertSlugs = map[string]struct{}{
	"rust": {},
}

func isDevPackExpertSlug(slug string) bool {
	if p, err := packs.LoadBuiltinManifest(PackSoftwareDevelopment); err == nil {
		for _, ep := range p.ExpertPresets {
			if ep.Slug == slug {
				return true
			}
		}
	}
	_, ok := legacyDevPackExpertSlugs[slug]
	return ok
}

// AvailableExpertPresets returns core presets plus slugs from enabled packs.
func (c *Config) AvailableExpertPresets() []ExpertPreset {
	out := append([]ExpertPreset(nil), coreExpertPresets...)
	if c == nil {
		return out
	}
	for _, pack := range c.PackCatalog() {
		if !c.IsPackEnabled(pack.ID) {
			continue
		}
		for _, ep := range pack.ExpertPresets {
			out = append(out, ExpertPreset{
				Slug:     ep.Slug,
				Label:    ep.Label,
				FromPack: pack.ID,
			})
		}
		if pack.ExpertSlug != "" {
			out = append(out, ExpertPreset{
				Slug:     pack.ExpertSlug,
				Label:    pack.ExpertLabel,
				FromPack: pack.ID,
			})
		}
	}
	return out
}

func (c *Config) PresetExpertDeniedMessage(slug string) string {
	if c == nil || c.PresetExpertAllowed(slug) {
		return ""
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	switch slug {
	case "biology":
		return "Biology experts require the **Life sciences** pack. Install and enable it in Settings → Domain packs."
	default:
		if isDevPackExpertSlug(slug) {
			return "Software development specialists require the **Software development** pack. Install and enable it in Settings → Domain packs."
		}
	}
	return ""
}

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
	if isDevPackExpertSlug(slug) {
		return c.IsPackEnabled(PackSoftwareDevelopment)
	}
	return true
}

func IsDevSpecialistType(agentType string) bool {
	t := strings.ToLower(strings.TrimSpace(agentType))
	for _, d := range devSpecialistTypes {
		if d == t {
			return true
		}
	}
	for _, d := range legacyDevSpecialistTypes {
		if d == t {
			return true
		}
	}
	return false
}

// PackStatus is returned by GET /api/packs.
type PackStatus struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Installed      bool     `json:"installed"`
	Enabled        bool     `json:"enabled"`
	LayoutProfile  string   `json:"layout_profile,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	ExpertSlug     string   `json:"expert_slug,omitempty"`
	ExpertLabel    string   `json:"expert_label,omitempty"`
	Version        string   `json:"version,omitempty"`
}

// PackCatalogStatus is a store row from GET /api/packs/catalog.
type PackCatalogStatus struct {
	ID               string   `json:"id"`
	Version          string   `json:"version"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	IconKey          string   `json:"icon_key,omitempty"`
	Publisher        string   `json:"publisher,omitempty"`
	Builtin          bool     `json:"builtin,omitempty"`
	Installed        bool     `json:"installed"`
	Enabled          bool     `json:"enabled"`
	LoRAAdapterCount int      `json:"lora_adapter_count,omitempty"`
	LoRABaseTags     []string `json:"lora_base_tags,omitempty"`
}

// PacksAPIResponse is GET /api/packs payload.
type PacksAPIResponse struct {
	Packs         []PackStatus `json:"packs"`
	LayoutOwner   string       `json:"layout_owner,omitempty"`
	LayoutProfile string       `json:"layout_profile,omitempty"`
	Capabilities  []string     `json:"capabilities,omitempty"`
}

// ListPackStatus returns installed packs with status.
func (c *Config) ListPackStatus() PacksAPIResponse {
	resp := PacksAPIResponse{
		LayoutOwner:   c.LayoutOwnerPackID(),
		LayoutProfile: c.LayoutProfile(),
		Capabilities:  c.EnabledCapabilities(),
	}
	for _, pack := range c.PackCatalog() {
		resp.Packs = append(resp.Packs, c.packStatusFromDomain(pack))
	}
	return resp
}

func (c *Config) packStatusFromDomain(pack DomainPack) PackStatus {
	m, _ := packs.LoadBuiltinManifest(pack.ID)
	ver := ""
	if m != nil {
		ver = m.Version
	}
	return PackStatus{
		ID:            pack.ID,
		Title:         pack.Title,
		Description:   pack.Description,
		Installed:     true,
		Enabled:       c.IsPackEnabled(pack.ID),
		LayoutProfile: pack.LayoutProfile,
		Capabilities:  pack.Capabilities,
		ExpertSlug:    pack.ExpertSlug,
		ExpertLabel:   pack.ExpertLabel,
		Version:       ver,
	}
}

// ListPackCatalogStatus returns store catalog with install/enable flags.
func (c *Config) ListPackCatalogStatus() ([]PackCatalogStatus, error) {
	cat, err := packs.FetchCatalog()
	if err != nil {
		// Listing still works when remote catalog is unreachable (embedded merge).
		cat, _ = packs.LoadBuiltinCatalog()
		if cat == nil {
			return nil, err
		}
	}
	var out []PackCatalogStatus
	for _, e := range cat.Packs {
		row := PackCatalogStatus{
			ID:          e.ID,
			Version:     e.Version,
			Title:       e.Title,
			Description: e.Description,
			IconKey:     e.IconKey,
			Publisher:   e.Publisher,
			Builtin:     e.Builtin,
			Installed:   c.IsPackInstalled(e.ID),
			Enabled:     c.IsPackEnabled(e.ID),
		}
		if m, err := c.packManifestForCatalog(e.ID); err == nil && m != nil {
			row.LoRAAdapterCount = len(m.LoRAAdapters)
			if row.LoRAAdapterCount > 0 {
				seen := make(map[string]struct{})
				for _, la := range m.LoRAAdapters {
					tag := strings.TrimSpace(la.BaseOllamaTag)
					if tag == "" {
						tag = DevOllamaCodeModel
					}
					if _, ok := seen[tag]; !ok {
						seen[tag] = struct{}{}
						row.LoRABaseTags = append(row.LoRABaseTags, tag)
					}
				}
			}
		}
		out = append(out, row)
	}
	return out, nil
}

// packManifestForCatalog returns the installed pack manifest when present, else the builtin embed.
func (c *Config) packManifestForCatalog(packID string) (*packs.Manifest, error) {
	if c != nil {
		if m, err := c.InstalledPackManifestByID(packID); err == nil && m != nil {
			return m, nil
		}
	}
	return packs.LoadBuiltinManifest(packID)
}

func ConfigurableSpecialistTypes() map[string]bool {
	types := make(map[string]bool)
	all, _ := packs.AllBuiltinManifests()
	for _, m := range all {
		for _, a := range m.Agents {
			types[a.Type] = true
		}
	}
	for _, t := range legacyDevSpecialistTypes {
		types[t] = true
	}
	return types
}

func (c *Config) migrateSoftwareDevelopmentPackIfNeeded() {
	if c == nil {
		return
	}
	if c.Packs.Enabled == nil {
		c.Packs.Enabled = make(map[string]bool)
	}
	if _, explicit := c.Packs.Enabled[PackSoftwareDevelopment]; explicit {
		return
	}
	for _, a := range c.Agents {
		if a.Enabled && IsDevSpecialistType(a.Type) {
			c.Packs.Enabled[PackSoftwareDevelopment] = true
			if !c.packInstalledLocked(PackSoftwareDevelopment) {
				_ = packs.InstallOfficialPack(PackSoftwareDevelopment)
				c.Packs.Installed = append(c.Packs.Installed, PackSoftwareDevelopment)
			}
			return
		}
	}
}

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

func (c *Config) SpecialistShouldBeRunning(agentType string) bool {
	if c == nil {
		return false
	}
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if !ConfigurableSpecialistTypes()[agentType] {
		return false
	}
	c.mu.RLock()
	packID := packForAgentTypeLocked(c, agentType)
	enabled := packID == "" || c.packEnabledLocked(packID)
	c.mu.RUnlock()
	if !enabled {
		return false
	}
	return c.AgentTypeEnabled(agentType)
}
