package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var assignablePackToolCapabilityIDs = []string{
	"maps-tools",
	"web-browser",
	"music-generation",
}

// handlePackToolsRoute implements Composition Model pack-tool grants:
//
//	GET  /api/mcp/pack-tools              list grantable pack capabilities + current grants
//	POST /api/mcp/pack-tools/{id}/grant   grant/revoke for a custom expert by name
func handlePackToolsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mcp/pack-tools"), "/")
	var parts []string
	if path != "" {
		parts = strings.Split(path, "/")
	}

	switch {
	case len(parts) == 0 && r.Method == http.MethodGet:
		handleListPackToolGrants(w, r)
	case len(parts) == 2 && parts[1] == "grant" && r.Method == http.MethodPost:
		handleGrantPackTool(w, r, parts[0])
	default:
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
	}
}

type packToolGrantView struct {
	CapabilityID  string   `json:"capability_id"`
	Available     bool     `json:"available"`
	Label         string   `json:"label"`
	ToolNames     []string `json:"tool_names,omitempty"`
	GrantedAgents []string `json:"granted_agents"`
}

func handleListPackToolGrants(w http.ResponseWriter, r *http.Request) {
	cfg := appConfig
	out := make([]packToolGrantView, 0, len(assignablePackToolCapabilityIDs))
	for _, id := range assignablePackToolCapabilityIDs {
		view := packToolGrantView{
			CapabilityID:  id,
			Available:     packCapabilityGrantAvailable(cfg, id),
			Label:         packToolCapabilityLabel(id),
			GrantedAgents: []string{},
		}
		if g, ok := cfg.PackToolGrantFor(id); ok {
			view.ToolNames = append([]string(nil), g.ToolNames...)
			view.GrantedAgents = append([]string(nil), g.GrantedAgents...)
		}
		out = append(out, view)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func handleGrantPackTool(w http.ResponseWriter, r *http.Request, capabilityID string) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	capabilityID = strings.TrimSpace(capabilityID)
	if !isAssignablePackToolCapability(capabilityID) {
		http.Error(w, "unknown pack capability", http.StatusBadRequest)
		return
	}
	if !packCapabilityGrantAvailable(appConfig, capabilityID) {
		http.Error(w, "pack capability not available (enable the pack first)", http.StatusBadRequest)
		return
	}
	var req struct {
		AgentName string `json:"agent_name"`
		Grant     bool   `json:"grant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	agentName := strings.TrimSpace(req.AgentName)
	if agentName == "" {
		http.Error(w, "agent_name is required", http.StatusBadRequest)
		return
	}

	grant, _ := appConfig.PackToolGrantFor(capabilityID)
	grant.CapabilityID = capabilityID
	filtered := make([]string, 0, len(grant.GrantedAgents)+1)
	for _, g := range grant.GrantedAgents {
		if !strings.EqualFold(strings.TrimSpace(g), agentName) {
			filtered = append(filtered, g)
		}
	}
	if req.Grant {
		filtered = append(filtered, agentName)
	}
	grant.GrantedAgents = filtered
	appConfig.UpsertPackToolGrant(grant)

	if err := appConfig.Save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config.SetAppConfig(appConfig)
	reattachPackToolsForExpert(agentName)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(grant)
}

func reattachPackToolsForExpert(agentName string) {
	if chatHub == nil {
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		return
	}
	if ag := ch.FindRuntimeAgentByDisplayName(agentName, protocol.AgentTypeExpert); ag != nil {
		agent.ReattachGrantedTools(ag)
	}
}

func isAssignablePackToolCapability(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, known := range assignablePackToolCapabilityIDs {
		if id == known {
			return true
		}
	}
	return false
}

func packCapabilityGrantAvailable(cfg *config.Config, id string) bool {
	if cfg == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "maps-tools":
		return cfg.IsPackEnabled(config.PackMaps) || cfg.HasPackCapability("maps-tools")
	case "web-browser":
		return cfg.IsPackEnabled(config.PackWebBrowser) || cfg.HasPackCapability("web-browser")
	case "music-generation":
		return cfg.IsPackEnabled(config.PackMusicCreation) || cfg.HasPackCapability("music-generation")
	default:
		return cfg.HasPackCapability(id)
	}
}

func packToolCapabilityLabel(id string) string {
	switch id {
	case "maps-tools":
		return "Maps tools"
	case "web-browser":
		return "Web browser automation"
	case "music-generation":
		return "Music generation"
	default:
		return id
	}
}
