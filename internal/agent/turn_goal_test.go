package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDeriveTurnGoalActions(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{ID: "agent-1", Type: protocol.AgentTypeFrontend}}
	tests := []struct {
		name     string
		content  string
		replyTo  string
		want     ActionIntent
		evidence EvidenceKind
	}{
		{name: "answer", content: "Explain dependency injection.", want: ActionAnswer, evidence: EvidenceAnswer},
		{name: "image", content: "Generate an image of a blue ship.", want: ActionImage, evidence: EvidenceImagePosted},
		{name: "indirect image", content: "Let's see what a sample cover art image will look like.", want: ActionImage, evidence: EvidenceImagePosted},
		{name: "edit", content: "Implement the login form in src/App.tsx.", want: ActionEdit, evidence: EvidenceEditProposed},
		{name: "run", content: "Run the test suite.", want: ActionRun, evidence: EvidenceCommandRun},
		{name: "continue", content: "Yes, continue.", replyTo: "prior-goal", want: ActionContinue, evidence: EvidenceEditProposed},
		{name: "ask user", content: "Ask me which deployment target to use.", want: ActionAskUser, evidence: EvidenceUserAnswer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "user"}, tc.content)
			msg.ReplyTo = tc.replyTo
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

	response, _, _, err := a.collectStreamTokens(ctx, msg, "stream-1", tokens)
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

func TestDeriveTurnGoal_chatModeDemotesFalseImageClassification(t *testing.T) {
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
	if goal.Action != ActionAnswer {
		t.Fatalf("false image stamp must demote to answer; got %s goal=%+v", goal.Action, goal)
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
