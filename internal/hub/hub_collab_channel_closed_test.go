package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClosedCollaborationChannelRejectsChat(t *testing.T) {
	h := NewHub()
	_ = h.RegisterAgent(&protocol.AgentInfo{ID: "a1", Name: "A1", Type: protocol.AgentTypeBackend, Status: "active"})
	_ = h.RegisterAgent(&protocol.AgentInfo{ID: "a2", Name: "A2", Type: protocol.AgentTypeFrontend, Status: "active"})
	cm := h.GetCollaborationManager()
	chName := "collab-closed-test"

	collab, err := cm.CreateCollaboration("done", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{
		MaxRounds:        1,
		TurnBudget:       1,
		MaxTotalMessages: 2,
	}, collaboration.CreateOptions{SkipDiscussion: true})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	_ = cm.BindCollaborationChannel(collab.ID, chName)
	h.CreateChannelWithType(chName, "Collab", "", protocol.ChannelTypeCollaboration, "")

	if _, err := cm.CancelCollaboration(collab.ID); err != nil {
		t.Fatalf("CancelCollaboration: %v", err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, chName, protocol.AgentInfo{
		Name: "User", Type: "human",
	}, "hello after close")
	if err := h.SendMessage(msg); err == nil {
		t.Fatal("expected send to closed collab channel to fail")
	}

	cmd := protocol.NewMessage(protocol.MessageTypeQuestion, chName, protocol.AgentInfo{
		Name: "User", Type: "human",
	}, "/collab-status")
	if err := h.SendMessage(cmd); err != nil {
		t.Fatalf("slash command on closed channel should work: %v", err)
	}
}
