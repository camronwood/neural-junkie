package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handleProjectSets(w http.ResponseWriter, r *http.Request) {
	if projectSetManager == nil {
		http.Error(w, "project sets unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list := projectSetManager.ListProjectSets()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"project_sets": list})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProjectSetByID(w http.ResponseWriter, r *http.Request) {
	if projectSetManager == nil || workspaceManager == nil {
		http.Error(w, "project sets unavailable", http.StatusServiceUnavailable)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/project-sets/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name               string   `json:"name"`
			PrimaryWorkspaceID string   `json:"primary_workspace_id"`
			MemberWorkspaceIDs []string `json:"member_workspace_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		ps, err := projectSetManager.UpdateProjectSet(id, req.Name, req.PrimaryWorkspaceID, req.MemberWorkspaceIDs, workspaceManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ps)
	case http.MethodDelete:
		if err := projectSetManager.DeleteProjectSet(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreateProjectSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if projectSetManager == nil || workspaceManager == nil {
		http.Error(w, "project sets unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Name               string   `json:"name"`
		PrimaryWorkspaceID string   `json:"primary_workspace_id"`
		MemberWorkspaceIDs []string `json:"member_workspace_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ps, err := projectSetManager.CreateProjectSet(req.Name, req.PrimaryWorkspaceID, req.MemberWorkspaceIDs, workspaceManager)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ps)
}
