package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
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

func TestStampedRunMisrouteForNeuralCanvasBecomesArtifact(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeCode,
		protocol.IdeMetaEditorMode: "agent",
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionRun, Action: intent.ActionRun,
		Retrieval: []intent.RetrievalTarget{
			intent.RetrievalPriorReference, intent.RetrievalCodebase, intent.RetrievalCodeGraph,
		},
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !UserRequestsArtifact(msg.Content) {
		t.Fatal("fixture must phrase-match Neural Canvas")
	}
	if !artifactToolsEnabledForMessage(msg) {
		t.Fatal("run misroute must still enable create_artifact for Neural Canvas asks")
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact || goal.ImplementationSession {
		t.Fatalf("goal=%+v, want artifact without implementation session", goal)
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); !ok || decision.Action != intent.ActionArtifact {
		t.Fatalf("restamped decision=%v ok=%v, want ActionArtifact", decision.Action, ok)
	}
	if userRequestsImplementationForMessage(a, msg) {
		t.Fatal("Neural Canvas ask must not force FILE_CHANGE implementation")
	}
	if len(goal.ExpectedEvidence) != 1 || goal.ExpectedEvidence[0] != EvidenceArtifactCreated {
		t.Fatalf("evidence=%v, want artifact_created", goal.ExpectedEvidence)
	}
	if shouldRunImplementationSession(a, msg) {
		t.Fatal("Neural Canvas run misroute must not enter implementation session")
	}
}

func TestArtifactGoalWithoutEvidenceRewritesFileChangeProse(t *testing.T) {
	goal := TurnGoal{
		Action: ActionArtifact, ExpectedEvidence: []EvidenceKind{EvidenceArtifactCreated},
	}
	ledger := &ActionEvidenceLedger{}
	resp := "Grounding: I loaded 8 file(s). I'll modify the existing codebase to meet the requirements. Here are the modifications:\n### scripts/test_mermaid_layouts.mjs"
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, resp, nil)
	if !shouldRewriteAsSafeFailureForGoal(goal, ledger, issues, resp) {
		t.Fatalf("expected rewrite for missing artifact evidence; issues=%v", issues)
	}
	if got := safeActionFailure(goal, ledger); got != "I couldn't create the requested Neural Canvas artifact in this turn." {
		t.Fatalf("safe failure=%q", got)
	}
}

func TestNeuralCanvasPromptOmitsFileChangeImplementation(t *testing.T) {
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), hub)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionRun, Action: intent.ActionRun,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	_ = deriveTurnGoal(a, msg, IntentTask)
	prompt := a.buildPrompt(msg, IntentTask)
	if strings.Contains(prompt, "IMPLEMENTATION REQUEST") {
		t.Fatal("Neural Canvas prompt must not force FILE_CHANGE implementation")
	}
	if strings.Contains(prompt, "You MUST emit one or more [FILE_CHANGE]") ||
		strings.Contains(prompt, "include this machine-readable block exactly") {
		t.Fatal("Neural Canvas prompt must not document FILE_CHANGE proposal protocol")
	}
	if !strings.Contains(prompt, "create_artifact") {
		t.Fatal("Neural Canvas prompt must mention create_artifact")
	}
}

func TestStampedInspectMisrouteForNeuralCanvasBecomesArtifact(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Show me a Neural Canvas chart of the coverage gaps",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionInspect, Action: intent.ActionInspect,
		Mutation: intent.MutationNone, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if !artifactToolsEnabledForMessage(msg) {
		t.Fatal("inspect misroute must enable create_artifact")
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact", goal.Action)
	}
}

func TestStampedImageMisrouteForNeuralCanvasBecomesArtifact(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionImage, Action: intent.ActionImage,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	if UserRequestsGeneratedImage(msg.Content) {
		t.Fatal("Neural Canvas ask must not match image heuristic")
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact {
		t.Fatalf("action=%s, want artifact", goal.Action)
	}
	if a.imageGenerationToolsEnabledForMessage(msg) {
		t.Fatal("generate_image must stay disabled for Neural Canvas")
	}
	if got, ok := a.tryHubImageGenerationShortcut(context.Background(), msg); ok || got != "" {
		t.Fatalf("image shortcut must not fire: ok=%v got=%q", ok, got)
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
