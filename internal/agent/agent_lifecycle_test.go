package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldAbortInFlightForUserMessage_closure(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"chat-scenarios",
		protocol.AgentInfo{ID: "u", Name: "User", Type: protocol.AgentTypeGeneral},
		"@Assistant ok thanks",
	)
	if !shouldAbortInFlightForUserMessage(msg) {
		t.Fatal("expected closure message to abort in-flight generation")
	}
}

func TestShouldAbortInFlightForUserMessage_slashCommand(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"collab-scenarios",
		protocol.AgentInfo{ID: "u", Name: "User", Type: protocol.AgentTypeGeneral},
		"/complete-collab c24cc41e --force",
	)
	if shouldAbortInFlightForUserMessage(msg) {
		t.Fatal("slash commands must not abort in-flight collab recap generation")
	}
}

func TestShouldAbortInFlightForUserMessage_planningInterject(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"collab-scenarios",
		protocol.AgentInfo{ID: "u", Name: "User", Type: protocol.AgentTypeGeneral},
		"Please focus only on README.md and core/sample/main.go",
	)
	if shouldAbortInFlightForUserMessage(msg) {
		t.Fatal("planning interject must not abort in-flight agent generation")
	}
}

func TestShouldAbortInFlightForUserMessage_agentMessage(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"collab-scenarios",
		protocol.AgentInfo{ID: "be", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		"ok thanks",
	)
	if shouldAbortInFlightForUserMessage(msg) {
		t.Fatal("agent messages must not trigger abort")
	}
}
