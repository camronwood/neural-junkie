package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func (c *Config) migrateCapabilityPolicy(raw []byte) {
	if c == nil {
		return
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, existed := doc["capability_policy"]; !existed {
		// Preserve pre-feature tool access for upgraded installations. Fresh
		// configurations use the safer broad-safe default.
		c.CapabilityPolicy.AllowSensitiveByDefault = true
	}
	if c.CapabilityPolicy.HandoffsEnabled == nil {
		enabled := true
		c.CapabilityPolicy.HandoffsEnabled = &enabled
	}
	if c.CapabilityPolicy.AgentOverrides == nil {
		c.CapabilityPolicy.AgentOverrides = map[string]AgentCapabilityOverride{}
	}
}

// AgentCapabilityState is the explainable capability policy resolved for one agent.
type AgentCapabilityState struct {
	Discoverable []packs.ResolvedCapability `json:"discoverable"`
	Available    []string                   `json:"available"`
	Effective    []string                   `json:"effective"`
	Denied       []string                   `json:"denied"`
	Unavailable  []string                   `json:"unavailable"`
	Allow        []string                   `json:"allow,omitempty"`
	Deny         []string                   `json:"deny,omitempty"`
}

// IsAgentAssignableCapability limits policy to executable pack bundles.
func IsAgentAssignableCapability(cap packs.ResolvedCapability) bool {
	return len(cap.MCPTools) > 0 || len(cap.MCPAgents) > 0 || cap.Kind == "mcp-tools"
}

// ResolveAgentCapabilities applies broad-safe defaults and sparse per-agent overrides.
func (c *Config) ResolveAgentCapabilities(agentID, agentType, agentName string) AgentCapabilityState {
	if c == nil {
		return AgentCapabilityState{}
	}
	registry := c.ResolvedCapabilityRegistry().CapabilityRegistry

	c.mu.RLock()
	policy := c.CapabilityPolicy
	override := resolveAgentCapabilityOverrideLocked(c, agentID, agentType, agentName)
	c.mu.RUnlock()

	allow := capabilityIDSet(registry, override.Allow)
	deny := capabilityIDSet(registry, override.Deny)
	state := AgentCapabilityState{
		Allow: append([]string(nil), override.Allow...),
		Deny:  append([]string(nil), override.Deny...),
	}
	for _, cap := range registry {
		if !IsAgentAssignableCapability(cap) {
			continue
		}
		state.Discoverable = append(state.Discoverable, cap)
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		state.Available = append(state.Available, id)
		if deny[id] {
			state.Denied = append(state.Denied, id)
			continue
		}
		if cap.Exposure == packs.CapabilityExposureSafe || policy.AllowSensitiveByDefault || allow[id] {
			state.Effective = append(state.Effective, id)
		} else {
			state.Denied = append(state.Denied, id)
		}
	}
	sort.Strings(state.Available)
	sort.Strings(state.Effective)
	sort.Strings(state.Denied)
	return state
}

func resolveAgentCapabilityOverrideLocked(c *Config, agentID, agentType, agentName string) AgentCapabilityOverride {
	var out AgentCapabilityOverride
	for _, ac := range c.Agents {
		if (agentType != "" && strings.EqualFold(ac.Type, agentType)) ||
			(agentName != "" && strings.EqualFold(ac.Name, agentName)) {
			out.Allow = append(out.Allow, ac.CapabilityAllow...)
			out.Deny = append(out.Deny, ac.CapabilityDeny...)
		}
	}
	if c.CapabilityPolicy.AgentOverrides == nil {
		return normalizeCapabilityOverride(out)
	}
	keys := []string{
		strings.TrimSpace(agentID),
		strings.ToLower(strings.TrimSpace(agentType)) + ":" + strings.ToLower(strings.TrimSpace(agentName)),
		strings.ToLower(strings.TrimSpace(agentName)),
	}
	for _, key := range keys {
		if key == "" || key == ":" {
			continue
		}
		if extra, ok := c.CapabilityPolicy.AgentOverrides[key]; ok {
			out.Allow = append(out.Allow, extra.Allow...)
			out.Deny = append(out.Deny, extra.Deny...)
		}
	}
	return normalizeCapabilityOverride(out)
}

func capabilityIDSet(registry []packs.ResolvedCapability, values []string) map[string]bool {
	out := make(map[string]bool)
	for _, raw := range values {
		if cap, ok := packs.ResolveCapabilityQuery(registry, strings.TrimSpace(raw)); ok {
			id := cap.QualifiedID
			if id == "" {
				id = cap.ID
			}
			out[id] = true
		}
	}
	return out
}

func normalizeCapabilityOverride(in AgentCapabilityOverride) AgentCapabilityOverride {
	in.Allow = uniqueTrimmed(in.Allow)
	in.Deny = uniqueTrimmed(in.Deny)
	return in
}

func uniqueTrimmed(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// SetCapabilityDefaults updates the global sensitive-capability default.
func (c *Config) SetCapabilityDefaults(allowSensitive bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.CapabilityPolicy.AllowSensitiveByDefault = allowSensitive
	if c.CapabilityPolicy.AgentOverrides == nil {
		c.CapabilityPolicy.AgentOverrides = map[string]AgentCapabilityOverride{}
	}
	c.mu.Unlock()
}

func (c *Config) CapabilityHandoffsEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CapabilityPolicy.HandoffsEnabled == nil || *c.CapabilityPolicy.HandoffsEnabled
}

func (c *Config) SensitiveCapabilitiesAllowedByDefault() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CapabilityPolicy.AllowSensitiveByDefault
}

