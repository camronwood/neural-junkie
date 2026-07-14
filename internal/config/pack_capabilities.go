package config

import (
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// CapabilityRegistryResponse is the resolved capability registry for enabled packs.
type CapabilityRegistryResponse struct {
	Capabilities       []string                   `json:"capabilities"`
	CapabilityRegistry []packs.ResolvedCapability `json:"capability_registry"`
	ShortIDCollisions  []string                   `json:"short_id_collisions,omitempty"`
}

// ResolvedCapabilityRegistry returns merged capability registry from enabled installed packs.
func (c *Config) ResolvedCapabilityRegistry() CapabilityRegistryResponse {
	if c == nil {
		return CapabilityRegistryResponse{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.resolvedCapabilityRegistryLocked()
}

func (c *Config) resolvedCapabilityRegistryLocked() CapabilityRegistryResponse {
	var enabled []*packs.Manifest
	for _, id := range c.Packs.Installed {
		if !c.packEnabledLocked(id) {
			continue
		}
		m, err := c.installedManifestLocked(id)
		if err != nil || m == nil {
			continue
		}
		enabled = append(enabled, m)
	}
	resolved, tokens, collisions := packs.MergeResolvedCapabilities(enabled)
	return CapabilityRegistryResponse{
		Capabilities:       tokens,
		CapabilityRegistry: resolved,
		ShortIDCollisions:  collisions,
	}
}

// HasPackCapability reports whether query matches an enabled capability (short or qualified).
func (c *Config) HasPackCapability(query string) bool {
	if c == nil {
		return false
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	reg := c.ResolvedCapabilityRegistry()
	if _, ok := packs.ResolveCapabilityQuery(reg.CapabilityRegistry, query); ok {
		return true
	}
	for _, cap := range c.enabledCapabilitiesLegacy() {
		if cap == query {
			return true
		}
	}
	return false
}

// CapabilityForRoutePrefix returns the capability id that gates a hub route prefix,
// preferring resolved registry hub-sidecar entries over hard-coded fallbacks.
func (c *Config) CapabilityForRoutePrefix(routePrefix string) string {
	routePrefix = strings.TrimSpace(routePrefix)
	if routePrefix == "" || c == nil {
		return ""
	}
	reg := c.ResolvedCapabilityRegistry()
	for _, rc := range reg.CapabilityRegistry {
		if rc.Kind != "hub-sidecar" {
			continue
		}
		for _, r := range rc.Routes {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			if r == routePrefix || strings.HasPrefix(routePrefix, r+"/") {
				id := strings.TrimSpace(rc.ID)
				if id == "" {
					id = strings.TrimSpace(rc.QualifiedID)
				}
				if i := strings.LastIndex(id, "/"); i >= 0 {
					id = id[i+1:]
				}
				return id
			}
		}
	}
	return ""
}

// RouteOwnerPackID returns the pack id that owns a hub route prefix, or "".
func (c *Config) RouteOwnerPackID(routePrefix string) string {
	routePrefix = strings.TrimSpace(routePrefix)
	if routePrefix == "" {
		return ""
	}
	reg := c.ResolvedCapabilityRegistry()
	for _, rc := range reg.CapabilityRegistry {
		if rc.Kind != "hub-sidecar" {
			continue
		}
		for _, r := range rc.Routes {
			if r == routePrefix || strings.HasPrefix(routePrefix, r+"/") || routePrefix == strings.TrimSuffix(r, "/") {
				return rc.PackID
			}
		}
	}
	return ""
}

// validateCapabilityCollisionsAfterEnableLocked errors when enabled packs share a short capability id.
// Caller must hold c.mu (write lock).
func (c *Config) validateCapabilityCollisionsAfterEnableLocked() error {
	reg := c.resolvedCapabilityRegistryLocked()
	if len(reg.ShortIDCollisions) == 0 {
		return nil
	}
	return fmt.Errorf("capability short id collision among enabled packs: %v (use qualified ids like pack-id/cap-id)", reg.ShortIDCollisions)
}

// EnabledCapabilitiesFromRegistry returns capability tokens including qualified ids.
func (c *Config) EnabledCapabilitiesFromRegistry() []string {
	return c.ResolvedCapabilityRegistry().Capabilities
}

// enabledCapabilitiesLegacy returns raw capabilities from enabled pack manifests.
func (c *Config) enabledCapabilitiesLegacy() []string {
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

// MCPToolsForCapability reports whether toolName is enabled by a pack-local mcp-tools capability.
func (c *Config) MCPToolsForCapability(toolName string) bool {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return false
	}
	reg := c.ResolvedCapabilityRegistry()
	for _, rc := range reg.CapabilityRegistry {
		for _, t := range rc.MCPTools {
			if t == toolName {
				return true
			}
		}
	}
	return false
}

// PackHasMCPCapabilityKind reports whether any enabled pack declares kind mcp-tools or lists mcp_tools.
func (c *Config) PackHasMCPCapabilityKind(kind string) bool {
	reg := c.ResolvedCapabilityRegistry()
	for _, rc := range reg.CapabilityRegistry {
		if rc.Kind == kind {
			return true
		}
	}
	return false
}

// ScanMCPToolAllowed reports whether a biology scan/QC MCP tool is allowed by pack capabilities.
func (c *Config) ScanMCPToolAllowed(toolName string) bool {
	if c == nil {
		return false
	}
	toolName = strings.TrimSpace(toolName)
	switch toolName {
	case "summarize_scan_summary":
		return c.HasPackCapability("scan-summary-api")
	case "summarize_scan_analysis", "summarize_panel_qc":
		return c.HasPackCapability("scan-analysis-viewer") || c.HasPackCapability("scan-summary-api")
	case "run_12plex_qc", "summarize_comparator_output", "run_secondary_analysis":
		return c.HasPackCapability("secondary-analysis-api") || c.HasPackCapability("secondary-analysis-python")
	default:
		return true
	}
}
