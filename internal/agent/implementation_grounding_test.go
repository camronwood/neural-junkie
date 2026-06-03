package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAttachIdeSessionMetadataToProposal(t *testing.T) {
	t.Parallel()
	user := protocol.NewMessage(protocol.MessageTypeChat, "dev", protocol.AgentInfo{ID: "u1", Name: "User"}, "implement x")
	user.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"editor_agent_trust":     "auto_apply_edits",
		"implementation_session": true,
	}
	proposalMsg := protocol.NewMessage(protocol.MessageTypeFileChange, "dev", protocol.AgentInfo{Name: "BackendEngineer"}, "proposing")
	attachIdeSessionMetadataToProposal(proposalMsg, user)
	if proposalMsg.EditorAgentTrust() != "auto_apply_edits" {
		t.Fatalf("trust: got %q", proposalMsg.EditorAgentTrust())
	}
	if proposalMsg.IdeEditorMode() != "agent" {
		t.Fatalf("mode: got %q", proposalMsg.IdeEditorMode())
	}
	if !proposalMsg.ImplementationSession() {
		t.Fatal("expected implementation_session on proposal message")
	}
}
