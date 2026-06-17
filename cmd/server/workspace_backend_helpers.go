package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/remotetokens"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

type hubWorkspaceSource struct {
	m *hub.WorkspaceManager
}

func (h hubWorkspaceSource) GetWorkspace(id string) (workspacebackend.WorkspaceRecord, bool) {
	ws, ok := h.m.GetWorkspace(id)
	if !ok || ws == nil {
		return workspacebackend.WorkspaceRecord{}, false
	}
	return workspacebackend.WorkspaceRecord{
		Path:       ws.Path,
		Kind:       ws.Kind,
		SidecarURL: ws.SidecarURL,
	}, true
}

func backendForWorkspace(workspaceID string) (workspacebackend.Backend, int, string) {
	if workspaceBackendResolver == nil {
		return nil, http.StatusServiceUnavailable, "workspace backend not configured"
	}
	b, err := workspaceBackendResolver.ForWorkspace(workspaceID)
	if err != nil {
		return nil, http.StatusNotFound, err.Error()
	}
	return b, 0, ""
}

func isRemoteWorkspace(ws *hub.Workspace) bool {
	if ws == nil {
		return false
	}
	k := strings.TrimSpace(ws.Kind)
	return k == workspacebackend.KindSSH || k == workspacebackend.KindDevcontainer
}

func backendListDir(ctx context.Context, workspaceID, rel string) ([]workspacebackend.Entry, int, string) {
	b, code, msg := backendForWorkspace(workspaceID)
	if b == nil {
		return nil, code, msg
	}
	entries, err := b.ReadDir(ctx, strings.TrimPrefix(rel, "/"))
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	return entries, 0, ""
}

func backendReadFile(ctx context.Context, workspaceID, rel string) ([]byte, int, string) {
	b, code, msg := backendForWorkspace(workspaceID)
	if b == nil {
		return nil, code, msg
	}
	data, err := b.ReadFile(ctx, strings.TrimPrefix(rel, "/"))
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}
	return data, 0, ""
}

func backendWriteFile(ctx context.Context, workspaceID, rel string, data []byte) (int, string) {
	b, code, msg := backendForWorkspace(workspaceID)
	if b == nil {
		return code, msg
	}
	if err := b.WriteFile(ctx, strings.TrimPrefix(rel, "/"), data); err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	return 0, ""
}

func registerRemoteWorkspacesOnStartup() {
	if workspaceBackendResolver == nil || workspaceManager == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, ws := range workspaceManager.ListWorkspaces() {
		if !isRemoteWorkspace(ws) || strings.TrimSpace(ws.SidecarURL) == "" {
			continue
		}
		token, _ := remotetokens.Get(ws.ID)
		remote := workspacebackend.NewRemote(ws.Path, ws.SidecarURL, token)
		if err := workspacebackend.HealthCheck(ctx, remote); err != nil {
			log.Printf("[remote] sidecar health failed for %s (%s): %v", ws.Name, ws.SidecarURL, err)
			continue
		}
		workspaceBackendResolver.RegisterRemote(ws.ID, remote)
	}
}

func backendStat(ctx context.Context, workspaceID, rel string) (interface{}, error) {
	b, code, msg := backendForWorkspace(workspaceID)
	if b == nil {
		return nil, fmt.Errorf("%s", msg)
	}
	if code != 0 {
		return nil, fmt.Errorf("%s", msg)
	}
	return b.Stat(ctx, strings.TrimPrefix(rel, "/"))
}

func backendForWorkspaceRoot(root string) filechange.WorkspaceIO {
	if workspaceBackendResolver == nil || workspaceManager == nil || root == "" {
		return nil
	}
	ws, ok := workspaceManager.FindWorkspaceByPath(root)
	if !ok || !isRemoteWorkspace(ws) {
		return nil
	}
	b, err := workspaceBackendResolver.ForWorkspace(ws.ID)
	if err != nil {
		return nil
	}
	return filechange.BackendIO{Backend: b}
}
