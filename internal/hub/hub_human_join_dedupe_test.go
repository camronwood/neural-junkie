package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHumanJoinDedup_skipsRepeatWithinWindow(t *testing.T) {
	h := NewHub()
	ch := "general"
	_ = h.CreateChannel(ch, "c", "test")

	from := protocol.AgentInfo{ID: "user-camron", Name: "Camron", Type: protocol.AgentTypeGeneral}
	join := func(content string) {
		m := protocol.NewMessage(protocol.MessageTypeSystemInfo, ch, from, content)
		if err := h.SendMessage(m); err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
	}

	join("Camron has joined the chat")
	join("Camron has joined the chat")

	msgs, _ := h.GetMessages(ch, 20)
	joins := 0
	for _, m := range msgs {
		if m != nil && isHumanJoinAnnouncement(m) {
			joins++
		}
	}
	if joins != 1 {
		t.Fatalf("expected 1 human join, got %d", joins)
	}
}

func TestHumanJoinDedup_allowsDifferentUsers(t *testing.T) {
	h := NewHub()
	ch := "general"
	_ = h.CreateChannel(ch, "c", "test")

	a := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		ch,
		protocol.AgentInfo{ID: "u1", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"Camron has joined the chat",
	)
	b := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		ch,
		protocol.AgentInfo{ID: "u2", Name: "Alex", Type: protocol.AgentTypeGeneral},
		"Alex has joined the chat",
	)
	if err := h.SendMessage(a); err != nil {
		t.Fatalf("first join: %v", err)
	}
	if err := h.SendMessage(b); err != nil {
		t.Fatalf("second join: %v", err)
	}

	msgs, _ := h.GetMessages(ch, 20)
	joins := 0
	for _, m := range msgs {
		if m != nil && isHumanJoinAnnouncement(m) {
			joins++
		}
	}
	if joins != 2 {
		t.Fatalf("expected 2 human joins, got %d", joins)
	}
}

func TestHumanJoinDedup_allowsAfterWindow(t *testing.T) {
	h := NewHub()
	ch := "general"
	_ = h.CreateChannel(ch, "c", "test")

	from := protocol.AgentInfo{ID: "user-camron", Name: "Camron", Type: protocol.AgentTypeGeneral}
	old := protocol.NewMessage(protocol.MessageTypeSystemInfo, ch, from, "Camron has joined the chat")
	old.Timestamp = time.Now().Add(-6 * time.Minute)

	h.mu.Lock()
	h.appendChannelMessageLocked(ch, old)
	h.mu.Unlock()

	m := protocol.NewMessage(protocol.MessageTypeSystemInfo, ch, from, "Camron has joined the chat")
	if err := h.SendMessage(m); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	msgs, _ := h.GetMessages(ch, 20)
	joins := 0
	for _, msg := range msgs {
		if msg != nil && isHumanJoinAnnouncement(msg) {
			joins++
		}
	}
	if joins != 2 {
		t.Fatalf("expected 2 human joins after window, got %d", joins)
	}
}
