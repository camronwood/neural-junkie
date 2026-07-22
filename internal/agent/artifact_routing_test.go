package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNeuralCanvasRequestOverridesCodeImplementationMode(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	a.Info.ID = "frontend-1"
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"code-channel",
		protocol.AgentInfo{ID: "user-1", Name: "Camron"},
		"Can you show me a Neural Canvas of the current test coverage in the workspace?",
	)
	msg.Metadata[MetadataConversationMode] = ConversationModeCode
	msg.Metadata[protocol.IdeMetaEditorMode] = "agent"
	msg.Metadata[protocol.IdeMetaImplementationSession] = true

	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact || goal.ImplementationSession {
		t.Fatalf("goal=%+v, want artifact without implementation session", goal)
	}
	if !artifactToolsEnabledForMessage(msg) {
		t.Fatal("artifact tools disabled for explicit canvas request in code mode")
	}
}

func TestArtifactApprovalContinuesArtifactInsteadOfFileEdits(t *testing.T) {
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), hub)
	a.Info.ID = "frontend-1"
	user := protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"}
	request := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"code-channel",
		user,
		"Show me a Neural Canvas of test coverage and the highest risk gaps.",
	)
	promise := protocol.NewMessage(
		protocol.MessageTypeChat,
		"code-channel",
		a.Info,
		"I'll start by reviewing the workspace and then create the canvas.",
	)
	a.replaceChannelHistory("code-channel", []*protocol.Message{request, promise})

	approval := protocol.NewMessage(protocol.MessageTypeChat, "code-channel", user, "ok sounds good")
	approval.Metadata[MetadataConversationMode] = ConversationModeCode
	approval.Metadata[protocol.IdeMetaEditorMode] = "agent"
	approval.Metadata[protocol.IdeMetaImplementationSession] = true

	goal := deriveTurnGoal(a, approval, IntentTask)
	if goal.Action != ActionArtifact || goal.ImplementationSession {
		t.Fatalf("approval goal=%+v, want pending artifact continuation", goal)
	}
	if goal.ContinuationParent != request.ID {
		t.Fatalf("continuation parent=%q, want original artifact request %q", goal.ContinuationParent, request.ID)
	}
	goal = persistTurnConversationState(a, approval, goal)
	if goal.ID != request.ID || approval.Metadata["original_goal_id"] != request.ID {
		t.Fatalf("persisted continuation goal=%q metadata=%v, want %q", goal.ID, approval.Metadata, request.ID)
	}
	approval.Metadata["turn_action"] = string(goal.Action)
	if !artifactToolsEnabledForMessage(approval) {
		t.Fatal("artifact tools disabled for pending artifact continuation")
	}
}

func TestArtifactPrerequisiteQuestionStaysAdvisory(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	a.Info.ID = "frontend-1"
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"code-channel",
		protocol.AgentInfo{ID: "user-1", Name: "Camron"},
		"Do we need to fix those items before we can get our canvas?",
	)
	msg.Metadata[MetadataConversationMode] = ConversationModeCode
	msg.Metadata[protocol.IdeMetaEditorMode] = "agent"
	msg.Metadata[protocol.IdeMetaImplementationSession] = true

	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionAnswer || goal.ImplementationSession {
		t.Fatalf("advisory goal=%+v, want conversational answer", goal)
	}
}

func TestArtifactToolResultProvidesActionEvidence(t *testing.T) {
	ledger := &ActionEvidenceLedger{}
	ledger.recordToolEvent(ai.ToolStepEvent{
		Kind: "result", Name: createArtifactToolName, Preview: "artifact created",
	})
	if !ledger.Has(EvidenceArtifactCreated) {
		t.Fatalf("missing artifact evidence: %+v", ledger.Entries())
	}
	goal := TurnGoal{
		Action: ActionArtifact, ExpectedEvidence: []EvidenceKind{EvidenceArtifactCreated},
	}
	if issues := validateResponseAgainstEvidence(
		goal, ledger, &protocol.Message{}, "Created the Neural Canvas artifact.", nil,
	); len(issues) != 0 {
		t.Fatalf("supported artifact claim rejected: %v", issues)
	}
}

func TestNoChangeOutcomeDoesNotReportInspectedCandidates(t *testing.T) {
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	outcome := a.buildImplementationSessionOutcome(nil, &ImplementationSessionState{
		FilesChanged: []string{"desktop/src/App.tsx", "desktop/src/components/ChatWindow.tsx"},
	}, false)
	if files, ok := outcome["files_changed"].([]string); !ok || len(files) != 0 {
		t.Fatalf("files_changed=%v, want empty for no-change outcome", outcome["files_changed"])
	}
}
