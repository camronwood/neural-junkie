package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestHubDMMultiTurnHistoryOrdering verifies DM channels preserve chronological
// user/agent alternation across multiple turns (conversation flow contract).
func TestHubDMMultiTurnHistoryOrdering(t *testing.T) {
	h := newTestHub(t)
	agent := protocol.AgentInfo{
		ID:   "backend-1",
		Name: "BackendEngineer",
		Type: protocol.AgentTypeBackend,
	}
	if err := h.RegisterAgent(&agent); err != nil {
		t.Fatal(err)
	}
	ch := h.CreateChannelWithType(
		"dm-chatscenario-backendengineer",
		"Direct message with BackendEngineer",
		"",
		protocol.ChannelTypeDM,
		"ChatScenario",
	)
	if err := h.JoinChannel(agent.ID, ch.Name); err != nil {
		t.Fatal(err)
	}

	user := protocol.AgentInfo{ID: "human-1", Name: "ChatScenario", Type: "human"}
	turns := []struct {
		userText  string
		agentText string
	}{
		{"I want to add theme support", "Theme support usually needs a config and CSS variables."},
		{"can you see my workspace?", "Yes — I have workspace context for this channel."},
		{"go deeper on the approach", "Start with a theme context provider and toggle in settings."},
	}

	for i, turn := range turns {
		userMsg := protocol.NewMessage(protocol.MessageTypeQuestion, ch.Name, user, turn.userText)
		if err := h.SendMessage(userMsg); err != nil {
			t.Fatalf("turn %d user send: %v", i+1, err)
		}
		agentMsg := protocol.NewMessage(protocol.MessageTypeChat, ch.Name, agent, turn.agentText)
		if err := h.SendMessage(agentMsg); err != nil {
			t.Fatalf("turn %d agent send: %v", i+1, err)
		}
	}

	msgs, err := h.GetMessages(ch.Name, 50)
	if err != nil {
		t.Fatal(err)
	}
	var conv []*protocol.Message
	for _, m := range msgs {
		if m == nil {
			continue
		}
		switch m.Type {
		case protocol.MessageTypeQuestion, protocol.MessageTypeChat, protocol.MessageTypeAnswer:
			conv = append(conv, m)
		}
	}
	if len(conv) != len(turns)*2 {
		t.Fatalf("expected %d conversation messages, got %d (total=%d)", len(turns)*2, len(conv), len(msgs))
	}
	for i, m := range conv {
		wantUser := i%2 == 0
		gotUser := protocol.IsUserLikeSender(m.From)
		if wantUser != gotUser {
			t.Fatalf("conv[%d]: expected user=%v got user=%v (from=%s type=%s)", i, wantUser, gotUser, m.From.Name, m.Type)
		}
	}
	if len(msgs) > 100 {
		t.Fatalf("unexpected history overflow: %d messages", len(msgs))
	}
}
