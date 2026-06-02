package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsImplementationForMessage_continuation(t *testing.T) {
	t.Parallel()
	a := &Agent{
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"general": {
					{
						ID: "u1",
						From: protocol.AgentInfo{
							Name: "camron",
							Type: protocol.AgentTypeGeneral,
						},
						Content: "add light and dark themes under settings",
					},
				},
			},
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "general",
		From: protocol.AgentInfo{
			Name: "camron",
			Type: protocol.AgentTypeGeneral,
		},
		Content: "yes please go ahead",
	}
	if !userRequestsImplementationForMessage(a, msg) {
		t.Fatal("expected continuation after implementation ask")
	}
}

func TestClassifyTurnIntent_continuationIsTask(t *testing.T) {
	t.Parallel()
	history := []*protocol.Message{
		{
			ID: "u1",
			From: protocol.AgentInfo{
				Name: "camron",
				Type: protocol.AgentTypeGeneral,
			},
			Content: "implement UI themes in settings",
		},
	}
	msg := &protocol.Message{
		ID:      "u2",
		Channel: "general",
		From: protocol.AgentInfo{
			Name: "camron",
			Type: protocol.AgentTypeGeneral,
		},
		Content: "ok please do it now",
		Metadata: map[string]interface{}{
			"context_scope":      "outline",
			"conversation_mode":  "code",
		},
	}
	got := ClassifyTurnIntentPublic(msg, protocol.ChannelTypePublic, "agent-1", history)
	if got != IntentTask {
		t.Fatalf("intent = %s, want task", got.String())
	}
}
