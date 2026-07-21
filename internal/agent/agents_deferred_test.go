package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type pendingQuestionHub struct {
	shouldRespondTestHub
	pending bool
}

func (h pendingQuestionHub) HasPendingUserQuestion(string) bool { return h.pending }

func TestAgentsDeferred_pendingUserQuestion(t *testing.T) {
	hub := pendingQuestionHub{pending: true}
	if !agentsDeferred(hub, "general") {
		t.Fatal("expected defer while ask_user pending")
	}
	hub.pending = false
	if agentsDeferred(hub, "general") {
		t.Fatal("expected no defer without pending or hold")
	}
}

func TestShouldRespond_userQuestionNever(t *testing.T) {
	hub := shouldRespondTestHub{}
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(protocol.MessageTypeUserQuestion, "general", protocol.AgentInfo{ID: "a", Name: "A"}, "question")
	if ag.shouldRespond(msg) {
		t.Fatal("agents must not auto-reply to user_question cards")
	}
}
