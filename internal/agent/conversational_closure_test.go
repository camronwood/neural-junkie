package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClassifyConversationalClosure(t *testing.T) {
	cases := []struct {
		in   string
		want ClosureKind
	}{
		{"ok thanks", ClosureThanks},
		{"Thank you!", ClosureThanks},
		{"I know you said that already", ClosureAlreadyAnswered},
		{"you already told me", ClosureAlreadyAnswered},
		{"cool", ClosureBriefAck},
		{"how far to STL?", ClosureNone},
		{"thanks but can you also check distance?", ClosureNone},
		{"I know you said that already — what about traffic?", ClosureNone},
	}
	for _, tc := range cases {
		got := classifyConversationalClosure(tc.in)
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestTryConversationalClosureThanks(t *testing.T) {
	ag := &Agent{
		Info: protocol.AgentInfo{ID: "asst-1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		Context: &ConversationContext{
			History: map[string][]*protocol.Message{
				"dm": {
					{Type: protocol.MessageTypeQuestion, From: protocol.AgentInfo{ID: "u", Name: "Camron"}, Content: "distance?"},
					{Type: protocol.MessageTypeChat, From: protocol.AgentInfo{ID: "asst-1", Name: "Assistant"}, Content: "39 miles"},
				},
			},
		},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "Camron"}, "ok thanks")
	resp, ok := tryConversationalClosure(ag, msg)
	if !ok || resp == "" {
		t.Fatal("expected closure response for ok thanks")
	}
}

func TestTryConversationalClosureBriefAckNeedsPriorAnswer(t *testing.T) {
	ag := &Agent{
		Info: protocol.AgentInfo{ID: "asst-1", Name: "Assistant"},
		Context: &ConversationContext{History: map[string][]*protocol.Message{"dm": {}}},
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "Camron"}, "cool")
	if _, ok := tryConversationalClosure(ag, msg); ok {
		t.Fatal("cool without prior agent answer should not closure")
	}
}
