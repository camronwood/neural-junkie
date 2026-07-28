package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const vagueBootFailureRequest = "Something is wrong with this code I am working on and the app will not boot up, can you sort me out here?"

func TestVagueBootFailureCanonicalizesImplementationTurn(t *testing.T) {
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), hub)
	a.Info.ID = "frontend-1"
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-frontendengineer",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		vagueBootFailureRequest,
	)
	msg.Metadata = map[string]interface{}{
		protocol.IdeMetaEditorMode:     "agent",
		MetadataConversationMode:       ConversationModeCode,
		"context_scope":                "full",
		protocol.IdeMetaRouteAgentType: "frontend",
		"workspace_context":            map[string]interface{}{"workspace_path": "/tmp/project"},
	}
	// The semantic classifier stamps this boot-failure report as a workspace-mutating
	// Edit (not a natural-language phrase match); governance grants the implementer
	// the ability to run the file-edit loop against the shared workspace.
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
		Provenance: "test",
	})
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionTask,
		RequestedAction: intent.ActionEdit,
		Action:          intent.ActionEdit,
		Mutation:        intent.MutationWorkspace,
		Confidence:      0.9,
		Source:          intent.SourceLocalModel,
		ReasonCodes:     []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}

	initialIntent := a.classifyTurnIntentForMessage(msg)
	goal := deriveTurnGoal(a, msg, initialIntent)
	if goal.Action != ActionEdit || !goal.ImplementationSession || goal.Intent != IntentTask {
		t.Fatalf("goal=%+v, want task implementation edit", goal)
	}

	persistTurnConversationState(a, msg, goal)
	if !msg.ImplementationSession() {
		t.Fatal("implementation_session was not canonicalized")
	}
	caps := protocol.ResolveTurnCapabilities(msg)
	if !caps.CanProposeFiles || !caps.CanRunImplSession || !caps.RequiresWorkspace {
		t.Fatalf("capabilities remain contradictory: %+v metadata=%v", caps, msg.Metadata)
	}
	if isCorrection, _ := msg.Metadata["is_correction"].(bool); isCorrection {
		t.Fatal("initial bug report was incorrectly classified as a correction")
	}
}

func TestVagueBootFailureTriggersDiagnosticFixPath(t *testing.T) {
	// Boot/build phrase detection is deprecated for routing — the semantic classifier's
	// stamped TurnDecision (Action=edit/debug + reason codes) drives the diagnostic fix
	// path now, not natural-language phrase matching.
	if messageHasBootOrBuildError(vagueBootFailureRequest) {
		t.Fatal("messageHasBootOrBuildError is deprecated and must always return false")
	}
	if messageImpliesFixLikeIntent(vagueBootFailureRequest, nil) {
		t.Fatal("messageImpliesFixLikeIntent must not phrase-match without a stamped decision")
	}

	a := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm-camron-frontendengineer",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		vagueBootFailureRequest,
	)
	decision := a.ClassifyConversationTrust(msg)
	if conversationContainsString(decision.Reasons, ConversationReasonUserCorrection) {
		t.Fatalf("bug report incorrectly escalated as correction: %+v", decision)
	}
}

func TestBootFailureSafeResponseReportsDiagnosticEvidence(t *testing.T) {
	goal := TurnGoal{Action: ActionEdit, ExpectedEvidence: []EvidenceKind{EvidenceEditApplied}}
	ledger := &ActionEvidenceLedger{}
	ledger.Record(ActionEvidence{Kind: EvidenceCommandRun, Tool: "run_command", Status: "succeeded"})
	if got := safeActionFailure(goal, ledger); got != "I reproduced the workspace failure, but I couldn't produce a grounded file change in this turn." {
		t.Fatalf("failed diagnostic response=%q", got)
	}

	ledger.Record(ActionEvidence{Kind: EvidenceCommandPass, Tool: "run_command", Status: "succeeded"})
	if got := safeActionFailure(goal, ledger); got != "I ran the workspace diagnostic, but it passed and did not reproduce the reported failure. Which command or screen fails when you try to start the app?" {
		t.Fatalf("passing diagnostic response=%q", got)
	}
}
