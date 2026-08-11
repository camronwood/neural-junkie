package hub

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClearChannelHistoryClearsConversationState(t *testing.T) {
	h := NewHub()
	ch := &protocol.Channel{Name: "dm-clear-test", Type: protocol.ChannelTypeDM}
	h.channels = map[string]*protocol.Channel{ch.Name: ch}
	h.messages = map[string][]*protocol.Message{ch.Name: {}}
	h.SetCurrentGoal(ch.Name, "goal-1", "msg-1", "Implement the API")
	h.RecordConversationActionPromise(ch.Name, "pending", "goal-1", "edit", "write tests", "promise-1")
	if st := h.GetChannelConversationState(ch.Name); st == nil || st.CurrentGoal == nil {
		t.Fatal("expected durable state before clear")
	}
	if err := h.ClearChannelHistory(ch.Name); err != nil {
		t.Fatal(err)
	}
	if st := h.GetChannelConversationState(ch.Name); st != nil && st.CurrentGoal != nil {
		t.Fatalf("clear history must drop durable conversation state, got %+v", st.CurrentGoal)
	}
}

func TestConversationState_recordsEntitiesAndOpenQuestions(t *testing.T) {
	h := NewHub()
	ch := &protocol.Channel{Name: "dm-ent", Type: protocol.ChannelTypeDM}
	h.channels = map[string]*protocol.Channel{ch.Name: ch}
	h.messages = map[string][]*protocol.Message{ch.Name: {}}
	h.SetCurrentGoal("dm-ent", "goal-1", "msg-1", "Build ThemeSettings for the app")
	h.RememberConversationSurface("dm-ent", "goal-1", "msg-2", "should we use a segmented control?")
	st := h.GetChannelConversationState("dm-ent")
	if st == nil {
		t.Fatal("expected state")
	}
	found := false
	for _, e := range st.NamedEntities {
		if e.Name == "ThemeSettings" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("named entities=%v, want ThemeSettings", st.NamedEntities)
	}
	if len(st.OpenQuestions) == 0 || !strings.Contains(st.OpenQuestions[0].Text, "segmented") {
		t.Fatalf("open questions=%v", st.OpenQuestions)
	}
	env := h.GetTurnConversationContext("dm-ent")
	if len(env.NamedEntities) == 0 || len(env.OpenQuestions) == 0 {
		t.Fatalf("turn context missing surface state: %+v", env)
	}
	if err := h.ClearChannelHistory("dm-ent"); err != nil {
		t.Fatal(err)
	}
	st = h.GetChannelConversationState("dm-ent")
	if st != nil && (len(st.NamedEntities) > 0 || len(st.OpenQuestions) > 0) {
		t.Fatalf("clear must drop entities/questions: %+v", st)
	}
}

func TestSetCurrentGoal_retainsPinnedTextAcrossFixFollowUps(t *testing.T) {
	h := NewHub()
	h.SetCurrentGoal("dm-pin", "goal-1", "msg-1", "The blank screen in App.tsx needs a real fix")
	h.SetCurrentGoal("dm-pin", "goal-2", "msg-2", "fix the app")
	st := h.GetChannelConversationState("dm-pin")
	if st == nil || st.CurrentGoal == nil {
		t.Fatal("expected goal")
	}
	if st.CurrentGoal.PinnedText != "The blank screen in App.tsx needs a real fix" {
		t.Fatalf("pinned=%q, want original task", st.CurrentGoal.PinnedText)
	}
	if st.CurrentGoal.MessageID != "msg-1" {
		t.Fatalf("message_id=%q, want original msg-1 for history pin", st.CurrentGoal.MessageID)
	}
	if st.CurrentGoal.LastMessageID != "msg-2" {
		t.Fatalf("last_message_id=%q, want msg-2", st.CurrentGoal.LastMessageID)
	}
	env := h.GetTurnConversationContext("dm-pin")
	if env.Goal == nil || env.Goal.PinnedText != st.CurrentGoal.PinnedText {
		t.Fatalf("turn context missing pinned text: %+v", env.Goal)
	}
}

func TestChannelConversationState_CorrectionSupersedesAtomically(t *testing.T) {
	h := NewHub()
	h.SetCurrentGoal("general", "goal-1", "message-goal", "Build the service")
	h.RecordConversationCorrection(
		"general", "goal-1", "message-correction", "Use Rust instead",
		[]string{"message-old-language"},
	)

	state := h.GetChannelConversationState("general")
	if state.CurrentGoal == nil || state.CurrentGoal.ID != "goal-1" {
		t.Fatalf("current goal not retained: %+v", state.CurrentGoal)
	}
	if len(state.Corrections) != 1 {
		t.Fatalf("corrections=%d want 1", len(state.Corrections))
	}
	superseded, ok := state.SupersededInstructions["message-old-language"]
	if !ok || superseded.SupersededByMessageID != "message-correction" {
		t.Fatalf("superseded instruction not correlated: %+v", superseded)
	}
}

