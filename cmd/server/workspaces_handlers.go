package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/pathutil"
)

func handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if removed, err := workspaceManager.PruneUnavailableTestWorkspaces(); err != nil {
			log.Printf("Warning: failed to prune unavailable test workspaces: %v", err)
		} else if removed > 0 {
			log.Printf("Pruned %d unavailable test workspace(s) from registry", removed)
		}
		workspaces := workspaceManager.ListWorkspaces()
		json.NewEncoder(w).Encode(workspaces)
	case "POST":
		var req struct {
			Name       string `json:"name"`
			Path       string `json:"path"`
			Create     bool   `json:"create"`
			ParentPath string `json:"parent_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		workspace, err := workspaceManager.AddWorkspace(req.Name, req.Path, hub.AddWorkspaceOptions{
			Create:     req.Create,
			ParentPath: req.ParentPath,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if workspace.Path != "" {
			codeindex.BuildIndexAsync(workspace.Path)
		}
		json.NewEncoder(w).Encode(workspace)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id parameter required", http.StatusBadRequest)
			return
		}
		if err := workspaceManager.RemoveWorkspace(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFileCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		Path        string `json:"path"`
		Content     string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, req.Path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	// Check if file already exists
	if _, err := os.Stat(absPath); err == nil {
		http.Error(w, "File already exists", http.StatusConflict)
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleFileRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	oldFullPath := filepath.Join(workspace.Path, req.OldPath)
	newFullPath := filepath.Join(workspace.Path, req.NewPath)

	oldAbsPath, err := pathutil.WithinRoot(workspace.Path, oldFullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}
	newAbsPath, err := pathutil.WithinRoot(workspace.Path, newFullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	if err := os.Rename(oldAbsPath, newAbsPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.URL.Query().Get("workspace")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	if err := os.RemoveAll(absPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// File change API handlers
