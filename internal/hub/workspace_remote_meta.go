package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// enrichRemoteWorkspaceMetadata adds sidecar routing fields to workspace_context for agent MCP tools.
func (h *Hub) enrichRemoteWorkspaceMetadata(msg *protocol.Message) {
	if h == nil || h.workspaceManager == nil || msg == nil || msg.Metadata == nil {
		return
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return
	}
	ws := h.workspaceFromContext(raw)
	if ws == nil || !isRemoteWorkspaceRecord(ws) {
		return
	}
	h.applyRemoteWorkspaceRoutingFields(raw, ws)
	msg.Metadata["workspace_context"] = raw
}

// applyRemoteWorkspaceRoutingFields sets workspace_id/sidecar_url on a workspace_context map (no secrets).
func (h *Hub) applyRemoteWorkspaceRoutingFields(ctx map[string]interface{}, ws *Workspace) {
	if ctx == nil || ws == nil || !isRemoteWorkspaceRecord(ws) {
		return
	}
	ctx["workspace_id"] = ws.ID
	ctx["workspace_kind"] = ws.Kind
	if strings.TrimSpace(ws.SidecarURL) != "" {
		ctx["sidecar_url"] = ws.SidecarURL
	}
}

func (h *Hub) remoteWorkspaceForRepoPath(repoPath string) *Workspace {
	if h == nil || h.workspaceManager == nil {
		return nil
	}
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil
	}
	if ws, ok := h.workspaceManager.FindWorkspaceByPath(repoPath); ok && isRemoteWorkspaceRecord(ws) {
		return ws
	}
	return nil
}

func (h *Hub) workspaceFromContext(ctx map[string]interface{}) *Workspace {
	if id, _ := ctx["workspace_id"].(string); strings.TrimSpace(id) != "" {
		if ws, ok := h.workspaceManager.GetWorkspace(strings.TrimSpace(id)); ok {
			return ws
		}
	}
	path, _ := ctx["workspace_path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if ws, ok := h.workspaceManager.FindWorkspaceByPath(path); ok {
		return ws
	}
	return nil
}

func isRemoteWorkspaceRecord(ws *Workspace) bool {
	if ws == nil {
		return false
	}
	k := strings.TrimSpace(ws.Kind)
	return k == workspacebackend.KindSSH || k == workspacebackend.KindDevcontainer
}
