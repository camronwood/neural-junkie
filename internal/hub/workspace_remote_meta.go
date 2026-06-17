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
	raw["workspace_id"] = ws.ID
	raw["workspace_kind"] = ws.Kind
	if strings.TrimSpace(ws.SidecarURL) != "" {
		raw["sidecar_url"] = ws.SidecarURL
	}
	if token, err := GetRemoteToken(ws.ID); err == nil && strings.TrimSpace(token) != "" {
		raw["sidecar_token"] = token
	}
	msg.Metadata["workspace_context"] = raw
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
