package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/codeintel"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// handleRepoSemanticSearch returns file chunks for @codebase (hybrid embed + keyword).
func handleRepoSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		WorkspaceID string   `json:"workspace_id"`
		RepoPath    string   `json:"repo_path"`
		RepoPaths   []string `json:"repo_paths"`
		Query       string   `json:"query"`
		Limit       int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}
	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 8
	}

	paths := normalizeRepoSearchPaths(req.RepoPaths, req.RepoPath, req.WorkspaceID)
	if len(paths) == 0 {
		http.Error(w, "repo_path or repo_paths required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	for _, root := range paths {
		meta, _ := codeindex.Status(root)
		if !meta.Ready && !meta.Building {
			codeindex.BuildIndexAsync(root)
		}
	}

	limitPerRepo := 4
	totalLimit := 12
	if len(paths) == 1 {
		limitPerRepo = limit
		totalLimit = limit
	}

	hits, err := codeintel.SemanticSearchMultiViaBackend(ctx, paths, func(path string) workspacebackend.Backend {
		if req.WorkspaceID == "" || workspaceManager == nil {
			return nil
		}
		ws, ok := workspaceManager.GetWorkspace(req.WorkspaceID)
		if !ok || ws == nil || ws.Path != path {
			return nil
		}
		if b, berr := workspaceBackendResolver.ForWorkspace(req.WorkspaceID); berr == nil {
			return b
		}
		return nil
	}, q, limitPerRepo, totalLimit)
	if err != nil && len(hits) == 0 && len(paths) == 1 {
		// Fallback single-path error for backward compatibility.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type chunk struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		RepoPath string `json:"repo_path,omitempty"`
		RepoName string `json:"repo_name,omitempty"`
	}
	chunks := make([]chunk, len(hits))
	for i, h := range hits {
		chunks[i] = chunk{
			Path:     h.Path,
			Content:  h.Content,
			RepoPath: h.RepoPath,
			RepoName: h.RepoName,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"chunks": chunks})
}

func normalizeRepoSearchPaths(repoPaths []string, singlePath, workspaceID string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range repoPaths {
		add(p)
	}
	if single := strings.TrimSpace(singlePath); single != "" {
		add(single)
	}
	if len(out) == 0 && workspaceID != "" {
		if ws, ok := workspaceManager.GetWorkspace(workspaceID); ok && ws != nil {
			add(ws.Path)
		}
	}
	return out
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
