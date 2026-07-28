package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/intent"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type conversationStateCaptureHub struct {
	*imageGenTestHub
	mu          sync.Mutex
	currentGoal string
	goals       []string
	corrections []capturedCorrection
	promises    map[string]string
	completed   map[string]string
}

type capturedCorrection struct {
	goalID     string
	messageID  string
	supersedes []string
}

func newConversationStateCaptureHub() *conversationStateCaptureHub {
	return &conversationStateCaptureHub{
		imageGenTestHub: &imageGenTestHub{},
		promises:        make(map[string]string),
		completed:       make(map[string]string),
	}
}

func (h *conversationStateCaptureHub) ResolveConversationGoalID(_ string, explicitGoalID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if explicitGoalID != "" {
		return explicitGoalID
	}
	return h.currentGoal
}

func (h *conversationStateCaptureHub) PersistConversationGoal(_, goalID, _, _ string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.currentGoal = goalID
	h.goals = append(h.goals, goalID)
}

func (h *conversationStateCaptureHub) RecordConversationCorrection(_, goalID, messageID, _ string, supersedes []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.corrections = append(h.corrections, capturedCorrection{
		goalID: goalID, messageID: messageID,
		supersedes: append([]string(nil), supersedes...),
	})
}

func (h *conversationStateCaptureHub) RecordConversationActionPromise(_, actionID, goalID, _, _, _ string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.promises[actionID] = goalID
}

func (h *conversationStateCaptureHub) CompleteConversationAction(_, actionID, messageID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.promises[actionID]; !ok {
		return false
	}
	h.completed[actionID] = messageID
	return true
}

func TestIntentClassificationPersistsGoalCorrectionAndContinuation(t *testing.T) {
	hub := newConversationStateCaptureHub()
	a := NewAgent(protocol.AgentTypeGeneral, "Agent", nil, ai.NewMockProvider(), hub)
	a.Info.ID = "agent-1"
	user := protocol.AgentInfo{ID: "user-1", Name: "User", Type: protocol.AgentTypeGeneral}
	original := protocol.NewMessage(protocol.MessageTypeChat, "general", user, "Explain the deployment design.")
	classifyTurnForPersistenceTest(t, a, original)
	if hub.currentGoal != original.ID {
		t.Fatalf("normal goal=%q want %q", hub.currentGoal, original.ID)
	}

	a.replaceChannelHistory("general", []*protocol.Message{original})
	correction := protocol.NewMessage(protocol.MessageTypeChat, "general", user, "No, use the configured AWS provider instead.")
	correctionState := classifyTurnForPersistenceTest(t, a, correction)
	if correctionState.goal.ID != original.ID {
		t.Fatalf("correction goal=%q want retained %q", correctionState.goal.ID, original.ID)
	}
	if len(hub.corrections) != 1 || len(hub.corrections[0].supersedes) != 1 ||
		hub.corrections[0].supersedes[0] != original.ID {
		t.Fatalf("correction did not supersede original instruction: %+v", hub.corrections)
	}

	a.replaceChannelHistory("general", []*protocol.Message{original, correction})
	approval := protocol.NewMessage(protocol.MessageTypeChat, "general", user, "Yes, continue.")
	approval.ReplyTo = "agent-plan-message"
	// The classifier stamps this approval as a continuation (not phrase-matched "yes"/"continue").
	if err := protocol.StampTurnDecision(approval, intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionContinuation,
		RequestedAction: intent.ActionContinue, Action: intent.ActionContinue,
		ContinuationTarget: "agent-plan-message", Mutation: intent.MutationWorkspace,
		Confidence: 0.9, Source: intent.SourceLocalModel,
	}); err != nil {
		t.Fatal(err)
	}
	approval.Metadata["goal_id"] = approval.ID
	approvalState := classifyTurnForPersistenceTest(t, a, approval)
	if approvalState.goal.Action != ActionContinue {
		t.Fatalf("approval action=%q want continue", approvalState.goal.Action)
	}
	if approvalState.goal.ID != original.ID || hub.currentGoal != original.ID {
		t.Fatalf("approval replaced original goal: turn=%q persisted=%q", approvalState.goal.ID, hub.currentGoal)
	}
	if got := approval.Metadata["original_goal_id"]; got != original.ID {
		t.Fatalf("approval original_goal_id=%v want %q", got, original.ID)
	}
	if hub.promises[original.ID] != original.ID {
		t.Fatalf("action promise not correlated to original goal: %+v", hub.promises)
	}
}

func TestActionCompletionRequiresValidatedEvidence(t *testing.T) {
	hub := newConversationStateCaptureHub()
	hub.promises["goal-run"] = "goal-run"
	a := NewAgent(protocol.AgentTypeGeneral, "Agent", nil, ai.NewMockProvider(), hub)
	a.Info.ID = "agent-1"
	msg := protocol.NewMessage(
		protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "user-1", Name: "User", Type: protocol.AgentTypeGeneral},
		"Run the tests.",
	)
	goal := TurnGoal{
		ID: "goal-run", Action: ActionRun,
		ExpectedEvidence: []EvidenceKind{EvidenceCommandRun},
	}
	failed := &turnState{
		agent: a, ctx: context.Background(), msg: msg, goal: goal,
		outcome: turnContinue, evidence: &ActionEvidenceLedger{},
		response:    "I ran the tests.",
		responseMsg: protocol.NewMessage(protocol.MessageTypeChat, "general", a.Info, "result"),
	}
	if err := failed.stepValidateResponse(context.Background()); err != nil {
		t.Fatalf("failed-evidence validation: %v", err)
	}
	if failed.completePersistedAction() {
		t.Fatal("action completed without successful evidence")
	}

	ledger := &ActionEvidenceLedger{}
	ledger.Record(ActionEvidence{Kind: EvidenceCommandRun, Tool: "run_command", Status: "succeeded"})
	succeeded := &turnState{
		agent: a, ctx: context.Background(), msg: msg, goal: goal,
		outcome: turnContinue, evidence: ledger,
		response:    "The test command was executed.",
		responseMsg: protocol.NewMessage(protocol.MessageTypeChat, "general", a.Info, "result"),
	}
	if err := succeeded.stepValidateResponse(context.Background()); err != nil {
		t.Fatalf("successful-evidence validation: %v", err)
	}
	if !succeeded.completePersistedAction() {
		t.Fatal("validated authoritative evidence did not complete action")
	}
	if hub.completed["goal-run"] != succeeded.responseMsg.ID {
		t.Fatalf("completion message=%q want %q", hub.completed["goal-run"], succeeded.responseMsg.ID)
	}
}

func TestFailedVerificationIsNotSuccessfulActionEvidence(t *testing.T) {
	ledger := &ActionEvidenceLedger{}
	st := &turnState{
		evidence: ledger,
		implSessionOutcome: map[string]interface{}{
			"outcome": "applied_verify_failed",
		},
	}
	st.buildActionEvidence()
	if ledger.Has(EvidenceEditApplied) {
		t.Fatalf("failed verification counted as successful evidence: %+v", ledger.Entries())
	}
}

func classifyTurnForPersistenceTest(t *testing.T, a *Agent, msg *protocol.Message) *turnState {
	t.Helper()
	st := &turnState{
		agent: a, ctx: context.Background(), msg: msg,
		outcome: turnContinue, evidence: &ActionEvidenceLedger{},
	}
	if err := st.stepIntentClassify(context.Background()); err != nil {
		t.Fatalf("intent classify: %v", err)
	}
	return st
}
