package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/google/uuid"
)

// handleUserServersRoute — MCP Tool Wizard remote MCP registry (persist only; dial later).
//
//	GET    /api/mcp/user-servers
//	POST   /api/mcp/user-servers
//	DELETE /api/mcp/user-servers/{id}
func handleUserServersRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mcp/user-servers"), "/")
	var parts []string
	if path != "" {
		parts = strings.Split(path, "/")
	}

	switch {
	case len(parts) == 0:
		handleUserServersCollection(w, r)
	case len(parts) == 1:
		handleUserServerByID(w, r, parts[0])
	default:
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
	}
}

func handleUserServersCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appConfig.MCP.UserServers)

	case http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var in config.UserMCPServer
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
		if strings.TrimSpace(in.TransportType) == "" {
			in.TransportType = "http"
		}
		in.ID = uuid.NewString()
		in.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		appConfig.MCP.UserServers = append(appConfig.MCP.UserServers, in)
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

func handleUserServerByID(w http.ResponseWriter, r *http.Request, id string) {
	idx := -1
	for i, s := range appConfig.MCP.UserServers {
		if s.ID == id {
			idx = i
			break
		}
	}

	switch r.Method {
	case http.MethodGet:
		if idx < 0 {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appConfig.MCP.UserServers[idx])

	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		if idx < 0 {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		appConfig.MCP.UserServers = append(appConfig.MCP.UserServers[:idx], appConfig.MCP.UserServers[idx+1:]...)
		if err := saveUserToolsConfig(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
