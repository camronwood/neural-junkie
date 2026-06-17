package agent

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// SetWorkspaceBackendLookup wires hub-side backend resolution (includes sidecar token).
func (a *Agent) SetWorkspaceBackendLookup(fn func(workspaceID string) workspacebackend.Backend) {
	if a == nil {
		return
	}
	a.workspaceBackendLookup = fn
}

func (a *Agent) contextWithWorkspaceBackend(ctx context.Context, msg *protocol.Message) context.Context {
	return withWorkspaceBackendFromMessage(ctx, msg, a.workspaceBackendLookup)
}

// withWorkspaceBackendFromMessage attaches a remote workspace backend when routing metadata is present.
func withWorkspaceBackendFromMessage(
	ctx context.Context,
	msg *protocol.Message,
	lookup func(workspaceID string) workspacebackend.Backend,
) context.Context {
	if msg == nil || msg.Metadata == nil {
		return ctx
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return ctx
	}
	if lookup != nil {
		if id, _ := raw["workspace_id"].(string); strings.TrimSpace(id) != "" {
			if b := lookup(strings.TrimSpace(id)); b != nil {
				return shared.ContextWithBackend(ctx, b)
			}
		}
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
	// No token in metadata — lookup by workspace_id is required for authenticated sidecars.
	return ctx
}

// RedactSidecarSecrets removes sidecar bearer tokens from message metadata before client export.
func RedactSidecarSecrets(msg *protocol.Message) {
	if msg == nil || msg.Metadata == nil {
		return
	}
	raw, ok := msg.Metadata["workspace_context"].(map[string]interface{})
	if !ok {
		return
	}
	delete(raw, "sidecar_token")
	msg.Metadata["workspace_context"] = raw
}
