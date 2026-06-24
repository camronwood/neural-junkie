package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

func handleAgentsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetAgents(w, r)
	case http.MethodPost:
		handleRegisterAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetAgents(w http.ResponseWriter, r *http.Request) {
	agents := chatHub.ListAgents()
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_tool_counts")), "true") {
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok && ch != nil {
			for i := range agents {
				if agents[i] != nil {
					agents[i].ToolCount = ch.ToolCountForAgent(agents[i].ID)
				}
			}
		}
	}

	json.NewEncoder(w).Encode(agents)
}

func handleAgentTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	if agentID == "" {
		http.Error(w, "agent_id query parameter required", http.StatusBadRequest)
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		http.Error(w, "command handler unavailable", http.StatusInternalServerError)
		return
	}
	cap, err := ch.GetAgentToolCapabilities(agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cap)
}

func handleChannelTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}
	ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler)
	if !ok || ch == nil {
		http.Error(w, "command handler unavailable", http.StatusInternalServerError)
		return
	}
	resp, err := ch.ListChannelToolCapabilities(channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	var agent protocol.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := chatHub.RegisterAgent(&agent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "agent_id": agent.ID})
}

func handleCachedAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get cached agents from all storage types
	cachedAgents, err := getAllCachedAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"cached_agents": cachedAgents,
	}

	json.NewEncoder(w).Encode(response)
}

func handleMyAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		myAgents, err := getAllCachedAgents()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"my_agents": myAgents})
	case http.MethodDelete:
		var req struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		deleted, err := hub.DeleteCachedAgent(req.Type, req.Name, req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !deleted {
			http.Error(w, "cached agent not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAllCachedAgents aggregates cached agents from all storage types

func getAllCachedAgents() ([]map[string]interface{}, error) {
	var allAgents []map[string]interface{}

	// Get cached repository agents
	repoStorage, err := repo.NewStorage()
	if err == nil {
		repoAgents, err := repoStorage.GetAllCachedRepos()
		if err == nil {
			allAgents = append(allAgents, repoAgents...)
		}
	}

	// TODO: Add confluence agents when storage is implemented
	// confluenceAgents, err := confluenceStorage.GetAllCachedSpaces()
	// if err == nil {
	//     allAgents = append(allAgents, confluenceAgents...)
	// }

	// Get cached CLI agents
	cliStorage, err := agent.NewCLIAgentStorage()
	if err == nil {
		cliAgents, err := cliStorage.ListWithMetadata()
		if err == nil {
			allAgents = append(allAgents, cliAgents...)
		}
	}

	return allAgents, nil
}

func handleRemovedAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get removed agents from hub
	removedAgents := chatHub.GetRemovedAgents()

	response := map[string]interface{}{
		"removed_agents": removedAgents,
	}

	json.NewEncoder(w).Encode(response)
}

// handleImport handles agent import requests from CLI

func handleAgentProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID and action from URL path: /api/agents/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}

	action := parts[1]

	// Route to approval-mode handler
	if action == "approval-mode" {
		handleSetApprovalMode(w, r, parts[0])
		return
	}

	if action == "rules" {
		handleSetAgentCustomRules(w, r, parts[0])
		return
	}

	if action != "provider" {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}

	agentID := parts[0]
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	var request struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider
	if !isAllowedRuntimeProvider(request.Provider) {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', 'lmstudio', or 'huggingface'", http.StatusBadRequest)
		return
	}

	commandHandler := chatHub.GetCommandHandler()
	if commandHandler == nil {
		http.Error(w, "Command handler not initialized", http.StatusServiceUnavailable)
		return
	}
	ch, ok := commandHandler.(*hub.CommandHandler)
	if !ok {
		http.Error(w, "Unsupported command handler type", http.StatusInternalServerError)
		return
	}

	targetAgent, err := ch.SwitchAgentProvider(agentID, request.Provider, request.Model, "general", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent %s switched to %s (%s)", targetAgent.Name, targetAgent.AIProvider, targetAgent.AIModel),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSwitchAllProviders handles switching all agents to the same provider

func handleSwitchAllProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider
	if !isAllowedRuntimeProvider(request.Provider) {
		http.Error(w, "Invalid provider. Use 'claude', 'ollama', 'lmstudio', or 'huggingface'", http.StatusBadRequest)
		return
	}

	commandHandler := chatHub.GetCommandHandler()
	if commandHandler == nil {
		http.Error(w, "Command handler not initialized", http.StatusServiceUnavailable)
		return
	}
	ch, ok := commandHandler.(*hub.CommandHandler)
	if !ok {
		http.Error(w, "Unsupported command handler type", http.StatusInternalServerError)
		return
	}

	switchedCount, err := ch.SwitchAllProviders(request.Provider, request.Model, "general", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success":        true,
		"message":        fmt.Sprintf("Switched %d agents to %s (%s)", switchedCount, request.Provider, request.Model),
		"switched_count": switchedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOllamaStatus checks if Ollama is running
