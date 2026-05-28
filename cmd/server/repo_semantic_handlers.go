package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/workspacefiles"
)

// handleRepoSemanticSearch returns file chunks for @codebase (keyword search; embeddings later).
func handleRepoSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RepoPath string `json:"repo_path"`
		Query    string `json:"query"`
		Limit    int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	root := strings.TrimSpace(req.RepoPath)
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
	paths, err := workspacefiles.Search(ctx, root, q, limit*3)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	type chunk struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	var chunks []chunk
	for _, rel := range paths {
		if len(chunks) >= limit {
			break
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		b, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(b)
		if len(content) > 4000 {
			content = content[:4000] + "\n…"
		}
		chunks = append(chunks, chunk{Path: rel, Content: content})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"chunks": chunks})
}
