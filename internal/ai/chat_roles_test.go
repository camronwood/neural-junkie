package ai

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestChatRoleForHistory(t *testing.T) {
	tests := []struct {
		name string
		from protocol.AgentInfo
		want string
	}{
		{"human", protocol.AgentInfo{Type: "human", Name: "Camron"}, "user"},
		{"general user", protocol.AgentInfo{Type: protocol.AgentTypeGeneral, Name: "Camron"}, "user"},
		{"system general", protocol.AgentInfo{Type: protocol.AgentTypeGeneral, Name: "System"}, "assistant"},
		{"backend agent", protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "GoExpert"}, "assistant"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.Message{From: tc.from, Content: "hello"}
			if got := ChatRoleForHistory(msg); got != tc.want {
				t.Fatalf("ChatRoleForHistory() = %q, want %q", got, tc.want)
			}
		})
	}
}