func TestTurnConversationContextContainsOnlyUnresolvedActions(t *testing.T) {
	h := NewHub()
	h.SetCurrentGoal("general", "goal-1", "goal-message", "Build the service")
	h.RecordConversationActionPromise("general", "pending", "goal-1", "edit", "write tests", "promise-1")
	h.RecordConversationActionPromise("general", "done", "goal-1", "edit", "write code", "promise-2")
	h.CompleteConversationAction("general", "done", "complete-2")
	h.RecordConversationCorrection("general", "goal-1", "correction", "Use Rust", []string{"old"})

	envelope := h.GetTurnConversationContext("general")
	if envelope.Goal == nil || envelope.Goal.ID != "goal-1" {
		t.Fatalf("goal missing from envelope: %+v", envelope.Goal)
	}
	if len(envelope.UnresolvedActions) != 1 || envelope.UnresolvedActions[0].ID != "pending" {
		t.Fatalf("unexpected unresolved actions: %+v", envelope.UnresolvedActions)
	}
	if len(envelope.SupersededMessageIDs) != 1 || envelope.SupersededMessageIDs[0] != "old" {
		t.Fatalf("unexpected superseded IDs: %+v", envelope.SupersededMessageIDs)
	}
}

func TestResolveConversationGoalID_ApprovalContinuationRetainsGoal(t *testing.T) {
	h := NewHub()
	h.SetCurrentGoal("general", "goal-original", "request-message", "Implement feature")
	if got := h.ResolveConversationGoalID("general", ""); got != "goal-original" {
		t.Fatalf("continuation goal=%q want goal-original", got)
	}
	if got := h.ResolveConversationGoalID("general", "goal-explicit"); got != "goal-explicit" {
		t.Fatalf("explicit goal=%q want goal-explicit", got)
	}
}

func TestChannelConversationState_RaceSafeActions(t *testing.T) {
	h := NewHub()
	const count = 64
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("action-%d", i)
			h.RecordPromisedAction("general", ConversationAction{
				ID: id, GoalID: "goal-1", Description: "action",
				PromisedMessageID: fmt.Sprintf("promise-%d", i),
			})
			if !h.CompletePromisedAction("general", id, fmt.Sprintf("complete-%d", i)) {
				t.Errorf("failed to complete %s", id)
			}
			_ = h.GetChannelConversationState("general")
		}()
	}
	wg.Wait()

	state := h.GetChannelConversationState("general")
	if len(state.Actions) != count {
		t.Fatalf("actions=%d want %d", len(state.Actions), count)
	}
	for id, action := range state.Actions {
		if action.CompletedAt == nil {
			t.Fatalf("%s was not completed", id)
		}
	}
}

func TestChannelConversationState_DedupesConcurrentCorrection(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.RecordConversationCorrection(
				"general", "goal-1", "correction-1", "Use Rust",
				[]string{"instruction-1"},
			)
		}()
	}
	wg.Wait()
	state := h.GetChannelConversationState("general")
	if len(state.Corrections) != 1 {
		t.Fatalf("corrections=%d want 1", len(state.Corrections))
	}
	if state.SupersededInstructions["instruction-1"].SupersededByMessageID != "correction-1" {
		t.Fatalf("supersession missing: %+v", state.SupersededInstructions)
	}
}

