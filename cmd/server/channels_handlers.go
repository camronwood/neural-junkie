package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	slackint "github.com/camronwood/neural-junkie/internal/integrations/slack"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleChannels(w http.ResponseWriter, r *http.Request) {
	channels := chatHub.ListChannels()
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_archived")), "true") {
		channels = chatHub.ListChannelsIncludingArchived()
	}
	if store, err := slackint.NewBindingStore(); err == nil {
		slackint.EnrichChannelsFromBindings(channels, store)
	}

	// Optional type filter
	typeFilter := r.URL.Query().Get("type")
	if typeFilter != "" {
		filtered := make([]*protocol.Channel, 0)
		for _, ch := range channels {
			if string(ch.Type) == typeFilter {
				filtered = append(filtered, ch)
			}
		}
		channels = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func handleArchiveChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := chatHub.ArchiveChannel(strings.TrimSpace(req.Name)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Project     string   `json:"project"`
		Type        string   `json:"type"`
		Members     []string `json:"members"`
		CreatedBy   string   `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	channelType := protocol.ChannelType(req.Type)
	if channelType == "" {
		channelType = protocol.ChannelTypePublic
	}

	// For DM channels, use the dedicated helper
	if channelType == protocol.ChannelTypeDM {
		if len(req.Members) == 0 {
			http.Error(w, "DM channels require at least one agent member", http.StatusBadRequest)
			return
		}
		ch, err := chatHub.CreateDMChannel(req.CreatedBy, req.Members[0], "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ch)
		return
	}

	channel := chatHub.CreateChannelWithType(req.Name, req.Description, req.Project, channelType, req.CreatedBy)

	// Auto-join requested agent members
	for _, agentID := range req.Members {
		if err := chatHub.AddAgentToChannel(agentID, req.Name); err != nil {
			log.Printf("Warning: failed to add agent %s to channel %s: %v", agentID, req.Name, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channel)
}

// handleOpenDM find-or-creates a DM with an agent. Primary UI path for sidebar agent clicks;
// rate-limit exempt so rapid switching cannot lock out /api/send.
func handleOpenDM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}

	var req struct {
		AgentID   string `json:"agent_id"`
		CreatedBy string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.AgentID) == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}
	ch, err := chatHub.CreateDMChannel(req.CreatedBy, req.AgentID, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ch)
}

// handleCreateDMAgent creates a new expert or CLI agent and a dedicated DM channel (agent is not joined to the caller's current channel).
func handleCreateDMAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CreatedBy       string   `json:"created_by"`
		Mode            string   `json:"mode"` // "expert" | "cli"
		DisplayName     string   `json:"display_name"`
		ExpertType      string   `json:"expert_type"`
		Persona         string   `json:"persona"` // optional extra instructions for custom experts
		ProviderID      string   `json:"provider_id"`
		Provider        string   `json:"provider"`
		Model           string   `json:"model"`
		CLIType         string   `json:"cli_type"`
		WorkDir         string   `json:"work_dir"`
		CapabilityAllow []string `json:"capability_allow"`
		CapabilityDeny  []string `json:"capability_deny"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.CreatedBy) == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		http.Error(w, "display_name is required", http.StatusBadRequest)
		return
	}

	rawHandler := chatHub.GetCommandHandler()
	ch, ok := rawHandler.(*hub.CommandHandler)
	if !ok || ch == nil {
		http.Error(w, "command handler unavailable", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	var dmCh *protocol.Channel
	var err error
	switch mode {
	case "expert":
		if strings.TrimSpace(req.ExpertType) == "" {
			http.Error(w, "expert_type is required for mode expert", http.StatusBadRequest)
			return
		}
		dmCh, err = ch.SpawnExpertAgentForDM(ctx, req.CreatedBy, req.ExpertType, req.DisplayName, req.ProviderID, req.Provider, req.Model, req.Persona, req.CapabilityAllow, req.CapabilityDeny)
	case "cli":
		if strings.TrimSpace(req.CLIType) == "" {
			http.Error(w, "cli_type is required for mode cli", http.StatusBadRequest)
			return
		}
		dmCh, err = ch.SpawnCLIAgentForDM(ctx, req.CreatedBy, req.CLIType, req.DisplayName, req.WorkDir)
	default:
		http.Error(w, `mode must be "expert" or "cli"`, http.StatusBadRequest)
		return
	}

	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "failed to start agent") {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dmCh)
}

func handleCLIAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statuses := cliMgr.ListStatus(cliProviderAPIKey)
	types := make([]string, 0, len(statuses))
	installed := make(map[string]bool, len(statuses))
	for _, st := range statuses {
		types = append(types, st.Type)
		installed[st.Type] = st.Installed
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"types":     types,
		"installed": installed,
		"agents":    statuses,
	})
}

func handleJoinChannel(w http.ResponseWriter, r *http.Request) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var req struct {
		AgentID  string `json:"agent_id"`
		Channel  string `json:"channel"`
		Greeting string `json:"greeting,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.JoinChannel(req.AgentID, req.Channel, req.Greeting); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	chatHub.EnsureAgentSubscribedToChannel(req.AgentID, req.Channel)

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleClearChannelHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := chatHub.ClearChannelHistory(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleChannelTurnLedger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	entries := chatHub.GetChannelTurnLedgerRaw(channel, limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"channel": channel,
		"limit":   limit,
		"count":   len(entries),
		"entries": entries,
	})
}

func handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.DeleteChannel(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleChannelAgentsManage(w http.ResponseWriter, r *http.Request) {
	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, channelName); !ok {
			return
		}
		var req struct {
			AgentIDs []string `json:"agent_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, id := range req.AgentIDs {
			if err := chatHub.AddAgentToChannel(id, channelName); err != nil {
				log.Printf("Warning: failed to add agent %s to %s: %v", id, channelName, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, channelName); !ok {
			return
		}
		agentID := r.URL.Query().Get("agent_id")
		if agentID == "" {
			http.Error(w, "agent_id query parameter required", http.StatusBadRequest)
			return
		}
		if err := chatHub.RemoveAgentFromChannel(agentID, channelName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id query parameter required", http.StatusBadRequest)
		return
	}

	channels := chatHub.GetAgentChannels(agentID)
	if channels == nil {
		channels = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"channels": channels})
}
