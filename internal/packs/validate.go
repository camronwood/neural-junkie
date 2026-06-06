package packs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// KnownCapabilityTokens are capability tokens recognized by the desktop app.
var KnownCapabilityTokens = []string{
	"ide-v2",
	"ide-v3-composer",
	"git-rest",
	"inline-completion",
	"scan-summary-api",
	"scan-summary-viewer",
	"scan-analysis-viewer",
	"secondary-analysis-api",
	"secondary-analysis-viewer",
	"secondary-analysis-python",
	"secondary-analysis-customer",
	"cad-api",
	"cad-viewer",
	"cad-workbench",
	"lora-training",
	"lora-compose",
	"lora-adapters",
	"personal-learning",
	"customer-pack",
	"phoenix-import",
}

// KnownOverlayKeys are settings_overlay keys applied by the hub for customer packs.
var KnownOverlayKeys = []string{
	"secondary_analysis_tools_path",
	"python_executable",
	"cumulative_qc_dir",
	"default_panel_profile",
	"artifacts_dir",
	"phoenix_environment",
	"environment",
	"phoenix_credentials_path",
	"credentials_path",
	"phoenix_auth_config_path",
	"auth_config_path",
	"mcp.biology.secondary_analysis_tools_path",
	"mcp.biology.cumulative_qc_dir",
	"mcp.biology.python_executable",
}

// PackRequirementsContext supplies installed/enabled state for requires_packs checks.
type PackRequirementsContext struct {
	IsInstalled func(packID string) bool
	IsEnabled   func(packID string) bool
	// EnabledCapabilities returns currently enabled capability tokens (optional).
	EnabledCapabilities func() []string
}

// ValidationReport is returned by pack validate APIs.
type ValidationReport struct {
	Valid             bool                   `json:"valid"`
	Errors            []string               `json:"errors,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
	Manifest          *ManifestSummary       `json:"manifest,omitempty"`
	Assets            AssetAudit             `json:"assets"`
	ResolvedOverlay   map[string]string      `json:"resolved_overlay,omitempty"`
	RequiresPacks     []RequiresPackStatus   `json:"requires_packs,omitempty"`
	Preview           *ValidationPreview     `json:"preview,omitempty"`
}

// ManifestSummary is a JSON-friendly manifest snapshot.
type ManifestSummary struct {
	ID              string            `json:"id"`
	Version         string            `json:"version,omitempty"`
	Title           string            `json:"title"`
	Description     string            `json:"description,omitempty"`
	Publisher       string            `json:"publisher,omitempty"`
	PackKind        string            `json:"pack_kind,omitempty"`
	LayoutProfile   string            `json:"layout_profile,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	RequiresPacks   []string          `json:"requires_packs,omitempty"`
	SettingsOverlay map[string]string `json:"settings_overlay,omitempty"`
	Agents          []AgentSpec       `json:"agents,omitempty"`
	MCPAgents       []string          `json:"mcp_agents,omitempty"`
}

// AssetAudit reports bundled asset presence.
type AssetAudit struct {
	WorkspaceGuideFound   bool     `json:"workspace_guide_found"`
	WorkspaceGuidePath    string   `json:"workspace_guide_path,omitempty"`
	WorkspaceGuidePreview string   `json:"workspace_guide_preview,omitempty"`
	RunbooksCount         int      `json:"runbooks_count"`
	RunbookPaths          []string `json:"runbook_paths,omitempty"`
}

