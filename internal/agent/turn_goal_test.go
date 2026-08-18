package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestDeriveTurnGoalActions verifies deriveTurnGoal maps a stamped TurnDecision to the
// matching ActionIntent/evidence. Routing is stamp-first: deriveTurnGoal no longer infers
// the action from message text when a decision is present (see deriveTurnGoalFromDecision).
func TestDeriveTurnGoalActions(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "agent-1", Type: protocol.AgentTypeFrontend}}
	tests := []struct {
		name       string
		content    string
		decision   intent.Action
		mutation   intent.Mutation
		continueID string
		want       ActionIntent
		evidence   EvidenceKind
	}{
		{name: "answer", content: "Explain dependency injection.", decision: intent.ActionAnswer, mutation: intent.MutationNone, want: ActionAnswer, evidence: EvidenceAnswer},
		{name: "image", content: "Generate an image of a blue ship.", decision: intent.ActionImage, mutation: intent.MutationExternal, want: ActionImage, evidence: EvidenceImagePosted},
		{name: "edit", content: "Implement the login form in src/App.tsx.", decision: intent.ActionEdit, mutation: intent.MutationWorkspace, want: ActionEdit, evidence: EvidenceEditProposed},
		{name: "run", content: "Run the test suite.", decision: intent.ActionRun, mutation: intent.MutationNone, want: ActionRun, evidence: EvidenceCommandRun},
		{name: "continue", content: "Yes, continue.", decision: intent.ActionContinue, mutation: intent.MutationWorkspace, continueID: "prior-goal", want: ActionContinue, evidence: EvidenceEditProposed},
		{name: "ask_user", content: "Ask me which deployment target to use.", decision: intent.ActionAskUser, mutation: intent.MutationNone, want: ActionAskUser, evidence: EvidenceUserAnswer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "user"}, tc.content)
			if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
				SchemaVersion:      intent.SchemaVersion,
				Interaction:        intent.InteractionTask,
				RequestedAction:    tc.decision,
				Action:             tc.decision,
				Mutation:           tc.mutation,
				ContinuationTarget: tc.continueID,
				Confidence:         0.95,
				Source:             intent.SourceLocalModel,
			}); err != nil {
				t.Fatal(err)
			}
			goal := deriveTurnGoal(a, msg, IntentTask)
			if goal.Action != tc.want {
				t.Fatalf("action=%q want %q", goal.Action, tc.want)
			}
			found := false
			for _, kind := range goal.ExpectedEvidence {
				found = found || kind == tc.evidence
			}
			if !found {
				t.Fatalf("expected evidence %q in %+v", tc.evidence, goal.ExpectedEvidence)
			}
			if goal.ID != msg.ID {
				t.Fatalf("goal ID=%q want message ID %q", goal.ID, msg.ID)
			}
		})
	}
}

func TestTurnIntentForContextUsesSingleClassifiedGoal(t *testing.T) {
	a := &Agent{}
	msg := &protocol.Message{Content: "hello"}
	ctx := contextWithTurnGoal(context.Background(), TurnGoal{Intent: IntentTask})
	if got := turnIntentForContext(ctx, a, msg); got != IntentTask {
		t.Fatalf("intent=%s want task", got)
	}
}

func TestValidateResponseRejectsUnsupportedActionClaims(t *testing.T) {
	goal := TurnGoal{Action: ActionImage, ExpectedEvidence: []EvidenceKind{EvidenceImagePosted}}
	ledger := &ActionEvidenceLedger{}
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, "Done — I generated and posted the image.", nil)
	if len(issues) == 0 {
		t.Fatal("expected unsupported image claim")
	}
	if got := safeActionFailure(goal, ledger); got != "I couldn't generate or post the requested image in this turn." {
		t.Fatalf("unexpected rewrite: %q", got)
	}

	ledger.Record(ActionEvidence{Kind: EvidenceImagePosted, Tool: generateImageToolName, Status: "succeeded"})
	if issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, "Done — I generated and posted the image.", nil); len(issues) != 0 {
		t.Fatalf("supported image claim rejected: %v", issues)
	}
}

func TestValidateResponseRewritesAppliedClaimToProposalStatus(t *testing.T) {
	goal := TurnGoal{Action: ActionEdit, ExpectedEvidence: []EvidenceKind{EvidenceEditProposed, EvidenceEditApplied}}
	ledger := &ActionEvidenceLedger{}
	ledger.Record(ActionEvidence{Kind: EvidenceEditProposed, Tool: proposeFileEditToolName, Status: "succeeded"})
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, "I implemented and updated the code change.", nil)
	if len(issues) == 0 {
		t.Fatal("proposal evidence must not support an applied claim")
	}
	if got := safeActionFailure(goal, ledger); got != "I submitted the file changes as proposals; they have not been applied yet." {
		t.Fatalf("unexpected proposal rewrite: %q", got)
	}
}

