package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/delegation"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const activateCapabilityToolName = "activate_capability"
const requestCapabilityHelpToolName = "request_capability_help"
const activeCapabilitiesMetadataKey = "active_capabilities"

// isConversationalOnlyTurn reports presence / vibe-check messages that must stay
// chat-only: no ask_user, capability handoffs, workspace MCP, or file tools.
func isConversationalOnlyTurn(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	if isSocialOrStatusPing(content) || intent.LooksLikePresenceCheck(content) {
		return true
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); ok {
		for _, o := range decision.PolicyOverrides {
			if o == "presence_check" {
				return true
			}
		}
		if decision.Action == intent.ActionAnswer && decision.Interaction == intent.InteractionCasual {
			return true
		}
	}
	return false
}

// shouldOfferCapabilityTools reports whether activate_capability / request_capability_help
// should be exposed for this turn.
func shouldOfferCapabilityTools(msg *protocol.Message) bool {
	return !isConversationalOnlyTurn(msg)
}

func (a *Agent) capabilityState() config.AgentCapabilityState {
	return config.AppConfig().ResolveAgentCapabilities(a.Info.ID, string(a.Info.Type), a.Info.Name)
}

func (a *Agent) discoverableCapabilityIDs() []string {
	state := a.capabilityState()
	out := make([]string, 0, len(state.Discoverable))
	for _, cap := range state.Discoverable {
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (a *Agent) activeCapabilityIDs(msg *protocol.Message) []string {
	state := a.capabilityState()
	effective := make(map[string]packs.ResolvedCapability, len(state.Effective))
	for _, id := range state.Effective {
		if cap, ok := packs.ResolveCapabilityQuery(state.Discoverable, id); ok {
			effective[id] = *cap
		}
	}
	active := make(map[string]bool)
	if msg != nil && msg.Metadata != nil {
		switch values := msg.Metadata[activeCapabilitiesMetadataKey].(type) {
		case []string:
			for _, id := range values {
				active[id] = true
			}
		case []interface{}:
			for _, raw := range values {
				if id, ok := raw.(string); ok {
					active[id] = true
				}
			}
		}
	}

	content := ""
	if msg != nil {
		content = strings.ToLower(msg.Content)
	}
	if msg == nil {
		providerCaps := config.AppConfig().CapabilitiesForAgentType(string(a.Info.Type))
		for _, cap := range providerCaps {
			id := cap.QualifiedID
			if id == "" {
				id = cap.ID
			}
			if _, ok := effective[id]; ok {
				active[id] = true
			}
		}
	}
	// Preserve the straightforward single-specialist case while avoiding a large
	// schema payload for agents that can reach many capability bundles.
	if len(effective) == 1 {
		for id := range effective {
			if a.ProvidesCapability(id) {
				active[id] = true
			}
		}
	} else if content != "" {
		for id, cap := range effective {
			if a.ProvidesCapability(id) && capabilityMatchesContent(cap, content) {
				active[id] = true
			}
		}
	}

	out := make([]string, 0, len(active))
	for id := range active {
		if _, ok := effective[id]; ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func capabilityMatchesContent(cap packs.ResolvedCapability, content string) bool {
	needles := []string{cap.ID, cap.Label, cap.Description}
	needles = append(needles, cap.MCPTools...)
	for _, needle := range needles {
		needle = strings.ToLower(strings.TrimSpace(needle))
		if needle == "" {
			continue
		}
		needle = strings.NewReplacer("_", " ", "-", " ", "/", " ").Replace(needle)
		for _, token := range strings.Fields(needle) {
			if len(token) >= 4 && strings.Contains(content, token) {
				return true
			}
		}
	}
	return false
}

func (a *Agent) activationToolDefinition(msg *protocol.Message) (ai.ClaudeToolDefinition, bool) {
	state := a.capabilityState()
	active := stringSet(a.activeCapabilityIDs(msg))
	var catalog []string
	for _, cap := range state.Discoverable {
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		if active[id] {
			continue
		}
		status := "denied"
		if capabilityContainsString(state.Effective, id) {
			status = "helper only"
			if a.ProvidesCapability(id) {
				status = "available locally"
			}
		}
		label := cap.Label
		if label == "" {
			label = cap.ID
		}
		catalog = append(catalog, fmt.Sprintf("%s (%s, %s)", id, label, status))
	}
	if len(catalog) == 0 {
		return ai.ClaudeToolDefinition{}, false
	}
	sort.Strings(catalog)
	schema := json.RawMessage(`{
	  "type": "object",
	  "properties": {
	    "capability_id": {
	      "type": "string",
	      "description": "Qualified capability bundle id to activate"
	    }
	  },
	  "required": ["capability_id"]
	}`)
	return ai.ClaudeToolDefinition{
		Name:        activateCapabilityToolName,
		Description: "Activate one capability bundle for this turn before using its tools. Catalog: " + strings.Join(catalog, "; "),
		InputSchema: schema,
	}, true
}

func (a *Agent) capabilityHelpToolDefinition(msg *protocol.Message) (ai.ClaudeToolDefinition, bool) {
	if msg != nil && msg.Metadata != nil {
		if depth, ok := msg.Metadata["handoff_depth"].(int); ok && depth >= 1 {
			return ai.ClaudeToolDefinition{}, false
		}
	}
	dc := a.getDelegationClient()
	if dc == nil || !config.AppConfig().CapabilityHandoffsEnabled() || len(a.discoverableCapabilityIDs()) == 0 {
		return ai.ClaudeToolDefinition{}, false
	}
	var providers []string
	for _, row := range dc.CapabilityDirectory(a.Info.ID) {
		names := make([]string, 0, len(row.AgentNames))
		for _, name := range row.AgentNames {
			names = append(names, "@"+name)
		}
		providers = append(providers, row.CapabilityID+" -> "+strings.Join(names, ", "))
	}
	description := "Open a temporary linked channel and ask one capable agent to perform a bounded task when local activation cannot work because of policy, credentials, environment, or specialist context."
	if len(providers) > 0 {
		description += " Available helpers: " + strings.Join(providers, "; ")
	}
	return ai.ClaudeToolDefinition{
		Name:        requestCapabilityHelpToolName,
		Description: description,
		InputSchema: json.RawMessage(`{
		  "type": "object",
		  "properties": {
		    "capability_id": {"type":"string","description":"Qualified capability id required for the task"},
		    "task": {"type":"string","description":"One bounded task for the helping agent"}
		  },
		  "required": ["capability_id", "task"]
		}`),
	}, true
}

func (a *Agent) executeActivateCapabilityTool(msg *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		CapabilityID string `json:"capability_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid activate_capability input: %w", err)
	}
	state := a.capabilityState()
	cap, ok := packs.ResolveCapabilityQuery(state.Discoverable, strings.TrimSpace(args.CapabilityID))
	if !ok {
		return "", fmt.Errorf("capability %q is not enabled or installed", args.CapabilityID)
	}
	id := cap.QualifiedID
	if id == "" {
		id = cap.ID
	}
	if !capabilityContainsString(state.Effective, id) {
		return "", fmt.Errorf("capability %q is denied by policy or requires an explicit sensitive-capability grant", id)
	}
	if !a.ProvidesCapability(id) {
		return "", fmt.Errorf("capability %q is available in the hub but not executable by %s; use request_capability_help", id, a.Info.Name)
	}
	if msg == nil {
		return "", fmt.Errorf("capability activation requires an active turn")
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	active := a.activeCapabilityIDs(msg)
	if !capabilityContainsString(active, id) {
		active = append(active, id)
		sort.Strings(active)
	}
	msg.Metadata[activeCapabilitiesMetadataKey] = active
	return fmt.Sprintf("Capability %s activated for this turn. Its tools are now available.", id), nil
}

func (a *Agent) executeRequestCapabilityHelpTool(ctx context.Context, msg *protocol.Message, input json.RawMessage) (string, error) {
	var args struct {
		CapabilityID string `json:"capability_id"`
		Task         string `json:"task"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid request_capability_help input: %w", err)
	}
	if msg == nil {
		return "", fmt.Errorf("capability help requires an active message")
	}
	dc := a.getDelegationClient()
	if dc == nil || !config.AppConfig().CapabilityHandoffsEnabled() {
		return "", fmt.Errorf("capability handoff is disabled")
	}
	result, err := dc.RequestCapabilityHelp(ctx, delegation.CapabilityHelpRequest{
		FromID:          a.Info.ID,
		FromName:        a.Info.Name,
		CreatedBy:       msg.From.Name,
		CapabilityID:    strings.TrimSpace(args.CapabilityID),
		Task:            strings.TrimSpace(args.Task),
		SourceChannel:   msg.Channel,
		SourceMessageID: msg.ID,
		Depth:           0,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Capability handoff %s completed in %s. The result was returned to %s; do not duplicate it. Summary: %s", result.HandoffID, result.Channel, result.SourceChannel, result.Summary), nil
}

func (a *Agent) filterToolsForActiveCapabilities(msg *protocol.Message, tools []ai.ClaudeToolDefinition) []ai.ClaudeToolDefinition {
	cfg := config.AppConfig()
	active := stringSet(a.activeCapabilityIDs(msg))
	providerCapabilities := cfg.CapabilitiesForAgentType(string(a.Info.Type))
	providerActive := len(providerCapabilities) == 0
	for _, cap := range providerCapabilities {
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		if active[id] {
			providerActive = true
			break
		}
	}
	out := make([]ai.ClaudeToolDefinition, 0, len(tools))
	for _, tool := range tools {
		cap, owned := cfg.CapabilityForTool(tool.Name)
		if !owned {
			// Agent-scoped legacy tools inherit the provider pack's activation.
			if providerActive {
				out = append(out, tool)
			}
			continue
		}
		id := cap.QualifiedID
		if id == "" {
			id = cap.ID
		}
		if active[id] {
			out = append(out, tool)
		}
	}
	return out
}

// ProvidesCapability reports whether this runtime can actually execute a permitted bundle.
func (a *Agent) ProvidesCapability(capabilityID string) bool {
	state := a.capabilityState()
	cap, ok := packs.ResolveCapabilityQuery(state.Discoverable, capabilityID)
	if !ok {
		return false
	}
	id := cap.QualifiedID
	if id == "" {
		id = cap.ID
	}
	if !capabilityContainsString(state.Effective, id) {
		return false
	}
	for _, provider := range cap.MCPAgents {
		if strings.EqualFold(provider, string(a.Info.Type)) ||
			strings.EqualFold(provider, a.Info.ID) ||
			strings.EqualFold(provider, a.Info.Name) {
			return true
		}
	}
	if len(cap.MCPTools) == 0 || a.MCPServer == nil {
		return false
	}
	owned := stringSet(cap.MCPTools)
	for _, tool := range claudeToolsFromMCPServer(mcpServerFromInterface(a.MCPServer), effectiveMCPToolAllowlist(a, nil)) {
		if owned[tool.Name] {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func capabilityContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
