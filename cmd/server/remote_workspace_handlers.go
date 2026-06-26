package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/devcontainer"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func handleConnectRemoteWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name       string `json:"name"`
		RemoteHost string `json:"remote_host"`
		RemoteUser string `json:"remote_user"`
		RemotePath string `json:"remote_path"`
		SidecarURL string `json:"sidecar_url"`
		Token      string `json:"token"`
		Kind       string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = workspacebackend.KindSSH
	}
	ws, err := workspaceManager.AddWorkspace(req.Name, req.RemotePath, hub.AddWorkspaceOptions{
		Create:        false,
		SkipPathCheck: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ws.Kind = kind
	ws.RemoteHost = req.RemoteHost
	ws.RemoteUser = req.RemoteUser
	ws.SidecarURL = strings.TrimRight(req.SidecarURL, "/")
	if err := workspaceManager.UpdateWorkspace(ws); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ws.SidecarURL != "" {
		_ = hub.SaveRemoteToken(ws.ID, req.Token)
		remote := workspacebackend.NewRemote(ws.Path, ws.SidecarURL, req.Token)
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := workspacebackend.HealthCheck(ctx, remote); err != nil {
			http.Error(w, "sidecar health check failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		workspaceBackendResolver.RegisterRemote(ws.ID, remote)
		ws.IsGitRepo = remoteGitRepo(ctx, remote)
		_ = workspaceManager.UpdateWorkspace(ws)
		ensureHiddenRepoAgentForWorkspace(ws)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ws)
}

func remoteGitRepo(ctx context.Context, b workspacebackend.Backend) bool {
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    []string{"rev-parse", "--git-dir"},
		RelCwd:  ".",
		Timeout: 15 * time.Second,
	})
	return err == nil && res.ExitCode == 0
}

func handleDevcontainerPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	repoPath := strings.TrimSpace(r.URL.Query().Get("path"))
	var cfg *devcontainer.Config
	var err error
	var root string
	ctx := r.Context()
	if wsID != "" {
		ws, ok := workspaceManager.GetWorkspace(wsID)
		if !ok {
			http.Error(w, "workspace not found", http.StatusNotFound)
			return
		}
		root = ws.Path
		if isRemoteWorkspace(ws) {
			b, berr := workspaceBackendResolver.ForWorkspace(ws.ID)
			if berr != nil {
				http.Error(w, berr.Error(), http.StatusBadGateway)
				return
			}
			cfg, err = devcontainer.LoadViaBackend(ctx, b)
		} else {
			cfg, err = devcontainer.Load(ws.Path)
		}
	} else if repoPath != "" {
		root = repoPath
		cfg, err = devcontainer.Load(repoPath)
	} else {
		http.Error(w, "workspace or path required", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan := devcontainer.PlanFromConfig(root, cfg)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(plan)
}
