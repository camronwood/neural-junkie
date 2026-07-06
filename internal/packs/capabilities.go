package packs

import (
	"fmt"
	"strings"
)

// PlatformCapabilityTokens are NJ-owned capabilities any pack may declare without capability_defs.
var PlatformCapabilityTokens = []string{
	"customer-pack",
	"settings-overlay",
	"workspace-guide",
}

// OfficialDomainCapabilityTokens are platform capabilities owned by official catalog domain packs.
var OfficialDomainCapabilityTokens = []string{
	"ide-v2",
	"ide-v3-composer",
	"git-rest",
	"inline-completion",
	"cad-api",
	"cad-viewer",
	"cad-workbench",
	"lora-training",
	"lora-compose",
	"lora-adapters",
	"personal-learning",
	"aws-api",
	"aws-sso",
	"incident-api",
	"jira-integration",
	"incident-triage",
	"github-issues-integration",
	"linear-integration",
	"pagerduty-integration",
	"sentry-integration",
	"web-browser",
	"web-preview",
	"web-browser-workbench",
	"biology-api",
	"biology-workbench",
	"biology-sidecar",
	"music-generation",
	"music-workbench",
	"music-sidecar",
	"lora-training-sidecar",
}

// KnownExtensionKinds are valid capability_defs.kind values.
var KnownExtensionKinds = []string{
	"hub-sidecar",
	"mcp-sidecar",
	"file-viewer",
	"toolbar-chip",
	"mcp-tools",
	"settings-schema",
}

// KnownCapabilityTokens is the union of platform + official domain tokens (for catalog packs).
// Pack-local capabilities are defined in capability_defs and are not listed here.
var KnownCapabilityTokens = append(
	append([]string{}, PlatformCapabilityTokens...),
	OfficialDomainCapabilityTokens...,
)

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
	Kind        string                 `yaml:"kind" json:"kind"`
	Routes      []string               `yaml:"routes,omitempty" json:"routes,omitempty"`
	Sidecar     *CapabilitySidecarSpec `yaml:"sidecar,omitempty" json:"sidecar,omitempty"`
	UI          *CapabilityUI          `yaml:"ui,omitempty" json:"ui,omitempty"`
	MatchGlob   string                 `yaml:"match_glob,omitempty" json:"match_glob,omitempty"`
	Viewer      string                 `yaml:"viewer,omitempty" json:"viewer,omitempty"`
	Settings    []string               `yaml:"settings,omitempty" json:"settings,omitempty"`
	MCPTools     []string               `yaml:"mcp_tools,omitempty" json:"mcp_tools,omitempty"`
	MCPToolsPath string                 `yaml:"mcp_tools_path,omitempty" json:"mcp_tools_path,omitempty"`
	MCPAgents    []string               `yaml:"mcp_agents,omitempty" json:"mcp_agents,omitempty"`
}

// ResolvedCapability is a runtime capability entry from platform or pack-local defs.
type ResolvedCapability struct {
	ID          string        `json:"id"`
	QualifiedID string        `json:"qualified_id"`
	PackID      string        `json:"pack_id,omitempty"`
	Kind        string        `json:"kind,omitempty"`
	Platform    bool          `json:"platform,omitempty"`
	Routes      []string      `json:"routes,omitempty"`
	UI          *CapabilityUI `json:"ui,omitempty"`
	MatchGlob   string        `json:"match_glob,omitempty"`
	Viewer      string        `json:"viewer,omitempty"`
	Settings    []string      `json:"settings,omitempty"`
	MCPTools    []string      `json:"mcp_tools,omitempty"`
	MCPAgents   []string      `json:"mcp_agents,omitempty"`
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

// IsPlatformCapability reports whether cap is an NJ platform or official domain token.
func IsPlatformCapability(cap string) bool {
	cap = strings.TrimSpace(cap)
	for _, k := range KnownCapabilityTokens {
		if k == cap {
			return true
		}
	}
	return false
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
		if IsPlatformCapability(capID) {
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
		if IsPlatformCapability(shortID) {
			out = append(out, ResolvedCapability{
				ID:          shortID,
				QualifiedID: shortID,
				PackID:      m.ID,
				Kind:        "platform",
				Platform:    true,
			})
			continue
		}
		def, ok := m.CapabilityDefs[shortID]
		if !ok {
			continue
		}
		rc := ResolvedCapability{
			ID:          shortID,
			QualifiedID: QualifiedCapabilityID(m.ID, shortID),
			PackID:      m.ID,
			Kind:        def.Kind,
			Routes:      append([]string(nil), def.Routes...),
			UI:          def.UI,
			MatchGlob:   def.MatchGlob,
			Viewer:      def.Viewer,
			Settings:    append([]string(nil), def.Settings...),
			MCPTools:    append([]string(nil), def.MCPTools...),
			MCPAgents:   append([]string(nil), def.MCPAgents...),
		}
		out = append(out, rc)
	}
	return out
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
