package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaybeAutoApproveIDEFileChange_yoloTrustAccepted(t *testing.T) {
	msg := &protocol.Message{
		From: protocol.AgentInfo{ID: "frontend"},
		Metadata: map[string]interface{}{
			"editor_mode":        "agent",
			"editor_agent_trust": "yolo",
		},
	}
	if msg.EditorAgentTrust() != "yolo" {
		t.Fatalf("trust = %q", msg.EditorAgentTrust())
	}
	if msg.IdeEditorMode() == "ask" {
		t.Fatal("expected agent mode")
	}
}

func TestMaybeAutoApproveIDEFileChange_askModeSkipped(t *testing.T) {
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"editor_mode":        "ask",
			"editor_agent_trust": "yolo",
		},
	}
	if msg.IdeEditorMode() != "ask" {
		t.Fatal("expected ask mode blocks auto-approve path")
	}
}
