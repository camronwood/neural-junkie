package collaboration

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestParticipationQuorumMet(t *testing.T) {
	t.Parallel()
	arch := "arch-id"
	be := "be-id"

	d := &DiscussionSession{
		Participants:      []string{arch, be},
		TotalMessageCount: 2,
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: arch, Name: "SoftwareArchitect"}, "plan"),
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: be, Name: "BackendEngineer"}, "agree"),
		},
	}
	if !participationQuorumMet(d) {
		t.Fatal("expected quorum when both participants spoke")
	}

	d.Messages = d.Messages[:1]
	d.TotalMessageCount = 1
	if participationQuorumMet(d) {
		t.Fatal("expected no quorum when one participant is silent")
	}
}

func TestSilentPlanningParticipantIDs(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	silent := cm.SilentPlanningParticipantIDs(collab.ID)
	if len(silent) != 2 {
		t.Fatalf("expected both participants silent, got %v", silent)
	}

	discMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatal(err)
	}
	silent = cm.SilentPlanningParticipantIDs(collab.ID)
	if len(silent) != 1 || silent[0] != "be-id" {
		t.Fatalf("expected only BackendEngineer silent, got %v", silent)
	}
	if got := cm.ParticipantAgentName(collab.ID, "be-id"); got != "BackendEngineer" {
		t.Fatalf("ParticipantAgentName = %q", got)
	}
}

func TestDiscussionTurnBudget_untilQuorum(t *testing.T) {
	t.Parallel()
	arch := "arch-id"
	be := "be-id"
	c := &Collaboration{Phase: PhasePlanning}
	d := &DiscussionSession{
		Participants: []string{arch, be},
		TurnBudget:   2,
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: arch, Name: "SoftwareArchitect"}, "plan"),
		},
		TotalMessageCount: 1,
	}
	if got := discussionTurnBudget(c, d); got != 1 {
		t.Fatalf("expected turn budget 1 before quorum, got %d", got)
	}
	d.Messages = append(d.Messages, protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
		protocol.AgentInfo{ID: be, Name: "BackendEngineer"}, "agree"))
	d.TotalMessageCount = 2
	if got := discussionTurnBudget(c, d); got != 2 {
		t.Fatalf("expected full turn budget after quorum, got %d", got)
	}
}

func TestPlanningDiscussionTimeoutElapsed_graceForSilent(t *testing.T) {
	t.Parallel()
	c := &Collaboration{Phase: PhasePlanning}
	d := &DiscussionSession{
		Participants:      []string{"a1", "a2", "a3"},
		Timeout:           2 * time.Minute,
		MaxTotalMessages:  6,
		TotalMessageCount: 1,
		StartedAt:         time.Now().Add(-3 * time.Minute),
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: "a1", Name: "A"}, "plan"),
		},
	}
	if planningDiscussionTimeoutElapsed(c, d) {
		t.Fatal("expected grace while two participants remain silent")
	}
	d.StartedAt = time.Now().Add(-6 * time.Minute)
	if !planningDiscussionTimeoutElapsed(c, d) {
		t.Fatal("expected timeout after grace window")
	}
}

func TestPlanningDiscussionTimeoutElapsed_graceBeforeFirstRecordedMessage(t *testing.T) {
	t.Parallel()
	c := &Collaboration{Phase: PhasePlanning}
	d := &DiscussionSession{
		Participants:      []string{"a1", "a2"},
		Timeout:           3 * time.Minute,
		MaxTotalMessages:  4,
		TotalMessageCount: 0,
		StartedAt:         time.Now().Add(-4 * time.Minute),
	}
	if planningDiscussionTimeoutElapsed(c, d) {
		t.Fatal("expected extra grace before any message is recorded")
	}
	d.StartedAt = time.Now().Add(-6 * time.Minute)
	if !planningDiscussionTimeoutElapsed(c, d) {
		t.Fatalf(
			"expected timeout after first-reply grace (elapsed=%v timeout=%v queue_factor=%d silent=%d)",
			time.Since(d.StartedAt),
			d.Timeout,
			planningOllamaQueueFactor(len(d.Participants)),
			len(silentParticipantIDsLocked(d)),
		)
	}
}

func TestRecordMessage_generationErrorDoesNotAdvanceTurn(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	discMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatal(err)
	}

	cm.mu.RLock()
	turnAfterArch := collab.Discussion.CurrentTurnIndex
	cm.mu.RUnlock()
	if turnAfterArch != 1 {
		t.Fatalf("expected turn index 1 (BackendEngineer) after architect spoke, got %d", turnAfterArch)
	}

	errMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "be-id", Name: "BackendEngineer"},
		"**BackendEngineer** could not complete this turn: timeout",
	)
	errMsg.Metadata = map[string]interface{}{"generation_error": true}
	if err := cm.RecordMessage(collab.ID, errMsg); err != nil {
		t.Fatal(err)
	}

	cm.mu.RLock()
	turnAfterErr := collab.Discussion.CurrentTurnIndex
	cm.mu.RUnlock()
	if turnAfterErr != 1 {
		t.Fatalf("generation_error should not advance turn, got index %d", turnAfterErr)
	}
}

