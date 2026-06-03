package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDMHumanMessageShouldRespond_plainText(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-user-bot",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"What is my name?")
	if !DMHumanMessageShouldRespond(msg, "bot-id") {
		t.Fatal("expected DM partner to respond without @mention")
	}
}

func TestDMHumanMessageShouldRespond_spuriousMention(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-user-bot",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"Email me at user@example.com")
	msg.Mentions = []string{"__INVALID__"}
	if !DMHumanMessageShouldRespond(msg, "bot-id") {
		t.Fatal("expected DM partner to respond when mentions are invalid/spurious")
	}
}

func TestDMHumanMessageShouldRespond_otherAgentMentioned(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm-user-bot",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"@BackendEngineer help")
	msg.Mentions = []string{"backend-id"}
	if DMHumanMessageShouldRespond(msg, "bot-id") {
		t.Fatal("expected DM partner to stay silent when another agent is @mentioned")
	}
}

func TestShouldRespond_DMWithoutMention(t *testing.T) {
	const dm = "dm-alice-assistant"
	hubStub := shouldRespondTestHub{dmChannel: dm}
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hubStub)

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "alice", Type: "human"},
		"What is my name?")
	if !ag.shouldRespond(msg) {
		t.Fatal("expected assistant to respond in DM without @mention")
	}
}
