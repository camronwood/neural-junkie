package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendChannelMessageDedupeByID(t *testing.T) {
	h := NewHub()
	ch := "dedupe-test"
	_ = h.CreateChannel(ch, "c", "test")

	from := protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral}
	m1 := protocol.NewMessage(protocol.MessageTypeQuestion, ch, from, "first")
	m1.ID = "fixed-id-1"

	h.mu.Lock()
	h.appendChannelMessageLocked(ch, m1)
	h.appendChannelMessageLocked(ch, m1)
	h.mu.Unlock()

	msgs, _ := h.GetMessages(ch, 10)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after duplicate append, got %d", len(msgs))
	}
	if msgs[0].Content != "first" {
		t.Fatalf("unexpected content: %q", msgs[0].Content)
	}

	m1.Content = "updated"
	h.mu.Lock()
	h.appendChannelMessageLocked(ch, m1)
	h.mu.Unlock()

	msgs, _ = h.GetMessages(ch, 10)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after upsert, got %d", len(msgs))
	}
	if msgs[0].Content != "updated" {
		t.Fatalf("expected upserted content, got %q", msgs[0].Content)
	}
}

func TestJoinChannelSkipsDuplicateJoinAnnouncement(t *testing.T) {
	h := NewHub()
	ch := "join-dedupe"
	_ = h.CreateChannel(ch, "c", "test")
	agent := &protocol.AgentInfo{ID: "a1", Name: "Cursor", Type: protocol.AgentTypeCLI, Status: "active"}
	_ = h.RegisterAgent(agent)

	if err := h.JoinChannel(agent.ID, ch); err != nil {
		t.Fatalf("first join: %v", err)
	}
	if err := h.JoinChannel(agent.ID, ch); err != nil {
		t.Fatalf("second join: %v", err)
	}

	msgs, _ := h.GetMessages(ch, 20)
	joins := 0
	for _, m := range msgs {
		if m.Type == protocol.MessageTypeAgentJoin {
			joins++
		}
	}
	if joins != 1 {
		t.Fatalf("expected 1 agent_join, got %d", joins)
	}
}
