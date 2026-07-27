package packs

import (
	"fmt"
	"strings"
)

// PlatformCapabilityTokens are thin NJ-owned tokens any pack may declare without capability_defs.
// Source of truth: capability_tokens.json (keep TS in sync via scripts/gen-pack-capabilities.py).
var PlatformCapabilityTokens = platformCapabilityTokensFromJSON()

// OfficialDomainCapabilityTokens are official catalog domain feature tokens (may also have hub/mcp sidecar defs).
// These are NOT the same as thin platform tokens — see IsThinPlatformCapability vs IsKnownCapabilityToken.
var OfficialDomainCapabilityTokens = officialDomainCapabilityTokensFromJSON()

// KnownExtensionKinds are valid capability_defs.kind values.
var KnownExtensionKinds = []string{
	"hub-sidecar",
	"mcp-sidecar",
	"file-viewer",
	"toolbar-chip",
	"mcp-tools",
	"settings-schema",
	"artifact-renderer",
}

// KnownCapabilityTokens is the union of thin platform + official domain tokens (for catalog packs).
// Pack-local capabilities are defined in capability_defs and are not listed here.
var KnownCapabilityTokens = append(
	append([]string{}, PlatformCapabilityTokens...),
	OfficialDomainCapabilityTokens...,
)

const (
	CapabilityExposureSafe      = "safe"
	CapabilityExposureSensitive = "sensitive"
)

// KnownArtifactRenderers are trusted, host-owned Neural Canvas renderers.
// Packs may map data to these IDs but cannot ship executable UI code.
var KnownArtifactRenderers = []string{
	"nj.markdown",
	"nj.mermaid",
	"nj.code",
	"nj.table",
	"nj.chart",
	"nj.timeline",
	"nj.image",
	"nj.graph",
	"nj.map",
	"nj.knowledge-graph",
	"nj.runbook",
	"nj.cad",
	"nj.structure",
	"nj.music",
	"nj.arena",
	"nj.scan-summary",
	"nj.scan-analysis",
	"nj.comparator-analysis",
}

// CapabilityToolbarUI declares a toolbar chip for a pack-local capability.
// Label is shown when Icon is empty (max 3 characters recommended). Icon is a pack-relative
// asset path (e.g. assets/icons/chip.png) or http(s) URL served by the desktop via the hub.
type CapabilityToolbarUI struct {
	ID    string `yaml:"id,omitempty" json:"id,omitempty"`
	Label string `yaml:"label,omitempty" json:"label,omitempty"`
	Icon  string `yaml:"icon,omitempty" json:"icon,omitempty"`
}

// CapabilityUI declares desktop UI hooks for a pack-local capability.
type CapabilityUI struct {
	Toolbar *CapabilityToolbarUI `yaml:"toolbar,omitempty" json:"toolbar,omitempty"`
	Modal   string               `yaml:"modal,omitempty" json:"modal,omitempty"`
}

