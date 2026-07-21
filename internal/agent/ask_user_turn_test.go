package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAskUserTurnState_oncePerTurn(t *testing.T) {
	ctx := withAskUserTurnState(context.Background())
	st := askUserTurnStateFromContext(ctx)
	if st == nil {
		t.Fatal("expected turn state")
	}
	st.count = 1
	st.answered = []string{"- platform → Desktop"}

	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "rustgame", protocol.AgentInfo{ID: "u", Name: "User"}, "cargo init failed")
	out, err := ag.executeAskUserTool(ctx, msg, []byte(`{"question":"What is the target platform?"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "already used") || !strings.Contains(out, "Proceed") {
		t.Fatalf("expected once-per-turn refusal, got %q", out)
	}
}

func TestAskUserTurnState_persistsAcrossImplementationRounds(t *testing.T) {
	sessionCtx := withImplementationSessionTurnState(context.Background())
	sessionState := askUserTurnStateFromContext(sessionCtx)
	if sessionState == nil {
		t.Fatal("expected implementation-session ask_user state")
	}
	sessionState.count = 1
	sessionState.answered = []string{"- genre → RTS"}

	for round := 0; round < 4; round++ {
		roundCtx := withImplementationSessionRound(sessionCtx, round)
		toolCtx := withAskUserTurnState(roundCtx)
		if got := askUserTurnStateFromContext(toolCtx); got != sessionState {
			t.Fatalf("round %d reset ask_user state", round)
		}
	}

	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "rustgame", protocol.AgentInfo{ID: "u", Name: "User"}, "keep going")
	out, err := ag.executeAskUserTool(sessionCtx, msg, []byte(`{"question":"What genre?"}`))
	if err != nil {
		t.Fatalf("executeAskUserTool: %v", err)
	}
	if !strings.Contains(out, "already used") || !strings.Contains(out, "RTS") {
		t.Fatalf("expected session-wide refusal with prior answer, got %q", out)
	}
}

func TestAbortChannel_skipsWhenPinned(t *testing.T) {
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, ai.NewMockProvider(), hub)
	cancelled := false
	ag.registerGenCancel("rustgame", func() { cancelled = true })
	ag.pinGenerationForUserWait("rustgame")
	ag.AbortChannel("rustgame")
	if cancelled {
		t.Fatal("pinned generation must not be aborted")
	}
	ag.unpinGenerationForUserWait("rustgame")
	ag.AbortChannel("rustgame")
	if !cancelled {
		t.Fatal("expected abort after unpin")
	}
}