func TestRecordMessage_generationErrorAdvancesTurnAfterRepeatedFailures(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	discMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxPlanningGenerationErrorsPerTurn; i++ {
		errMsg := protocol.NewMessage(
			protocol.MessageTypeCollabDiscussion,
			collab.Channel,
			protocol.AgentInfo{ID: "be-id", Name: "BackendEngineer"},
			"**BackendEngineer** could not complete this turn: timeout",
		)
		errMsg.Metadata = map[string]interface{}{"generation_error": true}
		if err := cm.RecordMessage(collab.ID, errMsg); err != nil {
			t.Fatal(err)
		}
	}

	cm.mu.RLock()
	turnAfter := collab.Discussion.CurrentTurnIndex
	cm.mu.RUnlock()
	if turnAfter != 0 {
		t.Fatalf("expected turn to advance back to architect after %d errors, got index %d",
			maxPlanningGenerationErrorsPerTurn, turnAfter)
	}
}

func TestRecordMessage_planningCooldownRejectsDominatingSpeaker(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	first := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, first); err != nil {
		t.Fatal(err)
	}

	second := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan again",
	)
	if err := cm.RecordMessage(collab.ID, second); err == nil {
		t.Fatal("expected architect blocked from second planning turn before backend speaks")
	}
}

func TestRecordMessage_generationErrorDoesNotConsumeTurn(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	errMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "be-id", Name: "BackendEngineer"},
		"**BackendEngineer** could not complete this turn: timeout",
	)
	errMsg.Metadata = map[string]interface{}{"generation_error": true}
	if err := cm.RecordMessage(collab.ID, errMsg); err != nil {
		t.Fatal(err)
	}

	cm.mu.RLock()
	d := collab.Discussion
	cm.mu.RUnlock()
	if d.TotalMessageCount != 0 {
		t.Fatalf("generation_error should not consume message budget, got %d", d.TotalMessageCount)
	}
	if d.TurnsThisRound["be-id"] != 0 {
		t.Fatalf("generation_error should not consume turn budget, got %d", d.TurnsThisRound["be-id"])
	}
	silent := cm.SilentPlanningParticipantIDs(collab.ID)
	if len(silent) != 2 {
		t.Fatalf("expected both participants still silent for quorum after generation_error, got %v", silent)
	}
}

func TestSkipStuckSilentPlanningTurn(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	h.addAgent("fe-id", "FrontendEngineer", protocol.AgentTypeFrontend, nil)
	h.addAgent("claude-id", "Claude", protocol.AgentTypeCLI, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"goal",
		[]string{"be-id", "fe-id", "claude-id"},
		"ch",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 10},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	turn, err := cm.GetCurrentTurnAgent(collab.ID)
	if err != nil || turn != "be-id" {
		t.Fatalf("expected be-id turn, got %q err=%v", turn, err)
	}
	next, ok := cm.SkipStuckSilentPlanningTurn(collab.ID)
	if !ok || next != "fe-id" {
		t.Fatalf("expected skip to fe-id, got %q ok=%v", next, ok)
	}
	turn, err = cm.GetCurrentTurnAgent(collab.ID)
	if err != nil || turn != "fe-id" {
		t.Fatalf("expected fe-id after skip, got %q err=%v", turn, err)
	}
	// After one participant speaks, only one silent peer remains — do not skip.
	first := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "fe-id", Name: "FrontendEngineer"},
		"fe plan",
	)
	if err := cm.RecordMessage(collab.ID, first); err != nil {
		t.Fatal(err)
	}
	if _, ok := cm.SkipStuckSilentPlanningTurn(collab.ID); ok {
		t.Fatal("must not skip when only one silent participant remains")
	}
}

func TestParticipationQuorumMetSolo(t *testing.T) {
	t.Parallel()
	solo := "solo-id"
	d := &DiscussionSession{
		Participants:      []string{solo},
		TotalMessageCount: 1,
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: solo, Name: "Gemini"}, "plan ready"),
		},
	}
	if !participationQuorumMet(d) {
		t.Fatal("expected solo quorum after one real message")
	}
}

func TestScaledDiscussionConfigSolo(t *testing.T) {
	t.Parallel()
	cfg := ScaledDiscussionConfig(1)
	if cfg.MaxRounds != 1 || cfg.TurnBudget != 1 || cfg.MaxTotalMessages != 4 {
		t.Fatalf("unexpected solo caps: %+v", cfg)
	}
}

func TestRecordMessage_generationErrorInvokesPlanningTurnAdvanced(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	first := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, first); err != nil {
		t.Fatal(err)
	}

	advanced := make(chan struct{}, 1)
	var gotCollab, gotAgent string
	cm.SetOnPlanningTurnAdvanced(func(collabID, nextAgentID string) {
		gotCollab = collabID
		gotAgent = nextAgentID
		advanced <- struct{}{}
	})

	for i := 0; i < maxPlanningGenerationErrorsPerTurn; i++ {
		errMsg := protocol.NewMessage(
			protocol.MessageTypeCollabDiscussion,
			collab.Channel,
			protocol.AgentInfo{ID: "be-id", Name: "BackendEngineer"},
			"**BackendEngineer** could not complete this turn: timeout",
		)
		errMsg.Metadata = map[string]interface{}{"generation_error": true}
		if err := cm.RecordMessage(collab.ID, errMsg); err != nil {
			t.Fatal(err)
		}
	}

	select {
	case <-advanced:
	case <-time.After(2 * time.Second):
		t.Fatal("expected onPlanningTurnAdvanced callback after repeated generation errors")
	}
	if gotCollab != collab.ID {
		t.Fatalf("callback collabID = %q want %q", gotCollab, collab.ID)
	}
	if gotAgent != "arch-id" {
		t.Fatalf("callback nextAgentID = %q want arch-id", gotAgent)
	}
}
