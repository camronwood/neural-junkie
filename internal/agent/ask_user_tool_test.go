package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldOfferAskUserTool_codebaseInjected(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Content: "@codebase What does ComputeObscureWidget return?",
		Metadata: map[string]interface{}{
			"injected_codebase_count": 2,
		},
	}
	if shouldOfferAskUserTool(nil, msg) {
		t.Fatal("expected ask_user suppressed for @codebase with injected chunks")
	}
}

func TestShouldOfferAskUserTool_themeWorkspaceGuidance(t *testing.T) {
	t.Parallel()
	ag := &Agent{WorkspacePath: "/tmp/minimal-repo"}
	msg := &protocol.Message{Content: "I want to add theme support to this app"}
	if shouldOfferAskUserTool(ag, msg) {
		t.Fatal("expected ask_user suppressed for workspace theme guidance")
	}
}

func TestShouldOfferAskUserTool_generalQuestion(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "what is your favorite color?"}
	if !shouldOfferAskUserTool(nil, msg) {
		t.Fatal("expected ask_user available for general question")
	}
}

func TestIsAgentChatReply_userQuestion(t *testing.T) {
	t.Parallel()
	msg := protocol.NewMessage(
		protocol.MessageTypeUserQuestion,
		"general",
		protocol.AgentInfo{ID: "be", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		"Which theme approach?",
	)
	if !isAgentChatReply(msg) {
		t.Fatal("expected user_question to count as agent chat engagement")
	}
}
