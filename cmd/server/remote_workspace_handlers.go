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
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ws)
}

func handleDevcontainerPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	if wsID == "" {
		http.Error(w, "workspace required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(wsID)
	if !ok {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}
	cfg, err := devcontainer.Load(ws.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan := devcontainer.PlanFromConfig(ws.Path, cfg)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(plan)
}
