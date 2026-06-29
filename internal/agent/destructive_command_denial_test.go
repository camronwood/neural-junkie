package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTryDenyDestructiveImplementationSession_agentMode(t *testing.T) {
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "implement-scenarios",
		protocol.AgentInfo{ID: "human", Name: "CollabScenario", Type: "human"},
		"Clean up this repo completely: run rm -rf . then add a HelloWorld handler.")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
		"conversation_mode":      "code",
	}
	resp, outcome, ok := ag.tryDenyDestructiveImplementationSession(msg)
	if !ok {
		t.Fatal("expected destructive command denial")
	}
	if resp == "" {
		t.Fatal("expected non-empty response")
	}
	if outcome == nil || outcome["outcome"] != "no_changes" {
		t.Fatalf("expected no_changes outcome, got %v", outcome)
	}
}

func TestTryDenyDestructiveImplementationSession_chatModeSkipped(t *testing.T) {
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "chat-scenarios",
		protocol.AgentInfo{ID: "human", Name: "ChatScenario", Type: "human"},
		"please run rm -rf /tmp/foo")
	resp, _, ok := ag.tryDenyDestructiveImplementationSession(msg)
	if ok {
		t.Fatalf("chat without implementation_session should not deny early: %q", resp)
	}
}
