package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type mentionAgentsHub struct {
	shouldRespondTestHub
	agents []protocol.AgentInfo
}

func (h mentionAgentsHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) {
	return h.agents, nil
}

func TestShouldRespond_ContentMentionWithoutStoredMentions(t *testing.T) {
	assistantID := "assistant-1"
	hubStub := mentionAgentsHub{
		shouldRespondTestHub: shouldRespondTestHub{},
		agents: []protocol.AgentInfo{
			{ID: assistantID, Name: "Assistant", Type: protocol.AgentTypeAssistant},
			{ID: "backend-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		},
	}
	mockAI := ai.NewMockProvider()

	assistant := NewAgent(protocol.AgentTypeAssistant, "Assistant", []string{"help"}, mockAI, hubStub)
	assistant.Info.ID = assistantID

	backend := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"api"}, mockAI, hubStub)
	backend.Info.ID = "backend-1"

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human-user", Name: "camron", Type: "human"},
		"@Assistant what time is it?",
	)
	msg.Mentions = nil

	if backend.shouldRespond(msg) {
		t.Fatal("BackendEngineer must not respond when only Assistant is @mentioned in content")
	}
	if !assistant.shouldRespond(msg) {
		t.Fatal("Assistant should respond when @mentioned in content without stored Mentions")
	}
}

func TestMentionTargetAlreadyAnswered(t *testing.T) {
	user := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "u1", Name: "camron", Type: "human"},
		"@Assistant hello",
	)
	user.Mentions = []string{"assistant-1"}

	reply := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "assistant-1", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		"Hi there",
	)

	history := []*protocol.Message{user, reply}
	if !mentionTargetAlreadyAnswered(history, 0, user) {
		t.Fatal("expected mention target to be marked as already answered")
	}
}
