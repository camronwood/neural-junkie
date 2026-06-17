package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// handleRepoSemanticSearch returns file chunks for @codebase (hybrid embed + keyword).
func handleRepoSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		WorkspaceID string `json:"workspace_id"`
		RepoPath    string `json:"repo_path"`
		Query       string `json:"query"`
		Limit       int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	root := strings.TrimSpace(req.RepoPath)
	if req.WorkspaceID != "" {
		if ws, ok := workspaceManager.GetWorkspace(req.WorkspaceID); ok && ws != nil {
			root = ws.Path
		}
	}
	q := strings.TrimSpace(req.Query)
	if root == "" || q == "" {
		http.Error(w, "repo_path and query required", http.StatusBadRequest)
		return
	}
	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var backend workspacebackend.Backend
	if req.WorkspaceID != "" {
		if b, err := workspaceBackendResolver.ForWorkspace(req.WorkspaceID); err == nil {
			backend = b
		}
	}

	meta, _ := codeindex.Status(root)
	if !meta.Ready && !meta.Building {
		if backend != nil {
			go func() {
				c, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				_ = codeindex.BuildIndexViaBackend(c, root, backend)
			}()
		} else {
			codeindex.BuildIndexAsync(root)
		}
	}

	results, err := codeindex.SearchViaBackend(ctx, root, backend, q, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	type chunk struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	chunks := make([]chunk, len(results))
	for i, r := range results {
		chunks[i] = chunk{Path: r.Path, Content: r.Content}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"chunks": chunks})
}

// handleRepoIndexStatus reports code index build state for a workspace.
func handleRepoIndexStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoPath := strings.TrimSpace(r.URL.Query().Get("repo_path"))
	if repoPath == "" {
		http.Error(w, "repo_path required", http.StatusBadRequest)
		return
	}
	meta, err := codeindex.Status(repoPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}
