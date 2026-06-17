package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func TestWithWorkspaceBackendFromMessage(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/remote/proj",
				"sidecar_url":    "http://127.0.0.1:19876",
				"sidecar_token":  "secret",
			},
		},
	}
	ctx := withWorkspaceBackendFromMessage(context.Background(), msg)
	b := shared.BackendFromContext(ctx)
	if b == nil {
		t.Fatal("expected backend in context")
	}
	if b.Root() != "/remote/proj" {
		t.Fatalf("root=%q", b.Root())
	}
	if b.Kind() != workspacebackend.KindSSH {
		t.Fatalf("kind=%q", b.Kind())
	}
}

func TestWithWorkspaceBackendFromMessageSkipsLocal(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/local/proj",
			},
		},
	}
	ctx := withWorkspaceBackendFromMessage(context.Background(), msg)
	if shared.BackendFromContext(ctx) != nil {
		t.Fatal("expected no backend without sidecar_url")
	}
}
