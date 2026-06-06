package hub

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNotifyFileChangeApproved_agentVisibleMessage(t *testing.T) {
	t.Parallel()
	h := NewHub()
	change := &filechange.FileChange{
		ID:        "abc123",
		Operation: filechange.FileOperationEdit,
		FilePath:  "src/App.tsx",
		Channel:   "general",
		Agent:     protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
	}
	h.NotifyFileChangeApproved(change, "camron")

	msgs := h.messages["general"]
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Type != protocol.MessageTypeSystemInfo {
		t.Fatalf("first message type = %q", msgs[0].Type)
	}
	approval := msgs[1]
	if approval.Type != protocol.MessageTypeChat {
		t.Fatalf("approval message type = %q", approval.Type)
	}
	if !approval.FileChangeApproved() {
		t.Fatal("expected file_change_approved metadata")
	}
	if approval.FileChangeApprovalAgentID() != "fe-1" {
		t.Fatalf("agent id = %q", approval.FileChangeApprovalAgentID())
	}
	if !strings.Contains(approval.Content, "src/App.tsx") {
		t.Fatalf("approval content = %q", approval.Content)
	}
	if !approval.IsMentioned("fe-1") {
		t.Fatal("expected agent mention on approval message")
	}
}
