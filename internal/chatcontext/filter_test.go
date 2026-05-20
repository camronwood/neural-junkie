package chatcontext

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestOmitFromLLMHistory_systemInfo(t *testing.T) {
	m := protocol.NewMessage(protocol.MessageTypeSystemInfo, "c", protocol.AgentInfo{ID: "system", Name: "System"}, "joined")
	if !OmitFromLLMHistory(m) {
		t.Fatal("expected system_info omitted")
	}
}

func TestOmitFromLLMHistory_userChatKept(t *testing.T) {
	m := protocol.NewMessage(protocol.MessageTypeChat, "c", protocol.AgentInfo{ID: "u", Name: "User"}, "hello")
	if OmitFromLLMHistory(m) {
		t.Fatal("expected user chat kept")
	}
}

func TestFilterForLLM_excludesNoise(t *testing.T) {
	msgs := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "c", protocol.AgentInfo{ID: "u", Name: "User"}, "hi"),
		protocol.NewMessage(protocol.MessageTypeSystemInfo, "c", protocol.AgentInfo{ID: "system", Name: "System"}, "noise"),
	}
	out := FilterForLLM(msgs, "", 10)
	if len(out) != 1 {
		t.Fatalf("got %d messages, want 1", len(out))
	}
}
