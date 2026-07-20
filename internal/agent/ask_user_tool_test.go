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

func TestShouldOfferAskUserTool_shortConfusedFollowUp(t *testing.T) {
	t.Parallel()
	cases := []string{"What?", "huh?", "??", "@BackendEngineer What?", "what do you mean?"}
	for _, content := range cases {
		msg := &protocol.Message{
			Content: content,
			Metadata: map[string]interface{}{
				MetadataConversationMode: ConversationModeChat,
			},
		}
		if shouldOfferAskUserTool(nil, msg) {
			t.Fatalf("expected ask_user suppressed for confused follow-up %q", content)
		}
		if !shortConfusedFollowUp(content) {
			t.Fatalf("expected shortConfusedFollowUp(%q)", content)
		}
	}
}

func TestShouldOfferAskUserTool_echoFollowupClass(t *testing.T) {
	t.Parallel()
	// Layer A companion for scenarios/chat/dm-backend-echo-followup.json:
	// after a theme turn, "What?" must not expose ask_user (preference menus / timeouts).
	ag := &Agent{WorkspacePath: "/tmp/minimal-repo"}
	msg := &protocol.Message{
		Content: "What?",
		Metadata: map[string]interface{}{
			MetadataConversationMode: ConversationModeChat,
		},
	}
	if shouldOfferAskUserTool(ag, msg) {
		t.Fatal("expected ask_user suppressed for dm-backend-echo-followup class")
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