// CapabilitySidecarSpec declares how the hub starts a pack sidecar route module.
type CapabilitySidecarSpec struct {
	Module  string   `yaml:"module,omitempty" json:"module,omitempty"`
	Binary  string   `yaml:"binary,omitempty" json:"binary,omitempty"`
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`
}

// CapabilityDef declares what a pack-local capability does.
type CapabilityDef struct {
	Label        string                 `yaml:"label,omitempty" json:"label,omitempty"`
	Description  string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Exposure     string                 `yaml:"exposure,omitempty" json:"exposure,omitempty"`
	Kind         string                 `yaml:"kind" json:"kind"`
	Routes       []string               `yaml:"routes,omitempty" json:"routes,omitempty"`
	Sidecar      *CapabilitySidecarSpec `yaml:"sidecar,omitempty" json:"sidecar,omitempty"`
	UI           *CapabilityUI          `yaml:"ui,omitempty" json:"ui,omitempty"`
	MatchGlob    string                 `yaml:"match_glob,omitempty" json:"match_glob,omitempty"`
	Viewer       string                 `yaml:"viewer,omitempty" json:"viewer,omitempty"`
	Settings     []string               `yaml:"settings,omitempty" json:"settings,omitempty"`
	MCPTools     []string               `yaml:"mcp_tools,omitempty" json:"mcp_tools,omitempty"`
	MCPToolsPath string                 `yaml:"mcp_tools_path,omitempty" json:"mcp_tools_path,omitempty"`
	MCPAgents    []string               `yaml:"mcp_agents,omitempty" json:"mcp_agents,omitempty"`
	Renderer     string                 `yaml:"renderer,omitempty" json:"renderer,omitempty"`
	MediaTypes   []string               `yaml:"media_types,omitempty" json:"media_types,omitempty"`
	RendererAPI  int                    `yaml:"renderer_api_version,omitempty" json:"renderer_api_version,omitempty"`
	SchemaMin    int                    `yaml:"schema_version_min,omitempty" json:"schema_version_min,omitempty"`
	SchemaMax    int                    `yaml:"schema_version_max,omitempty" json:"schema_version_max,omitempty"`
	Fallback     string                 `yaml:"fallback,omitempty" json:"fallback,omitempty"`
}

// ResolvedCapability is a runtime capability entry from platform or pack-local defs.
type ResolvedCapability struct {
	ID          string        `json:"id"`
	QualifiedID string        `json:"qualified_id"`
	PackID      string        `json:"pack_id,omitempty"`
	Label       string        `json:"label,omitempty"`
	Description string        `json:"description,omitempty"`
	Exposure    string        `json:"exposure,omitempty"`
	Kind        string        `json:"kind,omitempty"`
	Platform    bool          `json:"platform,omitempty"`
	Routes      []string      `json:"routes,omitempty"`
	UI          *CapabilityUI `json:"ui,omitempty"`
	MatchGlob   string        `json:"match_glob,omitempty"`
	Viewer      string        `json:"viewer,omitempty"`
	Settings    []string      `json:"settings,omitempty"`
	MCPTools    []string      `json:"mcp_tools,omitempty"`
	MCPAgents   []string      `json:"mcp_agents,omitempty"`
	Renderer    string        `json:"renderer,omitempty"`
	MediaTypes  []string      `json:"media_types,omitempty"`
	RendererAPI int           `json:"renderer_api_version,omitempty"`
	SchemaMin   int           `json:"schema_version_min,omitempty"`
	SchemaMax   int           `json:"schema_version_max,omitempty"`
	Fallback    string        `json:"fallback,omitempty"`
}

// QualifiedCapabilityID returns packID/capID.
func QualifiedCapabilityID(packID, capID string) string {
	packID = strings.TrimSpace(packID)
	capID = strings.TrimSpace(capID)
	if packID == "" || capID == "" {
		return capID
	}
	if strings.Contains(capID, "/") {
		return capID
	}
	return packID + "/" + capID
}

// ParseQualifiedCapabilityID splits packID/capID or returns ("", capID) for short ids.
func ParseQualifiedCapabilityID(raw string) (packID, capID string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if i := strings.Index(raw, "/"); i >= 0 {
		return raw[:i], raw[i+1:]
	}
	return "", raw
}

// IsThinPlatformCapability reports whether cap is a thin NJ platform token
// (customer-pack / settings-overlay / workspace-guide), not an official domain token.
func IsThinPlatformCapability(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, k := range PlatformCapabilityTokens {
		if k == cap {
			return true
		}
	}
	return false
}

// IsKnownCapabilityToken reports whether cap is a thin platform or official domain token.
func IsKnownCapabilityToken(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, k := range KnownCapabilityTokens {
		if k == cap {
			return true
		}
	}
	return false
}

// IsPlatformCapability reports whether cap is an NJ-known token (thin platform OR official domain).
// Prefer IsThinPlatformCapability / IsKnownCapabilityToken for clearer taxonomy.
func IsPlatformCapability(cap string) bool {
	return IsKnownCapabilityToken(cap)
}

// IsKnownExtensionKind reports whether kind is a supported capability_defs.kind.
func IsKnownExtensionKind(kind string) bool {
	kind = strings.TrimSpace(kind)
	for _, k := range KnownExtensionKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// ValidateCapabilityDefs checks capability_defs and capabilities consistency for a manifest.
func (m *Manifest) ValidateCapabilityDefs(packDir string) (warnings, errors []string) {
	if m == nil {
		return nil, []string{"nil manifest"}
	}
	if m.CapabilityDefs == nil {
		m.CapabilityDefs = map[string]CapabilityDef{}
	}
	for capID, def := range m.CapabilityDefs {
		capID = strings.TrimSpace(capID)
		if capID == "" {
			errors = append(errors, "capability_defs contains empty key")
			continue
		}
		if IsThinPlatformCapability(capID) {
			if def.Kind == SidecarKindHub || def.Kind == SidecarKindMCP {
				warnings = append(warnings, validateCapabilityDefPaths(capID, def, packDir, &errors)...)
				continue
			}
			errors = append(errors, fmt.Sprintf("capability_defs[%q]: platform capability cannot be redefined in a pack", capID))
			continue
		}
		if strings.Contains(capID, "/") {
			errors = append(errors, fmt.Sprintf("capability_defs[%q]: use short id without pack prefix", capID))
			continue
		}
		if !IsKnownExtensionKind(def.Kind) {
			errors = append(errors, fmt.Sprintf("capability_defs[%q]: unknown kind %q", capID, def.Kind))
		}
		if def.Kind == "artifact-renderer" {
			if !isKnownArtifactRenderer(def.Renderer) {
				errors = append(errors, fmt.Sprintf("capability_defs[%q]: unknown trusted artifact renderer %q", capID, def.Renderer))
			}
			if len(def.MediaTypes) == 0 && strings.TrimSpace(def.MatchGlob) == "" {
				errors = append(errors, fmt.Sprintf("capability_defs[%q]: artifact renderer requires media_types or match_glob", capID))
			}
			if def.RendererAPI < 0 || def.SchemaMin < 0 || def.SchemaMax < 0 || (def.SchemaMax > 0 && def.SchemaMin > def.SchemaMax) {
				errors = append(errors, fmt.Sprintf("capability_defs[%q]: invalid renderer/schema version range", capID))
			}
		}
		exposure := strings.ToLower(strings.TrimSpace(def.Exposure))
		if exposure != "" && exposure != CapabilityExposureSafe && exposure != CapabilityExposureSensitive {
			errors = append(errors, fmt.Sprintf("capability_defs[%q]: exposure must be %q or %q", capID, CapabilityExposureSafe, CapabilityExposureSensitive))
		} else if exposure == "" && (len(def.MCPTools) > 0 || len(def.MCPAgents) > 0 || def.Kind == "mcp-tools") {
			warnings = append(warnings, fmt.Sprintf("capability_defs[%q]: executable capability has no exposure classification; using pack default", capID))
		}
		warnings = append(warnings, validateCapabilityDefPaths(capID, def, packDir, &errors)...)
	}
	for _, cap := range m.Capabilities {
		cap = strings.TrimSpace(cap)
		if cap == "" {
			continue
		}
		packID, shortID := ParseQualifiedCapabilityID(cap)
		if shortID == "" {
			shortID = cap
		}
		if packID != "" && packID != m.ID {
			errors = append(errors, fmt.Sprintf("capabilities[%q]: qualified id must use this pack's id %q", cap, m.ID))
			continue
		}
		if IsPlatformCapability(shortID) {
			continue
		}
		if _, ok := m.CapabilityDefs[shortID]; !ok {
			errors = append(errors, fmt.Sprintf("capabilities[%q]: pack-local capability requires capability_defs[%q] (see docs/PACK_CAPABILITY_DEFS.md)", shortID, shortID))
		}
	}
	for capID := range m.CapabilityDefs {
		if !m.HasCapability(capID) {
			warnings = append(warnings, fmt.Sprintf("capability_defs[%q] is defined but not listed in capabilities", capID))
		}
	}
	return warnings, errors
}

func validateCapabilityDefPaths(capID string, def CapabilityDef, packDir string, errors *[]string) []string {
	var warnings []string
	if packDir == "" {
		return warnings
	}
	if def.Sidecar != nil {
		if mod := strings.TrimSpace(def.Sidecar.Module); mod != "" {
			if _, err := ResolvePackRelativePath(packDir, mod); err != nil {
				*errors = append(*errors, fmt.Sprintf("capability_defs[%q].sidecar.module: %v", capID, err))
			}
		}
		if bin := strings.TrimSpace(def.Sidecar.Binary); bin != "" {
			if _, err := ResolvePackRelativePath(packDir, bin); err != nil {
				*errors = append(*errors, fmt.Sprintf("capability_defs[%q].sidecar.binary: %v", capID, err))
			}
		}
	}
	if path := strings.TrimSpace(def.MCPToolsPath); path != "" {
		if _, err := ResolvePackRelativePath(packDir, path); err != nil {
			*errors = append(*errors, fmt.Sprintf("capability_defs[%q].mcp_tools_path: %v", capID, err))
		}
	}
	return warnings
}

// BuildResolvedCapabilities returns resolved entries for a single enabled manifest.
func BuildResolvedCapabilities(m *Manifest) []ResolvedCapability {
	if m == nil {
		return nil
	}
	var out []ResolvedCapability
	for _, cap := range m.Capabilities {
		cap = strings.TrimSpace(cap)
		if cap == "" {
			continue
		}
		_, shortID := ParseQualifiedCapabilityID(cap)
		if shortID == "" {
			shortID = cap
		}
		if def, ok := m.CapabilityDefs[shortID]; ok && strings.TrimSpace(def.Kind) != "" {
			rc := resolvedCapabilityFromDef(m.ID, shortID, def)
			if IsPlatformCapability(shortID) {
				rc.Platform = true
				rc.QualifiedID = shortID
			}
			out = append(out, rc)
			continue
		}
		if IsPlatformCapability(shortID) {
			out = append(out, ResolvedCapability{
				ID:          shortID,
				QualifiedID: shortID,
				PackID:      m.ID,
				Label:       strings.ReplaceAll(shortID, "-", " "),
				Description: strings.TrimSpace(m.Description),
				Exposure:    defaultCapabilityExposure(m.ID),
				Kind:        "platform",
				Platform:    true,
				MCPAgents:   append([]string(nil), m.MCPAgents...),
			})
			continue
		}
	}
	return out
}

func resolvedCapabilityFromDef(packID, shortID string, def CapabilityDef) ResolvedCapability {
	exposure := strings.ToLower(strings.TrimSpace(def.Exposure))
	if exposure != CapabilityExposureSafe && exposure != CapabilityExposureSensitive {
		exposure = defaultCapabilityExposure(packID)
	}
	return ResolvedCapability{
		ID:          shortID,
		QualifiedID: QualifiedCapabilityID(packID, shortID),
		PackID:      packID,
		Label:       strings.TrimSpace(def.Label),
		Description: strings.TrimSpace(def.Description),
		Exposure:    exposure,
		Kind:        def.Kind,
		Routes:      append([]string(nil), def.Routes...),
		UI:          def.UI,
		MatchGlob:   def.MatchGlob,
		Viewer:      def.Viewer,
		Settings:    append([]string(nil), def.Settings...),
		MCPTools:    append([]string(nil), def.MCPTools...),
		MCPAgents:   append([]string(nil), def.MCPAgents...),
		Renderer:    strings.TrimSpace(def.Renderer),
		MediaTypes:  append([]string(nil), def.MediaTypes...),
		RendererAPI: def.RendererAPI,
		SchemaMin:   def.SchemaMin,
		SchemaMax:   def.SchemaMax,
		Fallback:    strings.TrimSpace(def.Fallback),
	}
}

func isKnownArtifactRenderer(id string) bool {
	id = strings.TrimSpace(id)
	for _, known := range KnownArtifactRenderers {
		if id == known {
			return true
		}
	}
	return false
}

func defaultCapabilityExposure(packID string) string {
	switch strings.ToLower(strings.TrimSpace(packID)) {
	case "cad", "life-sciences", "model-arena", "music-creation", "software-development", "web-browser":
		return CapabilityExposureSafe
	default:
		// AWS, incident-management, and unclassified third-party packs require a grant.
		return CapabilityExposureSensitive
	}
}

// MergeResolvedCapabilities merges capabilities from multiple manifests with short-id collision detection.
func MergeResolvedCapabilities(manifests []*Manifest) (resolved []ResolvedCapability, capTokens []string, shortCollisions []string) {
	seenToken := make(map[string]struct{})
	shortOwners := make(map[string]string)
	for _, m := range manifests {
		if m == nil {
			continue
		}
		for _, rc := range BuildResolvedCapabilities(m) {
			resolved = append(resolved, rc)
			if !rc.Platform {
				if owner, ok := shortOwners[rc.ID]; ok && owner != rc.PackID {
					shortCollisions = append(shortCollisions, rc.ID)
				} else {
					shortOwners[rc.ID] = rc.PackID
				}
			}
			for _, tok := range []string{rc.ID, rc.QualifiedID} {
				if _, ok := seenToken[tok]; ok {
					continue
				}
				seenToken[tok] = struct{}{}
				capTokens = append(capTokens, tok)
			}
		}
	}
	return resolved, capTokens, shortCollisions
}

// ResolveCapabilityQuery finds a capability by short or qualified id among resolved entries.
func ResolveCapabilityQuery(resolved []ResolvedCapability, query string) (*ResolvedCapability, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, false
	}
	packID, shortID := ParseQualifiedCapabilityID(query)
	if packID != "" {
		for i := range resolved {
			rc := &resolved[i]
			if rc.QualifiedID == query || (rc.PackID == packID && rc.ID == shortID) {
				return rc, true
			}
		}
		return nil, false
	}
	var matches []*ResolvedCapability
	for i := range resolved {
		rc := &resolved[i]
		if rc.ID == query {
			matches = append(matches, rc)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return nil, false
}
