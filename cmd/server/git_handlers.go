package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/workspacefiles"
	"github.com/camronwood/neural-junkie/internal/workspacesymbols"
)

func requireSoftwareDevPack(w http.ResponseWriter) bool {
	if appConfig == nil || !appConfig.IsPackEnabled(config.PackSoftwareDevelopment) {
		http.Error(w, "Software development pack required for git operations", http.StatusForbidden)
		return false
	}
	return true
}

func resolveWorkspaceForGit(w http.ResponseWriter, r *http.Request) (*hub.Workspace, bool) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	if workspaceID == "" {
		var body struct {
			WorkspaceID string `json:"workspace_id"`
		}
		if r.Body != nil && r.Method != http.MethodGet {
			_ = json.NewDecoder(r.Body).Decode(&body)
			workspaceID = strings.TrimSpace(body.WorkspaceID)
		}
	}
	if workspaceID == "" {
		http.Error(w, "workspace parameter required", http.StatusBadRequest)
		return nil, false
	}
	ws, ok := workspaceManager.GetWorkspace(workspaceID)
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return nil, false
	}
	return ws, true
}

func handleGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	ws, ok := resolveWorkspaceForGit(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	st, err := git.Status(ctx, ws.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func handleGitDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	ws, ok := resolveWorkspaceForGit(w, r)
	if !ok {
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}
	staged := strings.EqualFold(r.URL.Query().Get("staged"), "true") ||
		strings.EqualFold(r.URL.Query().Get("cached"), "true")
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	diff, err := git.Diff(ctx, ws.Path, path, staged)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"diff": diff})
}

func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string   `json:"workspace_id"`
		Message     string   `json:"message"`
		Paths       []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := git.Commit(ctx, ws.Path, req.Message, req.Paths); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGitPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := git.Push(ctx, ws.Path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGitPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := git.Pull(ctx, ws.Path); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGitFileSides(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	ws, ok := resolveWorkspaceForGit(w, r)
	if !ok {
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}
	staged := strings.EqualFold(r.URL.Query().Get("staged"), "true")
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	orig, mod, err := git.FileSides(ctx, ws.Path, path, staged)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"original": orig, "modified": mod})
}

func handleGitAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string   `json:"workspace_id"`
		Paths       []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := git.Add(ctx, ws.Path, req.Paths); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGitReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	var req struct {
		WorkspaceID string   `json:"workspace_id"`
		Paths       []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(strings.TrimSpace(req.WorkspaceID))
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := git.ResetUnstage(ctx, ws.Path, req.Paths); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleWorkspaceFileSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parsePositiveInt(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if workspaceID == "" {
		http.Error(w, "workspace parameter required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(workspaceID)
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	paths, err := workspacefiles.Search(ctx, ws.Path, q, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"paths": paths})
}

func handleWorkspaceSymbolSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parsePositiveInt(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if workspaceID == "" {
		http.Error(w, "workspace parameter required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(workspaceID)
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	syms, err := workspacesymbols.Search(ctx, ws.Path, q, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"symbols": syms})
}

func parsePositiveInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}
