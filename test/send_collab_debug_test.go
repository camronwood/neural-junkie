package test

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSendCollaborateViaMessage(t *testing.T) {
	h := hub.NewHub()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SecurityExpert", Type: protocol.AgentTypeSecurity, Status: "active"}
	if err := h.RegisterAgent(a1); err != nil {
		t.Fatal(err)
	}
	if err := h.RegisterAgent(a2); err != nil {
		t.Fatal(err)
	}
	content := "/collaborate @RustExpert @SecurityExpert build a secure CLI"
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "Tester", Type: "human"},
		content,
	)
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage err: %v", err)
	}
	active := h.GetCollaborationManager().ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active collaboration, got %d", len(active))
	}
}