func TestValidateResponseRequiresPassingCommandEvidence(t *testing.T) {
	goal := TurnGoal{Action: ActionRun, ExpectedEvidence: []EvidenceKind{EvidenceCommandRun}}
	ledger := &ActionEvidenceLedger{}
	code := 1
	ledger.Record(ActionEvidence{Kind: EvidenceCommandRun, Tool: "run_command", Status: "succeeded", ExitCode: &code})
	if issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, "I ran the tests and they passed.", nil); len(issues) == 0 {
		t.Fatal("failed command must not support a passing claim")
	}
}

func TestValidateResponseKeepsCreativeAnswerUnderMisclassifiedRunGoal(t *testing.T) {
	goal := TurnGoal{Action: ActionRun, ExpectedEvidence: []EvidenceKind{EvidenceCommandRun}}
	ledger := &ActionEvidenceLedger{}
	answer := "Here is an alternate ending where Arya sails west and the throne dissolves into a council of cities."
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, answer, nil)
	if shouldRewriteAsSafeFailure(issues, answer) {
		t.Fatalf("creative answer under misclassified run goal should be kept; issues=%v", issues)
	}
}

func TestValidateResponseStillRewritesUnsupportedRunClaim(t *testing.T) {
	goal := TurnGoal{Action: ActionRun, ExpectedEvidence: []EvidenceKind{EvidenceCommandRun}}
	ledger := &ActionEvidenceLedger{}
	claim := "I ran the test suite and everything passed."
	issues := validateResponseAgainstEvidence(goal, ledger, &protocol.Message{}, claim, nil)
	if !shouldRewriteAsSafeFailure(issues, claim) {
		t.Fatalf("unsupported run claim must soft-fail; issues=%v", issues)
	}
}

func TestValidateResponseRewritesFakeNeuralCanvasProse(t *testing.T) {
	goal := TurnGoal{Action: ActionAnswer, ExpectedEvidence: []EvidenceKind{"answer"}}
	ledger := &ActionEvidenceLedger{}
	msg := &protocol.Message{Content: "create the canvans with this information please"}
	claim := "Based on the meeting notes, I will create a neural canvas to visualize the key points.\n\n**Neural Canvas: PHOENIX TEAM MEETING (July 21, 2026)**\n\nSection 1: Meeting Summary\n"
	issues := validateResponseAgainstEvidence(goal, ledger, msg, claim, nil)
	if !containsValidationIssue(issues, issueUnsupportedArtifact) {
		t.Fatalf("fake Neural Canvas prose must flag unsupported_artifact_claim; issues=%v", issues)
	}
	if !shouldRewriteAsSafeFailure(issues, claim) {
		t.Fatal("fake Neural Canvas prose must rewrite")
	}
}

func containsValidationIssue(issues []responseValidationIssue, want responseValidationIssue) bool {
	for _, issue := range issues {
		if issue == want {
			return true
		}
	}
	return false
}

func TestActionEvidenceRecordsToolOutcomes(t *testing.T) {
	ledger := &ActionEvidenceLedger{}
	ledger.recordToolEvent(ai.ToolStepEvent{Kind: "result", Name: proposeFileEditToolName, Preview: "proposal registered"})
	if !ledger.Has(EvidenceEditProposed) {
		t.Fatalf("missing proposal evidence: %+v", ledger.Entries())
	}
}

type streamCaptureHub struct {
	*imageGenTestHub
	broadcasts []*protocol.Message
}

func (h *streamCaptureHub) BroadcastDirect(_ string, msg *protocol.Message) {
	h.broadcasts = append(h.broadcasts, msg)
}

func TestActionTurnStreamingIsBufferedForValidation(t *testing.T) {
	hub := &streamCaptureHub{imageGenTestHub: &imageGenTestHub{}}
	a := &Agent{Info: protocol.AgentInfo{ID: "agent-1"}, Hub: hub}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "user"}, "Run the tests.")
	ctx := contextWithTurnGoal(context.Background(), TurnGoal{
		Action:           ActionRun,
		ExpectedEvidence: []EvidenceKind{EvidenceCommandRun},
	})
	tokens := make(chan ai.StreamToken, 2)
	tokens <- ai.StreamToken{Content: "The tests passed."}
	tokens <- ai.StreamToken{Done: true}
	close(tokens)

	response, _, _, err := a.collectStreamTokens(ctx, msg, "stream-1", tokens, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response != "The tests passed." {
		t.Fatalf("response=%q", response)
	}
	for _, sent := range hub.broadcasts {
		if sent.Type == protocol.MessageTypeStreamDelta {
			t.Fatal("action response delta was visible before evidence validation")
		}
	}
}

func TestDeriveTurnGoal_chatModeKeepsAnswerDespiteDesignWording(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"Design a theme settings flow. Keep the toggle in an Appearance section and call the component ThemeSettings.")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Mutation: intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: false, CanRunImplSession: false,
	})
	goal := deriveTurnGoal(a, msg, IntentSubstantive)
	if goal.Action != ActionAnswer {
		t.Fatalf("action=%s want answer; goal=%+v", goal.Action, goal)
	}
}

