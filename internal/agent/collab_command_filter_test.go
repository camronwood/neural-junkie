package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestFilterCollabCommandSuggestions_dropsStackToolsForMarkdownTask(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Write collabs/abc/api_schema.md defining the API schema.",
	)
	msg.SetCollaborationPhase("executing")

	in := []protocol.CommandSuggestion{
		{Command: "docker-compose up -d"},
		{Command: "npm install"},
		{Command: "cat resource-api/json_endpoints/products.json"},
	}
	out := filterCollabCommandSuggestions(msg, in)
	if len(out) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %#v", len(out), out)
	}
	if out[0].Command != "cat resource-api/json_endpoints/products.json" {
		t.Fatalf("unexpected kept command %q", out[0].Command)
	}
}

func TestFilterCollabCommandSuggestions_keepsCommandsOutsideExecution(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Write collabs/abc/api_schema.md defining the API schema.",
	)
	msg.SetCollaborationPhase("planning")

	in := []protocol.CommandSuggestion{{Command: "npm test"}}
	out := filterCollabCommandSuggestions(msg, in)
	if len(out) != 1 {
		t.Fatalf("expected planning task to pass through, got %d", len(out))
	}
}
