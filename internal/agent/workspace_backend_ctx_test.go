package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/mcp/shared"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func TestWithWorkspaceBackendFromMessage(t *testing.T) {
	remote := workspacebackend.NewRemote("/remote/proj", "http://127.0.0.1:19876", "secret")
	lookup := func(workspaceID string) workspacebackend.Backend {
		if workspaceID == "ws-remote" {
			return remote
		}
		return nil
	}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_id":   "ws-remote",
				"workspace_path": "/remote/proj",
				"sidecar_url":    "http://127.0.0.1:19876",
			},
		},
	}
	ctx := withWorkspaceBackendFromMessage(context.Background(), msg, lookup)
	b := shared.BackendFromContext(ctx)
	if b == nil {
		t.Fatal("expected backend in context")
	}
	if b.Root() != "/remote/proj" {
		t.Fatalf("root=%q", b.Root())
	}
}

func TestWithWorkspaceBackendFromMessageSkipsWithoutLookup(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/local/proj",
				"sidecar_url":    "http://127.0.0.1:19876",
			},
		},
	}
	ctx := withWorkspaceBackendFromMessage(context.Background(), msg, nil)
	if shared.BackendFromContext(ctx) != nil {
		t.Fatal("expected no backend without workspace_id lookup")
	}
}

func TestRedactSidecarSecrets(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"sidecar_token": "secret",
			},
		},
	}
	RedactSidecarSecrets(msg)
	ws := msg.Metadata["workspace_context"].(map[string]interface{})
	if _, ok := ws["sidecar_token"]; ok {
		t.Fatal("sidecar_token should be redacted")
	}
}
