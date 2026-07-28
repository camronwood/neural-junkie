package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestTryBootFixImplementerRedirect_architectureDM documents the stamp-first replacement for
// the old "won't boot" phrase detection: the classifier now tags boot/startup runtime failures
// via reason codes (see messageStampedBootFailure) rather than natural-language phrase matching.
func TestTryBootFixImplementerRedirect_architectureDM(t *testing.T) {
	const dm = "dm-camron-softwarearchitect"
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"This app is not booting up, can you fix the Makefile?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
		"workspace_context": map[string]interface{}{
			"workspace_path": t.TempDir(),
		},
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	resp, outcome, ok := ag.tryBootFixImplementerRedirect(msg)
	if !ok {
		t.Fatal("expected redirect for architecture DM boot-fix")
	}
	if resp == "" || !strings.Contains(resp, "FrontendEngineer") || !strings.Contains(resp, "SoftwareArchitect") {
		t.Fatalf("unexpected redirect: %q", resp)
	}
	if outcome == nil || outcome["outcome"] != "wrong_route" {
		t.Fatalf("expected wrong_route outcome, got %v", outcome)
	}
}

func TestTryBootFixImplementerRedirect_frontendDM(t *testing.T) {
	const dm = "dm-camron-frontendengineer"
	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"the app is not booting up")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
	}
	resp, _, ok := ag.tryBootFixImplementerRedirect(msg)
	if ok {
		t.Fatalf("frontend DM should not redirect: %q", resp)
	}
}

func TestTryBootFixImplementerRedirect_assistantNever(t *testing.T) {
	const dm = "dm-camron-assistant"
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"why did you query github?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
		MetadataConversationMode: ConversationModeCode,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := ag.tryBootFixImplementerRedirect(msg); ok {
		t.Fatal("Assistant DM must never emit wrong_route redirect")
	}
}

func TestTryBootFixImplementerRedirect_agentModeAloneInsufficient(t *testing.T) {
	const dm = "dm-camron-softwarearchitect"
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"This app is not booting up")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		MetadataConversationMode: ConversationModeChat,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := ag.tryBootFixImplementerRedirect(msg); ok {
		t.Fatal("sticky editor_mode=agent without code/impl must not redirect")
	}
}

func TestTryBootFixImplementerRedirect_workspaceVisibilityNever(t *testing.T) {
	const dm = "dm-camron-backendengineer"
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"can you see my workspace I have open?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"editor_mode":            "agent",
		MetadataConversationMode: ConversationModeCode,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := ag.tryBootFixImplementerRedirect(msg); ok {
		t.Fatal("workspace visibility asks must not emit wrong_route redirect")
	}
}

func TestTryBootFixImplementerRedirect_codeModeAloneInsufficient(t *testing.T) {
	const dm = "dm-camron-backendengineer"
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{dmChannel: dm})
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, dm,
		protocol.AgentInfo{ID: "human-user", Name: "camronwood", Type: "human"},
		"@codebase What does ComputeObscureWidget return?")
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		MetadataConversationMode: ConversationModeCode,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := ag.tryBootFixImplementerRedirect(msg); ok {
		t.Fatal("conversation_mode=code without implementation_session must not redirect")
	}
}
