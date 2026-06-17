package agent

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// withWorkspaceBackendFromMessage attaches a remote workspace backend when sidecar metadata is present.
func withWorkspaceBackendFromMessage(ctx context.Context, msg *protocol.Message) context.Context {
	if msg == nil || msg.Metadata == nil {
		return ctx
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return ctx
	}
	sidecarURL, _ := raw["sidecar_url"].(string)
	sidecarURL = strings.TrimSpace(sidecarURL)
	if sidecarURL == "" {
		return ctx
	}
	root, _ := raw["workspace_path"].(string)
	root = strings.TrimSpace(root)
	if root == "" {
		return ctx
	}
	token, _ := raw["sidecar_token"].(string)
	remote := workspacebackend.NewRemote(root, sidecarURL, strings.TrimSpace(token))
	return shared.ContextWithBackend(ctx, remote)
}