func (c *Config) SetCapabilityHandoffsEnabled(enabled bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.CapabilityPolicy.HandoffsEnabled = &enabled
	c.mu.Unlock()
}

// SetAgentCapabilityOverride validates and stores a sparse override.
func (c *Config) SetAgentCapabilityOverride(agentKey string, override AgentCapabilityOverride) error {
	if c == nil {
		return fmt.Errorf("config unavailable")
	}
	agentKey = strings.TrimSpace(agentKey)
	if agentKey == "" {
		return fmt.Errorf("agent key is required")
	}
	registry := c.ResolvedCapabilityRegistry().CapabilityRegistry
	override = normalizeCapabilityOverride(override)
	for _, id := range append(append([]string{}, override.Allow...), override.Deny...) {
		cap, ok := packs.ResolveCapabilityQuery(registry, id)
		if !ok || !IsAgentAssignableCapability(*cap) {
			return fmt.Errorf("unknown or non-executable capability %q", id)
		}
	}
	c.mu.Lock()
	if c.CapabilityPolicy.AgentOverrides == nil {
		c.CapabilityPolicy.AgentOverrides = map[string]AgentCapabilityOverride{}
	}
	if len(override.Allow) == 0 && len(override.Deny) == 0 {
		delete(c.CapabilityPolicy.AgentOverrides, agentKey)
	} else {
		c.CapabilityPolicy.AgentOverrides[agentKey] = override
	}
	c.mu.Unlock()
	return nil
}

// CapabilityForTool returns the executable bundle that owns an MCP tool.
func (c *Config) CapabilityForTool(toolName string) (*packs.ResolvedCapability, bool) {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" || c == nil {
		return nil, false
	}
	for _, cap := range c.ResolvedCapabilityRegistry().CapabilityRegistry {
		for _, candidate := range cap.MCPTools {
			if candidate == toolName {
				copy := cap
				return &copy, true
			}
		}
	}
	return nil, false
}

// CapabilitiesForAgentType returns executable bundles provided by an MCP agent type.
func (c *Config) CapabilitiesForAgentType(agentType string) []packs.ResolvedCapability {
	if c == nil {
		return nil
	}
	var out []packs.ResolvedCapability
	for _, cap := range c.ResolvedCapabilityRegistry().CapabilityRegistry {
		if !IsAgentAssignableCapability(cap) {
			continue
		}
		for _, provider := range cap.MCPAgents {
			if strings.EqualFold(strings.TrimSpace(provider), strings.TrimSpace(agentType)) {
				out = append(out, cap)
				break
			}
		}
	}
	return out
}
