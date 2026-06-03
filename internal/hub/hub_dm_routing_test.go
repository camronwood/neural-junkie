package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNormalizeDMMentionRouting_autoRoutesPartner(t *testing.T) {
	h := NewHub()
	agent := protocol.AgentInfo{ID: "assistant-1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	h.RegisterAgent(&agent)
	ch := h.CreateChannelWithType("dm-camron-assistant", "Direct message with Assistant", "", protocol.ChannelTypeDM, "camron")
	_ = h.JoinChannel(agent.ID, ch.Name)

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, ch.Name,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"What is my name?")

	h.normalizeDMMentionRouting(msg, nil, nil)
	if len(msg.Mentions) != 1 || msg.Mentions[0] != agent.ID {
		t.Fatalf("expected auto-route to %q, got %v", agent.ID, msg.Mentions)
	}
}

func TestSendMessage_DMPlainTextAutoRoutes(t *testing.T) {
	h := NewHub()
	agent := protocol.AgentInfo{ID: "assistant-1", Name: "Assistant", Type: protocol.AgentTypeAssistant}
	h.RegisterAgent(&agent)
	ch := h.CreateChannelWithType("dm-camron-assistant", "Direct message with Assistant", "", protocol.ChannelTypeDM, "camron")
	_ = h.JoinChannel(agent.ID, ch.Name)

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, ch.Name,
		protocol.AgentInfo{ID: "human-1", Name: "camron", Type: "human"},
		"Hello there")
	if err := h.SendMessage(msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Mentions) != 1 || msg.Mentions[0] != agent.ID {
		t.Fatalf("expected mention auto-route, got %v", msg.Mentions)
	}
}
