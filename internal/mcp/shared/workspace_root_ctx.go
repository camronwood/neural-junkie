package shared

import (
	"context"
	"strings"
)

type workspaceRootKey struct{}

// ContextWithWorkspaceRoot attaches the turn's workspace root for MCP file tools.
// Prefer this over mutable agent.WorkspacePath so concurrent turns cannot race the root.
func ContextWithWorkspaceRoot(ctx context.Context, root string) context.Context {
	root = strings.TrimSpace(root)
	if root == "" {
		return ctx
	}
	return context.WithValue(ctx, workspaceRootKey{}, root)
}

// WorkspaceRootFromContext returns the workspace root when set on the context.
func WorkspaceRootFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	root, _ := ctx.Value(workspaceRootKey{}).(string)
	return strings.TrimSpace(root)
}
