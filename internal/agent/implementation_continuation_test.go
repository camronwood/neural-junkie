package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserRequestsImplementationForMessage_continuation(t *testing.T) {
	t.Parallel()
	a := &Agent{Context: &ConversationContext{History: map[string][]*protocol.Message{}}}
	msg := &protocol.Message{
		ID: "u2", Channel: "general",
		From: protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "yes please go ahead",
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionContinuation,
		RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
		Mutation: intent.MutationWorkspace, Confidence: 1, Source: intent.SourceLocalModel,
		ContinuationTarget: "goal-1",
	}); err != nil {
		t.Fatal(err)
	}
	if !userRequestsImplementationForMessage(a, msg) {
		t.Fatal("stamped ActionContinue must request implementation")
	}
}

func TestClassifyTurnIntent_continuationIsTask(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		ID: "u2", Channel: "general",
		From: protocol.AgentInfo{Name: "camron", Type: protocol.AgentTypeGeneral},
		Content: "ok please do it now",
		Metadata: map[string]interface{}{
			"context_scope": "outline", "conversation_mode": "code",
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionContinuation,
		RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
		Mutation: intent.MutationWorkspace, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	got := ClassifyTurnIntentPublic(msg, protocol.ChannelTypePublic, "agent-1", nil)
	if got != IntentTask {
		t.Fatalf("intent = %s, want task", got.String())
	}
}
