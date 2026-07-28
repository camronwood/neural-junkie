package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestUserDeclinedOrDeferred(t *testing.T) {
	t.Parallel()
	for _, answer := range []string{
		"nothing right now",
		"Nothing right now.",
		"never mind",
		"not now",
		"stop",
		"no thanks",
		"maybe later",
		"skip",
	} {
		if !userDeclinedOrDeferred(answer) {
			t.Fatalf("expected decline for %q", answer)
		}
	}
	for _, answer := range []string{
		"Desktop",
		"use the payments namespace",
		"option 2",
		"yes continue",
	} {
		if userDeclinedOrDeferred(answer) {
			t.Fatalf("did not expect decline for %q", answer)
		}
	}
}

type declineAskUserHub struct {
	shouldRespondTestHub
	answer string
}

func (h declineAskUserHub) AskUserQuestion(agentID, agentName, channel, question string, options []string) (string, error) {
	return h.answer, nil
}

func TestExecuteAskUserTool_declinedStops(t *testing.T) {
	hub := declineAskUserHub{answer: "nothing right now"}
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, ai.NewMockProvider(), hub)
	ctx := withAskUserTurnState(context.Background())
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{ID: "u", Name: "User"}, "pick a topic")
	out, err := ag.executeAskUserTool(ctx, msg, []byte(`{"question":"What should we work on?"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "declined or deferred") || !strings.Contains(out, "Stop") {
		t.Fatalf("expected stop-on-decline message, got %q", out)
	}
	if strings.Contains(out, "Continue the original task") {
		t.Fatal("decline must not continue the original task")
	}
}