func TestDeriveTurnGoal_chatModePreservesWorkspaceFix(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"fix the app")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeFull,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionDebug, Action: intent.ActionDebug,
		Mutation: intent.MutationWorkspace, Confidence: 1, Source: intent.SourceLocalModel,
		ReasonCodes: []string{"runtime_failure"},
	}); err != nil {
		t.Fatal(err)
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: true, CanRunImplSession: true, RequiresWorkspace: true,
	})
	goal := deriveTurnGoal(a, msg, IntentTask)
	if goal.Action != ActionDebug {
		t.Fatalf("action=%s want debug; goal=%+v", goal.Action, goal)
	}
	if goal.Mutation != MutationWorkspace {
		t.Fatalf("mutation=%s want workspace", goal.Mutation)
	}
}

// TestDeriveTurnGoal_chatModeTrustsImageStampRegardlessOfWording documents the stamp-first
// replacement for the old UserRequestsGeneratedImage re-check: a stamped ActionImage // phrase-migration-shim
// decision is authoritative and is never demoted back to Answer by inspecting the message
// text for image phrases, even in advisory conversation_mode=chat.
func TestDeriveTurnGoal_chatModeTrustsImageStampRegardlessOfWording(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"Design a theme settings flow. Keep the toggle in an Appearance section and call the component ThemeSettings.")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionImage, Action: intent.ActionImage,
		Mutation: intent.MutationExternal, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: false, CanRunImplSession: false,
	})
	goal := deriveTurnGoal(a, msg, IntentSubstantive)
	if goal.Action != ActionImage {
		t.Fatalf("stamped image must stay image regardless of wording; got %s goal=%+v", goal.Action, goal)
	}
}

func TestDeriveTurnGoal_chatModeKeepsExplicitImageRequest(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"Generate an image of a theme settings mockup.")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	if err := protocol.StampTurnDecision(msg, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionTask,
		RequestedAction: intent.ActionImage, Action: intent.ActionImage,
		Mutation: intent.MutationExternal, Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	goal := deriveTurnGoal(a, msg, IntentSubstantive)
	if goal.Action != ActionImage {
		t.Fatalf("explicit image ask must stay image; got %s", goal.Action)
	}
}

func TestDeriveTurnGoal_chatModeWithoutDecisionStaysAnswer(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "fe", Type: protocol.AgentTypeFrontend, Name: "FrontendEngineer"}}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{ID: "u", Name: "User", Type: "human"},
		"Design a theme settings flow. Keep the toggle in an Appearance section and call the component ThemeSettings.")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		MetadataContextScope:     ContextScopeNone,
	}
	goal := deriveTurnGoal(a, msg, IntentSubstantive)
	if goal.Action != ActionAnswer {
		t.Fatalf("action=%s want answer without decision; goal=%+v", goal.Action, goal)
	}
}

func TestDeriveTurnGoal_userFlowRustBlackjackForcesSession(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend, Name: "BackendEngineer"}}
	content := "Let's design and implement a simple game using Rust. This will be a local CLI card game where the user can play blackjack against the house. Keep everything local (no network). Prefer a terminal/CLI or simple text UI — not a 3D engine. Put the crate at the workspace root with Cargo.toml and src/main.rs. Include hit/stand, dealer play, and win/lose so cargo build works and the game is playable."
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"user-flow-scenarios",
		protocol.AgentInfo{ID: "u1", Name: "User"},
		content,
	)
	msg.Metadata = map[string]interface{}{
		"editor_mode":            "agent",
		"ide_route_agent_type":   "backend",
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"conversation_mode":      "code",
	}
	protocol.StampTurnGovernance(msg, protocol.TurnGovernance{
		ComposerMode: "agent", CanProposeFiles: false, CanRunImplSession: false, RequiresWorkspace: false,
		Provenance: "test_demoted_caps",
	})
	decision := intent.TurnDecision{
		SchemaVersion:   intent.SchemaVersion,
		Interaction:     intent.InteractionTask,
		RequestedAction: intent.ActionAnswer,
		Action:          intent.ActionAnswer,
		Mutation:        intent.MutationNone,
		Confidence:      0.9,
		Source:          intent.SourceLocalModel,
	}
	if err := protocol.StampTurnDecision(msg, decision); err != nil {
		t.Fatal(err)
	}
	if !shouldRunImplementationSession(a, msg) {
		t.Fatal("shouldRunImplementationSession must force user-flow harness turns")
	}
	goal := deriveTurnGoal(a, msg, IntentTask)
	if !turnGoalRunsImplementationSession(goal) {
		t.Fatalf("turn goal must run implementation session; goal=%+v", goal)
	}
	if goal.Action != ActionEdit {
		t.Fatalf("action=%s want edit after harness sync", goal.Action)
	}
}
