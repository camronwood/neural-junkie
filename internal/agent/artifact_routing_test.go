package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestNeuralCanvasRequestOverridesCodeImplementationMode documents the stamp-first
// replacement for the old Neural Canvas phrase override: a classifier that stamps
// ActionArtifact is trusted as-is, even when ambient IDE metadata says an implementation
// session is active — routing no longer re-derives that from message phrasing.
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
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 0.95, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionArtifact || goal.ImplementationSession {
		t.Fatalf("goal=%+v, want artifact without implementation session", goal)
	}
	if !artifactToolsEnabledForMessage(msg) {
		t.Fatal("artifact tools disabled for explicit canvas request in code mode")
	}
}

// TestArtifactApprovalContinuesArtifactInsteadOfFileEdits documents the stamp-first
// replacement for the old phrase-based "approval continues a pending artifact request"
// heuristic: the classifier now stamps the approval turn ActionArtifact with a
// ContinuationTarget pointing at the original request, rather than routing re-scanning
// channel history for Neural Canvas phrasing.
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
	if err := protocol.StampTurnDecision(approval, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		ContinuationTarget: request.ID,
		Mutation:           intent.MutationExternal, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}

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

// TestArtifactPrerequisiteQuestionStaysAdvisory documents the stamp-first replacement for
// the old isAdvisoryImplementationQuestion phrase veto: a classifier that stamps ActionAnswer
// with the "advisory_question" reason code stays conversational, even when ambient IDE
// metadata says an implementation session is active.
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
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 0.9, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"advisory_question"},
	}); err != nil {
		t.Fatal(err)
	}

	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionAnswer || goal.ImplementationSession {
		t.Fatalf("advisory goal=%+v, want conversational answer", goal)
	}
}

// TestStampedRunDecisionTrustedDespiteCanvasPhrasing documents the stamp-first
// replacement for the old Neural Canvas "run misroute" restamp: a classifier that stamps
// ActionRun is trusted as-is, even when the message text mentions "Neural Canvas" / "Mermaid
// diagram". If the classifier means a durable artifact, it must stamp ActionArtifact
// directly — routing no longer re-derives that from message phrasing.
func TestStampedRunDecisionTrustedDespiteCanvasPhrasing(t *testing.T) {
	a := NewAgent(protocol.AgentTypeAssistant, "Assistant", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-assistant",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"Create a Neural Canvas Mermaid diagram of this architecture",
	)
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode:  ConversationModeCode,
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
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionRun || goal.ImplementationSession {
		t.Fatalf("goal=%+v, want stamped run trusted despite canvas phrasing", goal)
	}
	if decision, ok := protocol.ExtractTurnDecision(msg); !ok || decision.Action != intent.ActionRun {
		t.Fatalf("decision=%v ok=%v, want ActionRun unchanged", decision.Action, ok)
	}
	if artifactToolsEnabledForMessage(msg) {
		t.Fatal("ActionRun must not enable artifact tools despite canvas phrasing")
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
		RequestedAction: intent.ActionArtifact, Action: intent.ActionArtifact,
		Mutation: intent.MutationExternal, Confidence: 1, Source: intent.SourceLocalModel,
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

// TestStampedInspectDecisionTrustedDespiteCanvasPhrasing documents the stamp-first
// replacement for the old Neural Canvas "inspect misroute" restamp: a classifier that
// stamps ActionInspect is trusted as-is, even with Neural Canvas phrasing.
func TestStampedInspectDecisionTrustedDespiteCanvasPhrasing(t *testing.T) {
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
	if artifactToolsEnabledForMessage(msg) {
		t.Fatal("ActionInspect must not enable artifact tools despite canvas phrasing")
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionInspect {
		t.Fatalf("action=%s, want inspect trusted as stamped", goal.Action)
	}
}

// TestStampedImageDecisionTrustedDespiteCanvasPhrasing documents the stamp-first
// replacement for the old Neural Canvas "image misroute" restamp: a classifier that
// stamps ActionImage is trusted as-is, even with Neural Canvas phrasing.
func TestStampedImageDecisionTrustedDespiteCanvasPhrasing(t *testing.T) {
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
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionImage {
		t.Fatalf("action=%s, want image trusted as stamped", goal.Action)
	}
	if messageSuppressesImageGeneration(msg) {
		t.Fatal("generate_image must stay enabled for a stamped ActionImage despite canvas phrasing")
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

func TestChatSurfaceKeepsArtifactSoftFailResponse(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "please summarize my last meeting notes in the chat")
	msg.Metadata = map[string]interface{}{
		"open_artifact": map[string]interface{}{
			"id": "art-1", "renderer_id": "nj.markdown", "title": "St. Louis to Sea Side",
		},
	}
	goal := TurnGoal{Action: ActionArtifact, ExpectedEvidence: []EvidenceKind{EvidenceArtifactCreated}}
	resp := "Yesterday's trip planning call covered the St. Louis to Seaside drive and the Friday arrival."
	issues := validateResponseAgainstEvidence(goal, &ActionEvidenceLedger{}, msg, resp, nil)
	if !shouldKeepChatSurfaceResponse(msg, goal, issues, resp) {
		t.Fatalf("expected keep chat-surface summary; issues=%v", issues)
	}
	if !shouldRewriteAsSafeFailureForGoal(goal, &ActionEvidenceLedger{}, issues, resp) {
		t.Fatal("missing artifact evidence would normally rewrite")
	}
	canvasAsk := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "Create a Neural Canvas of the meeting notes")
	if shouldKeepChatSurfaceResponse(canvasAsk, goal, issues, resp) {
		t.Fatal("explicit canvas create must not use chat-surface keep path")
	}
}

func TestOpenCanvasReviseKeepsEditSoftFailResponse(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{
		ID: "user", Name: "Camron", Type: "human",
	}, "the 3rd item is Arrive in Flordia")
	msg.Metadata = map[string]interface{}{
		"open_artifact": map[string]interface{}{
			"id": "art-trip-1", "renderer_id": "nj.markdown", "title": "Trip Planning",
		},
	}
	goal := TurnGoal{Action: ActionEdit, ExpectedEvidence: []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}}
	issues := []responseValidationIssue{issueUnsupportedEdit, issueMissingRequiredEvidence}
	if !shouldKeepOpenCanvasReviseResponse(msg, goal, issues) {
		t.Fatal("expected keep response for open-canvas list-item revise under Edit goal")
	}
	askGoal := TurnGoal{Action: ActionAskUser, ExpectedEvidence: []EvidenceKind{EvidenceUserAnswer}}
	if !shouldKeepOpenCanvasReviseResponse(msg, askGoal, []responseValidationIssue{issueActionDeflection}) {
		t.Fatal("expected keep response for open-canvas revise under AskUser goal")
	}
	if shouldKeepOpenCanvasReviseResponse(msg, TurnGoal{Action: ActionRun}, issues) {
		t.Fatal("run goals must not use canvas revise keep path")
	}
}