// RequiresPackStatus is one requires_packs entry with hub state.
type RequiresPackStatus struct {
	ID        string `json:"id"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
}

// ValidationPreview shows runtime effects if the pack were enabled.
type ValidationPreview struct {
	Agents                []AgentSpec `json:"agents,omitempty"`
	EffectiveCapabilities []string    `json:"effective_capabilities,omitempty"`
}

// ValidateZipBytes dry-runs a pack zip without installing.
func ValidateZipBytes(data []byte, ctx *PackRequirementsContext) (*ValidationReport, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty pack zip")
	}
	if len(data) > maxPackZipBytes {
		return nil, fmt.Errorf("pack zip exceeds %d bytes", maxPackZipBytes)
	}
	tmpZip, err := os.CreateTemp("", "nj-pack-validate-*.zip")
	if err != nil {
		return nil, err
	}
	zipPath := tmpZip.Name()
	defer os.Remove(zipPath)
	if _, err := tmpZip.Write(data); err != nil {
		tmpZip.Close()
		return nil, err
	}
	if err := tmpZip.Close(); err != nil {
		return nil, err
	}
	tmpDir, err := os.MkdirTemp("", "nj-pack-validate-extract-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	if err := extractZipSafe(zipPath, tmpDir); err != nil {
		return nil, err
	}
	manifestDir, err := findManifestDir(tmpDir)
	if err != nil {
		return nil, err
	}
	return validateManifestDir(manifestDir, ctx)
}

// ValidatePackDir validates an on-disk pack folder.
func ValidatePackDir(dir string, ctx *PackRequirementsContext) (*ValidationReport, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("pack directory required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("pack directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
	}
	manifestDir, err := findManifestDir(abs)
	if err != nil {
		return nil, err
	}
	return validateManifestDir(manifestDir, ctx)
}

// ValidateYAML parses pack.yaml text; asset checks run only when assetRoot is set.
func ValidateYAML(yamlText string, assetRoot string, ctx *PackRequirementsContext) (*ValidationReport, error) {
	yamlText = strings.TrimSpace(yamlText)
	if yamlText == "" {
		return nil, fmt.Errorf("pack_yaml required")
	}
	var m Manifest
	if err := yaml.Unmarshal([]byte(yamlText), &m); err != nil {
		return &ValidationReport{
			Valid:  false,
			Errors: []string{fmt.Sprintf("parse pack.yaml: %v", err)},
		}, nil
	}
	root := strings.TrimSpace(assetRoot)
	if root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		root = abs
	}
	return buildValidationReport(&m, root, ctx)
}

func validateManifestDir(manifestDir string, ctx *PackRequirementsContext) (*ValidationReport, error) {
	m, err := LoadManifest(manifestDir)
	if err != nil {
		return &ValidationReport{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}
	return buildValidationReport(m, manifestDir, ctx)
}

func buildValidationReport(m *Manifest, packDir string, ctx *PackRequirementsContext) (*ValidationReport, error) {
	report := &ValidationReport{
		Manifest: manifestSummary(m),
		Assets:   AssetAudit{},
	}
	var errors []string
	var warnings []string

	if err := m.Validate(); err != nil {
		errors = append(errors, err.Error())
	}
	if isBuiltinPackID(m.ID) {
		errors = append(errors, fmt.Sprintf("pack id %q collides with a builtin official pack", m.ID))
	}
	if m.IsCustomerPack() {
		// customer packs are expected for dev studio
	} else if strings.TrimSpace(m.PackKind) == "" {
		warnings = append(warnings, "pack_kind is not customer; custom packs should set pack_kind: customer or declare customer-pack capability")
	}

	for _, cap := range m.Capabilities {
		if !isKnownCapability(cap) {
			warnings = append(warnings, fmt.Sprintf("unknown capability token %q (may still work if the hub/desktop adds support later)", cap))
		}
	}
	for key := range m.SettingsOverlay {
		if !isKnownOverlayKey(key) {
			warnings = append(warnings, fmt.Sprintf("settings_overlay key %q is not a recognized biology/phoenix overlay key", key))
		}
	}

	if packDir != "" {
		warnings = append(warnings, auditAssets(m, packDir, report)...)
		if len(m.SettingsOverlay) > 0 {
			resolved, err := ResolveSettingsOverlay(m, packDir)
			if err != nil {
				errors = append(errors, err.Error())
			} else {
				report.ResolvedOverlay = resolved
			}
		}
	} else if strings.TrimSpace(m.Assets.WorkspaceGuide) != "" {
		warnings = append(warnings, "workspace_guide declared but pack_dir not set; asset existence not checked")
	}
	if strings.TrimSpace(m.Assets.RunbooksGlob) != "" && packDir == "" {
		warnings = append(warnings, "runbooks_glob declared but pack_dir not set; glob not checked")
	}

	if ctx != nil {
		for _, req := range m.RequiresPacks {
			req = strings.TrimSpace(req)
			if req == "" {
				continue
			}
			st := RequiresPackStatus{ID: req}
			if ctx.IsInstalled != nil {
				st.Installed = ctx.IsInstalled(req)
			}
			if ctx.IsEnabled != nil {
				st.Enabled = ctx.IsEnabled(req)
			}
			report.RequiresPacks = append(report.RequiresPacks, st)
			if !st.Installed {
				warnings = append(warnings, fmt.Sprintf("requires_packs: %q is not installed", req))
			} else if !st.Enabled {
				warnings = append(warnings, fmt.Sprintf("requires_packs: %q is installed but not enabled", req))
			}
		}
		report.Preview = previewIfEnabled(m, ctx)
	}

	report.Errors = errors
	report.Warnings = warnings
	report.Valid = len(errors) == 0
	return report, nil
}

func auditAssets(m *Manifest, packDir string, report *ValidationReport) []string {
	var warnings []string
	if m == nil || report == nil {
		return warnings
	}
	if guide := strings.TrimSpace(m.Assets.WorkspaceGuide); guide != "" {
		path, err := ResolvePackRelativePath(packDir, guide)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("workspace_guide: %v", err))
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("workspace_guide not readable: %v", err))
			} else {
				report.Assets.WorkspaceGuideFound = true
				report.Assets.WorkspaceGuidePath = guide
				preview := string(data)
				if len(preview) > 4000 {
					preview = preview[:4000] + "\n…"
				}
				report.Assets.WorkspaceGuidePreview = preview
			}
		}
	} else {
		warnings = append(warnings, "no workspace_guide declared in assets")
	}
	if glob := strings.TrimSpace(m.Assets.RunbooksGlob); glob != "" {
		paths, err := matchRunbooksGlob(packDir, glob)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("runbooks_glob: %v", err))
		} else {
			report.Assets.RunbooksCount = len(paths)
			report.Assets.RunbookPaths = paths
			if len(paths) == 0 {
				warnings = append(warnings, fmt.Sprintf("runbooks_glob %q matched no files", glob))
			}
		}
	}
	return warnings
}

func matchRunbooksGlob(packDir, glob string) ([]string, error) {
	glob = strings.TrimSpace(strings.ReplaceAll(glob, "\\", "/"))
	if glob == "" {
		return nil, nil
	}
	if filepath.IsAbs(glob) {
		return nil, fmt.Errorf("absolute glob not allowed")
	}
	full := filepath.Join(packDir, filepath.FromSlash(glob))
	matches, err := filepath.Glob(full)
	if err != nil {
		return nil, err
	}
	var rel []string
	packAbs, _ := filepath.Abs(packDir)
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		abs, _ := filepath.Abs(m)
		if r, err := filepath.Rel(packAbs, abs); err == nil {
			rel = append(rel, strings.ReplaceAll(r, "\\", "/"))
		}
	}
	return rel, nil
}

func manifestSummary(m *Manifest) *ManifestSummary {
	if m == nil {
		return nil
	}
	return &ManifestSummary{
		ID:              m.ID,
		Version:         m.Version,
		Title:           m.Title,
		Description:     m.Description,
		Publisher:       m.Publisher,
		PackKind:        m.PackKind,
		LayoutProfile:   m.DefaultLayoutProfile(),
		Capabilities:    append([]string(nil), m.Capabilities...),
		RequiresPacks:   append([]string(nil), m.RequiresPacks...),
		SettingsOverlay: copyStringMap(m.SettingsOverlay),
		Agents:          append([]AgentSpec(nil), m.Agents...),
		MCPAgents:       append([]string(nil), m.MCPAgents...),
	}
}

func previewIfEnabled(m *Manifest, ctx *PackRequirementsContext) *ValidationPreview {
	if m == nil {
		return nil
	}
	preview := &ValidationPreview{
		Agents: append([]AgentSpec(nil), m.Agents...),
	}
	seen := make(map[string]struct{})
	if ctx.EnabledCapabilities != nil {
		for _, cap := range ctx.EnabledCapabilities() {
			seen[cap] = struct{}{}
			preview.EffectiveCapabilities = append(preview.EffectiveCapabilities, cap)
		}
	}
	for _, cap := range m.Capabilities {
		if _, ok := seen[cap]; ok {
			continue
		}
		seen[cap] = struct{}{}
		preview.EffectiveCapabilities = append(preview.EffectiveCapabilities, cap)
	}
	return preview
}

func isKnownCapability(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, k := range KnownCapabilityTokens {
		if k == cap {
			return true
		}
	}
	return false
}

func isKnownOverlayKey(key string) bool {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "mcp.biology.")
	for _, k := range KnownOverlayKeys {
		if k == key || k == "mcp.biology."+key {
			return true
		}
	}
	return false
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
