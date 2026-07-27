package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp/usertools"
	"github.com/google/uuid"
)

// handleUserToolsRoute implements the MCP Tool Wizard's minimal API:
//
//	GET    /api/mcp/user-tools           list all user-defined tools
//	POST   /api/mcp/user-tools           create a tool
//	GET    /api/mcp/user-tools/{id}      get one tool
//	PUT    /api/mcp/user-tools/{id}      update a tool
//	DELETE /api/mcp/user-tools/{id}      delete a tool
//	POST   /api/mcp/user-tools/{id}/test run the tool once with sample args
//	POST   /api/mcp/user-tools/{id}/grant grant/revoke access for an agent by name
func handleUserToolsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mcp/user-tools"), "/")
	var parts []string
	if path != "" {
		parts = strings.Split(path, "/")
	}

	switch {
	case len(parts) == 0:
		handleUserToolsCollection(w, r)
	case len(parts) == 1:
		handleUserToolByID(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "test":
		handleTestUserTool(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "grant":
		handleGrantUserTool(w, r, parts[0])
	default:
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
	}
}

func handleUserToolsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appConfig.MCP.UserTools)

	case http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var in config.UserMCPTool
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		in.Name = strings.TrimSpace(in.Name)
		in.URL = strings.TrimSpace(in.URL)
		if in.Name == "" || in.URL == "" {
			http.Error(w, "name and url are required", http.StatusBadRequest)
			return
		}
		in.ID = uuid.NewString()
		in.CreatedAt = time.Now().UTC().Format(time.RFC3339)

		appConfig.MCP.UserTools = append(appConfig.MCP.UserTools, in)
		if err := saveUserToolsConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(in)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUserToolByID(w http.ResponseWriter, r *http.Request, id string) {
	idx := findUserToolIndex(id)

	switch r.Method {
	case http.MethodGet:
		if idx < 0 {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appConfig.MCP.UserTools[idx])

	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		if idx < 0 {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}
		var in config.UserMCPTool
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		existing := appConfig.MCP.UserTools[idx]
		in.ID = existing.ID
		in.CreatedAt = existing.CreatedAt
		if strings.TrimSpace(in.Name) == "" {
			in.Name = existing.Name
		}
		if strings.TrimSpace(in.URL) == "" {
			in.URL = existing.URL
		}
		appConfig.MCP.UserTools[idx] = in
		if err := saveUserToolsConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(in)

	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		if idx < 0 {
			http.Error(w, "tool not found", http.StatusNotFound)
			return
		}
		appConfig.MCP.UserTools = append(appConfig.MCP.UserTools[:idx], appConfig.MCP.UserTools[idx+1:]...)
		if err := saveUserToolsConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTestUserTool runs a tool once with optional sample args, so the
// wizard can preview output before granting it to any agent.
func handleTestUserTool(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idx := findUserToolIndex(id)
	if idx < 0 {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}
	var req struct {
		Query string `json:"query"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result, err := usertools.Execute(ctx, nil, appConfig.MCP.UserTools[idx], strings.TrimSpace(req.Query))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": result})
}

// handleGrantUserTool grants or revokes an agent's (by display name) access
// to a user-defined tool. Custom expert agents pick up newly granted tools
// the next time they're (re)created; this only updates the persisted grant.
func handleGrantUserTool(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	idx := findUserToolIndex(id)
	if idx < 0 {
		http.Error(w, "tool not found", http.StatusNotFound)
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

	tool := &appConfig.MCP.UserTools[idx]
	filtered := tool.GrantedAgents[:0:0]
	for _, g := range tool.GrantedAgents {
		if !strings.EqualFold(strings.TrimSpace(g), agentName) {
			filtered = append(filtered, g)
		}
	}
	if req.Grant {
		filtered = append(filtered, agentName)
	}
	tool.GrantedAgents = filtered

	if err := saveUserToolsConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(*tool)
}

func findUserToolIndex(id string) int {
	for i, t := range appConfig.MCP.UserTools {
		if t.ID == id {
			return i
		}
	}
	return -1
}

func saveUserToolsConfig() error {
	if err := appConfig.Save(); err != nil {
		return err
	}
	config.SetAppConfig(appConfig)
	return nil
}