func TestSessionPersistence_RestoresConversationAndResolvedQuestion(t *testing.T) {
	h := NewHub()
	h.SetCurrentGoal("general", "goal-original", "message-original", "Ship the API")
	uqm := h.GetUserQuestionManager()

	result := make(chan string, 1)
	go func() {
		answer, err := uqm.AskWithContext(
			"a1", "Architect", "general", "Which database?", nil,
			"goal-original", "database", time.Second,
		)
		if err != nil {
			result <- "ERROR: " + err.Error()
			return
		}
		result <- answer
	}()
	pending := waitForPendingQuestion(t, uqm)
	if err := uqm.Answer(pending.ID, "PostgreSQL"); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if got := <-result; got != "PostgreSQL" {
		t.Fatalf("answer=%q", got)
	}

	path := filepath.Join(t.TempDir(), "last-session.json")
	if err := h.SaveSessionToFile(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	restored := NewHub()
	if err := restored.LoadSessionFromFile(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	state := restored.GetChannelConversationState("general")
	if state.CurrentGoal == nil || state.CurrentGoal.ID != "goal-original" {
		t.Fatalf("restored goal: %+v", state.CurrentGoal)
	}
	decision, ok := state.AnsweredDecisions[decisionStateKey("goal-original", "database")]
	if !ok || decision.Answer != "PostgreSQL" {
		t.Fatalf("restored decision: %+v, ok=%v", decision, ok)
	}
	restoredQuestions := restored.GetUserQuestionManager()
	restoredQuestions.mu.Lock()
	if question := restoredQuestions.questions[pending.ID]; question == nil || question.Status != UserQuestionAnswered {
		restoredQuestions.mu.Unlock()
		t.Fatalf("resolved question was not reconstructed: %+v", question)
	}
	// Durable decision state must also dedupe after in-memory question cleanup.
	delete(restoredQuestions.questions, pending.ID)
	restoredQuestions.mu.Unlock()
	answer, err := restoredQuestions.AskWithContext(
		"a2", "Backend", "general", "Choose the persistence engine", nil,
		"goal-original", "database", 50*time.Millisecond,
	)
	if err != nil || answer != "PostgreSQL" {
		t.Fatalf("restored question dedupe answer=%q err=%v", answer, err)
	}
	if len(restoredQuestions.ListPending()) != 0 {
		t.Fatal("restored decision must not create another question")
	}
}

func TestSessionPersistence_RestoresIntegratedGoalCorrectionAndApproval(t *testing.T) {
	h := NewHub()
	h.PersistConversationGoal("general", "goal-message", "goal-message", "Implement the API")
	h.RecordConversationCorrection(
		"general", "goal-message", "correction-message", "Use Rust instead",
		[]string{"goal-message"},
	)
	h.PersistConversationGoal("general", "goal-message", "approval-message", "Yes, continue")
	h.RecordConversationActionPromise(
		"general", "goal-message", "goal-message", "edit", "Implement the API", "approval-message",
	)

	path := filepath.Join(t.TempDir(), "last-session.json")
	if err := h.SaveSessionToFile(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	restored := NewHub()
	if err := restored.LoadSessionFromFile(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	state := restored.GetChannelConversationState("general")
	if state == nil || state.CurrentGoal == nil {
		t.Fatal("conversation state did not restore")
	}
	if state.CurrentGoal.ID != "goal-message" ||
		state.CurrentGoal.MessageID != "goal-message" ||
		state.CurrentGoal.LastMessageID != "approval-message" {
		t.Fatalf("restored goal lost origin/continuation: %+v", state.CurrentGoal)
	}
	if superseded := state.SupersededInstructions["goal-message"]; superseded.SupersededByMessageID != "correction-message" {
		t.Fatalf("correction supersession did not restore: %+v", superseded)
	}
	action, ok := state.Actions["goal-message"]
	if !ok || action.GoalID != "goal-message" || action.CompletedAt != nil {
		t.Fatalf("promised action did not restore pending: %+v, ok=%v", action, ok)
	}
}

func TestUserQuestionManager_DecisionKeyScopedByGoal(t *testing.T) {
	h := NewHub()
	uqm := h.GetUserQuestionManager()
	result := make(chan string, 1)
	go func() {
		answer, _ := uqm.AskWithContext(
			"a1", "Architect", "general", "Desktop or web?", nil,
			"goal-a", "target_platform", time.Second,
		)
		result <- answer
	}()
	pending := waitForPendingQuestion(t, uqm)
	if err := uqm.Answer(pending.ID, "Desktop"); err != nil {
		t.Fatal(err)
	}
	<-result

	answer, err := uqm.AskWithContext(
		"a1", "Architect", "general", "Where should it run?", nil,
		"goal-a", "target_platform", 50*time.Millisecond,
	)
	if err != nil || answer != "Desktop" {
		t.Fatalf("same goal/key did not dedupe: answer=%q err=%v", answer, err)
	}

	other := make(chan error, 1)
	go func() {
		_, askErr := uqm.AskWithContext(
			"a1", "Architect", "general", "Where should it run?", nil,
			"goal-b", "target_platform", time.Second,
		)
		other <- askErr
	}()
	pending = waitForPendingQuestion(t, uqm)
	if pending.GoalID != "goal-b" {
		t.Fatalf("goal=%q want goal-b", pending.GoalID)
	}
	if err := uqm.Answer(pending.ID, "Web"); err != nil {
		t.Fatal(err)
	}
	if err := <-other; err != nil {
		t.Fatal(err)
	}
}

func waitForPendingQuestion(t *testing.T, uqm *UserQuestionManager) *UserQuestion {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if pending := uqm.ListPending(); len(pending) > 0 {
			return pending[0]
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("question did not become pending")
	return nil
}
