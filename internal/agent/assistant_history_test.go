package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestFilterAssistantHistoryDedupesAndSkipsJoins(t *testing.T) {
	current := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm",
		protocol.AgentInfo{ID: "h", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"current?",
	)
	current.ID = "cur-id"

	history := []*protocol.Message{
		{
			ID:      "join-1",
			Type:    protocol.MessageTypeAgentJoin,
			Channel: "dm",
			From:    protocol.AgentInfo{ID: "a", Name: "Assistant", Type: protocol.AgentTypeAssistant},
			Content: "joined",
		},
		{
			ID:      "dup",
			Type:    protocol.MessageTypeQuestion,
			Channel: "dm",
			From:    protocol.AgentInfo{ID: "h", Name: "Camron", Type: protocol.AgentTypeGeneral},
			Content: "older",
		},
		{
			ID:      "dup",
			Type:    protocol.MessageTypeQuestion,
			Channel: "dm",
			From:    protocol.AgentInfo{ID: "h", Name: "Camron", Type: protocol.AgentTypeGeneral},
			Content: "older",
		},
		current,
	}

	out := filterAssistantHistory(history, current)
	if len(out) != 1 {
		t.Fatalf("expected 1 history row (older dup only), got %d", len(out))
	}
	if out[0].ID != "dup" {
		t.Fatalf("unexpected id %q", out[0].ID)
	}
}

func TestUserAsksAboutPromptContext(t *testing.T) {
	if !userAsksAboutPromptContext("what information do you get when I send you a prompt?") {
		t.Fatal("expected meta context detection")
	}
	if userAsksAboutPromptContext("remind me in 5 minutes") {
		t.Fatal("did not expect meta context detection")
	}
}
