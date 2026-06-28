package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResolveImagePostAgent_DMPrefersPartner(t *testing.T) {
	h := NewHub()
	agent := protocol.AgentInfo{
		ID:                      "assistant-1",
		Name:                    "Assistant",
		Type:                    protocol.AgentTypeAssistant,
		SupportsImageGeneration: true,
	}
	h.RegisterAgent(&agent)
	ch := h.CreateChannelWithType("dm-camron-assistant", "Direct message with Assistant", "", protocol.ChannelTypeDM, "camron")
	_ = h.JoinChannel(agent.ID, ch.Name)

	got := h.resolveImagePostAgent(ch.Name)
	if got.ID != agent.ID {
		t.Fatalf("expected DM partner %q, got %+v", agent.ID, got)
	}
}

func TestResolveImagePostAgent_FallsBackToAssistant(t *testing.T) {
	h := NewHub()
	agent := protocol.AgentInfo{
		ID:                      "assistant-1",
		Name:                    "Assistant",
		Type:                    protocol.AgentTypeAssistant,
		SupportsImageGeneration: true,
	}
	h.RegisterAgent(&agent)
	ch := h.CreateChannel("general", "General", "")

	got := h.resolveImagePostAgent(ch.Name)
	if got.ID != agent.ID {
		t.Fatalf("expected Assistant %q, got %+v", agent.ID, got)
	}
}
