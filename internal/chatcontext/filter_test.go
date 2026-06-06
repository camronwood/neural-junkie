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

func TestOmitFromLLMHistory_systemCollaborationDiscussion(t *testing.T) {
	m := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"c",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration Started",
	)
	if !OmitFromLLMHistory(m) {
		t.Fatal("expected system-authored collaboration prompt omitted")
	}
}

func TestOmitFromLLMHistory_userChatKept(t *testing.T) {
	m := protocol.NewMessage(protocol.MessageTypeChat, "c", protocol.AgentInfo{ID: "u", Name: "User"}, "hello")
	if OmitFromLLMHistory(m) {
		t.Fatal("expected user chat kept")
	}
}

func TestOmitFromLLMHistory_commandOutputWithContentKept(t *testing.T) {
	m := protocol.NewMessage(
		protocol.MessageTypeCommandOutput,
		"c",
		protocol.AgentInfo{ID: "terminal", Name: "Terminal", Type: protocol.AgentTypeGeneral},
		"Command: `ls`\nExit code: 0",
	)
	if OmitFromLLMHistory(m) {
		t.Fatal("expected command_output with content kept for LLM history")
	}
}

func TestOmitFromLLMHistory_commandOutputEmptyOmitted(t *testing.T) {
	m := protocol.NewMessage(
		protocol.MessageTypeCommandOutput,
		"c",
		protocol.AgentInfo{ID: "terminal", Name: "Terminal", Type: protocol.AgentTypeGeneral},
		"",
	)
	if !OmitFromLLMHistory(m) {
		t.Fatal("expected empty command_output omitted")
	}
}

func TestOmitFromLLMHistory_fileChangeApprovalKept(t *testing.T) {
	m := protocol.NewMessage(
		protocol.MessageTypeChat,
		"c",
		protocol.AgentInfo{ID: "human-user", Name: "User", Type: "human"},
		"Approved and applied your edit change to `src/App.tsx`.",
	)
	m.Metadata = map[string]interface{}{protocol.MetaFileChangeApproved: true}
	if OmitFromLLMHistory(m) {
		t.Fatal("expected file-change approval chat kept in LLM history")
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
