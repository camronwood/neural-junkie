package test

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// mockCollabHub implements collaboration.HubInterface for testing
type mockCollabHub struct {
	agents   map[string]*protocol.AgentInfo
	messages []*protocol.Message
}

func newMockCollabHub() *mockCollabHub {
	return &mockCollabHub{
		agents: make(map[string]*protocol.AgentInfo),
	}
}

func (h *mockCollabHub) SendMessage(msg *protocol.Message) error {
	h.messages = append(h.messages, msg)
	return nil
}

func (h *mockCollabHub) GetAgent(agentID string) (*protocol.AgentInfo, error) {
	if a, ok := h.agents[agentID]; ok {
		return a, nil
	}
	return nil, nil
}

func (h *mockCollabHub) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	var out []protocol.AgentInfo
	for _, a := range h.agents {
		out = append(out, *a)
	}
	return out, nil
}

func (h *mockCollabHub) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	return &protocol.Channel{
		ID:   uuid.New().String(),
		Name: name,
		Type: channelType,
	}
}

func (h *mockCollabHub) addAgent(id, name string, agentType protocol.AgentType, expertise []string) {
	h.agents[id] = &protocol.AgentInfo{
		ID:        id,
		Name:      name,
		Type:      agentType,
		Expertise: expertise,
		Status:    "active",
	}
}

// ── Collaboration Lifecycle Tests ────────────────────────────────────

func TestCreateCollaboration(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("rust-1", "RustExpert", protocol.AgentTypeRust, []string{"rust", "systems"})
	hub.addAgent("sec-1", "SecurityExpert", protocol.AgentTypeSecurity, []string{"security", "auth"})

	cm := collaboration.NewCollaborationManager(hub)

	collab, err := cm.CreateCollaboration(
		"Build a CLI encryption tool",
		[]string{"rust-1", "sec-1"},
		"general",
		"testuser",
		collaboration.DiscussionConfig{},
	)

	if err != nil {
		t.Fatalf("CreateCollaboration failed: %v", err)
	}
	if collab.Phase != collaboration.PhasePlanning {
		t.Errorf("expected phase planning, got %s", collab.Phase)
	}
	if len(collab.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(collab.Agents))
	}
	if collab.Discussion == nil {
		t.Fatal("expected discussion session to be created")
	}
	if collab.Plan == nil {
		t.Fatal("expected plan artifact to be created")
	}
	if collab.Discussion.MaxRounds != collaboration.DefaultMaxRounds {
		t.Errorf("expected default max rounds %d, got %d", collaboration.DefaultMaxRounds, collab.Discussion.MaxRounds)
	}
}

func TestCreateCollaborationRequiresMinAgents(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("rust-1", "RustExpert", protocol.AgentTypeRust, nil)

	cm := collaboration.NewCollaborationManager(hub)
	_, err := cm.CreateCollaboration("test", []string{"rust-1"}, "general", "user", collaboration.DiscussionConfig{})
	if err == nil {
		t.Fatal("expected error for single agent collaboration")
	}
}

func TestMaxConcurrentCollaborations(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)
	hub.addAgent("a3", "Agent3", protocol.AgentTypeDevOps, nil)
	hub.addAgent("a4", "Agent4", protocol.AgentTypeSecurity, nil)

	cm := collaboration.NewCollaborationManager(hub)

	for i := 0; i < collaboration.MaxConcurrentCollaborations; i++ {
		_, err := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
		if err != nil {
			t.Fatalf("collaboration %d creation failed: %v", i, err)
		}
	}

	// One more should fail
	_, err := cm.CreateCollaboration("test", []string{"a3", "a4"}, "general", "user", collaboration.DiscussionConfig{})
	if err == nil {
		t.Fatal("expected error when exceeding max concurrent collaborations")
	}
}

func TestDiscussionBudgetExhaustionMovesPlanningToReviewing(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:        1,
		TurnBudget:       1,
		MaxTotalMessages: 1,
	})
	if err != nil {
		t.Fatalf("CreateCollaboration failed: %v", err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1", Type: protocol.AgentTypeBackend},
		"initial proposal",
	)
	if err := cm.RecordMessage(collab.ID, msg); err != nil {
		t.Fatalf("RecordMessage failed: %v", err)
	}

	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration failed: %v", err)
	}
	if got.Phase != collaboration.PhaseReviewing {
		t.Fatalf("expected phase reviewing after budget exhaustion, got %s", got.Phase)
	}
}

func TestConsensusConvergenceMovesPlanningToReviewing(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration failed: %v", err)
	}

	msg1 := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1", Type: protocol.AgentTypeBackend},
		"I agree with this plan",
	)
	cm.AnalyzeConsensus(collab.ID, msg1)

	msg2 := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"general",
		protocol.AgentInfo{ID: "a2", Name: "Agent2", Type: protocol.AgentTypeFrontend},
		"Looks good, I agree",
	)
	cm.AnalyzeConsensus(collab.ID, msg2)

	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration failed: %v", err)
	}
	if got.Phase != collaboration.PhaseReviewing {
		t.Fatalf("expected phase reviewing after convergence, got %s", got.Phase)
	}
	if got.Discussion == nil || got.Discussion.Status != collaboration.DiscussionConverged {
		t.Fatalf("expected discussion status converged, got %+v", got.Discussion)
	}
}

func TestTransitionToExecutingAutoCancelsOtherExecutingInSameChannel(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)

	first, err := cm.CreateCollaboration("first", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration first: %v", err)
	}
	if _, err := cm.TransitionToReviewing(first.ID); err != nil {
		t.Fatalf("TransitionToReviewing first: %v", err)
	}
	if _, err := cm.ApprovePlan(first.ID); err != nil {
		t.Fatalf("ApprovePlan first: %v", err)
	}
	if _, err := cm.TransitionToExecuting(first.ID); err != nil {
		t.Fatalf("TransitionToExecuting first: %v", err)
	}

	second, err := cm.CreateCollaboration("second", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration second: %v", err)
	}
	if _, err := cm.TransitionToReviewing(second.ID); err != nil {
		t.Fatalf("TransitionToReviewing second: %v", err)
	}
	if _, err := cm.ApprovePlan(second.ID); err != nil {
		t.Fatalf("ApprovePlan second: %v", err)
	}
	if _, err := cm.TransitionToExecuting(second.ID); err != nil {
		t.Fatalf("TransitionToExecuting second: %v", err)
	}

	gotFirst, err := cm.GetCollaboration(first.ID)
	if err != nil {
		t.Fatalf("GetCollaboration first: %v", err)
	}
	if gotFirst.Phase != collaboration.PhaseCancelled {
		t.Fatalf("expected first collaboration cancelled, got phase %s", gotFirst.Phase)
	}
	if gotFirst.Discussion == nil || gotFirst.Discussion.Status != collaboration.DiscussionCancelled {
		t.Fatalf("expected first discussion cancelled, got %+v", gotFirst.Discussion)
	}

	gotSecond, err := cm.GetCollaboration(second.ID)
	if err != nil {
		t.Fatalf("GetCollaboration second: %v", err)
	}
	if gotSecond.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected second collaboration executing, got %s", gotSecond.Phase)
	}
}

func TestApprovePlanIdempotentWhenAlreadyApproved(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("x", []string{"a1", "a2"}, "general", "u", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("first ApprovePlan: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("second ApprovePlan (idempotent): %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}
	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration: %v", err)
	}
	if got.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected executing, got %s", got.Phase)
	}
}

func TestEnsureExecutionTasksCreatesDefaultTasks(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, "general", "u", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}
	if _, err := cm.EnsureExecutionTasks(collab.ID); err != nil {
		t.Fatalf("EnsureExecutionTasks: %v", err)
	}
	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration: %v", err)
	}
	if len(got.Tasks) != 2 {
		t.Fatalf("expected 2 default tasks, got %d", len(got.Tasks))
	}
	for _, tsk := range got.Tasks {
		if tsk.AssignedTo == "" {
			t.Fatal("default task should have AssignedTo set")
		}
	}
}

func TestEnsureExecutionTasksAssignsUnassignedExtractedTasks(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, "general", "u", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}

	// Simulate tasks parsed from a plan without @mentions (all assignees empty).
	raw := []collaboration.CollaborationTask{
		{ID: "t1", Title: "Task 1", Description: "Build contract", Status: collaboration.TaskPending},
		{ID: "t2", Title: "Task 2", Description: "Shortlist", Status: collaboration.TaskPending},
		{ID: "t3", Title: "Task 3", Description: "Runbook", Status: collaboration.TaskPending},
	}
	if err := cm.SetTasks(collab.ID, raw); err != nil {
		t.Fatalf("SetTasks: %v", err)
	}
	if _, err := cm.EnsureExecutionTasks(collab.ID); err != nil {
		t.Fatalf("EnsureExecutionTasks: %v", err)
	}
	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration: %v", err)
	}
	if len(got.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(got.Tasks))
	}
	for i, tsk := range got.Tasks {
		if tsk.AssignedTo == "" {
			t.Fatalf("task %d should have AssignedTo after EnsureExecutionTasks", i)
		}
		if tsk.AssignedName == "" {
			t.Fatalf("task %d should have AssignedName after EnsureExecutionTasks", i)
		}
	}
	// Round-robin on unassigned-only order: a1, a2, a1
	if got.Tasks[0].AssignedTo != "a1" || got.Tasks[1].AssignedTo != "a2" || got.Tasks[2].AssignedTo != "a1" {
		t.Fatalf("unexpected round-robin assignees: %#v, %#v, %#v", got.Tasks[0].AssignedTo, got.Tasks[1].AssignedTo, got.Tasks[2].AssignedTo)
	}
}

func TestTransitionToExecutingDoesNotCancelExecutingInOtherChannel(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)

	gen, err := cm.CreateCollaboration("gen", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration general: %v", err)
	}
	if _, err := cm.TransitionToReviewing(gen.ID); err != nil {
		t.Fatalf("TransitionToReviewing general: %v", err)
	}
	if _, err := cm.ApprovePlan(gen.ID); err != nil {
		t.Fatalf("ApprovePlan general: %v", err)
	}
	if _, err := cm.TransitionToExecuting(gen.ID); err != nil {
		t.Fatalf("TransitionToExecuting general: %v", err)
	}

	other, err := cm.CreateCollaboration("other", []string{"a1", "a2"}, "project-alpha", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration other channel: %v", err)
	}
	if _, err := cm.TransitionToReviewing(other.ID); err != nil {
		t.Fatalf("TransitionToReviewing other: %v", err)
	}
	if _, err := cm.ApprovePlan(other.ID); err != nil {
		t.Fatalf("ApprovePlan other: %v", err)
	}
	if _, err := cm.TransitionToExecuting(other.ID); err != nil {
		t.Fatalf("TransitionToExecuting other: %v", err)
	}

	gotGen, err := cm.GetCollaboration(gen.ID)
	if err != nil {
		t.Fatalf("GetCollaboration general: %v", err)
	}
	if gotGen.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected general collab still executing, got %s", gotGen.Phase)
	}
}

func TestCollaborationPhaseTransitions(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	// planning -> reviewing
	_, err := cm.TransitionToReviewing(collab.ID)
	if err != nil {
		t.Fatalf("TransitionToReviewing failed: %v", err)
	}
	c, _ := cm.GetCollaboration(collab.ID)
	if c.Phase != collaboration.PhaseReviewing {
		t.Errorf("expected reviewing, got %s", c.Phase)
	}

	// reviewing -> approved -> executing
	_, err = cm.ApprovePlan(collab.ID)
	if err != nil {
		t.Fatalf("ApprovePlan failed: %v", err)
	}
	_, err = cm.TransitionToExecuting(collab.ID)
	if err != nil {
		t.Fatalf("TransitionToExecuting failed: %v", err)
	}
	c, _ = cm.GetCollaboration(collab.ID)
	if c.Phase != collaboration.PhaseExecuting {
		t.Errorf("expected executing, got %s", c.Phase)
	}

	// executing -> completed
	_, err = cm.CompleteCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("CompleteCollaboration failed: %v", err)
	}
	c, _ = cm.GetCollaboration(collab.ID)
	if c.Phase != collaboration.PhaseCompleted {
		t.Errorf("expected completed, got %s", c.Phase)
	}
}

func TestRevisePlan(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	cm.TransitionToReviewing(collab.ID)
	_, err := cm.RevisePlan(collab.ID, "add error handling")
	if err != nil {
		t.Fatalf("RevisePlan failed: %v", err)
	}

	c, _ := cm.GetCollaboration(collab.ID)
	if c.Phase != collaboration.PhasePlanning {
		t.Errorf("expected planning after revision, got %s", c.Phase)
	}
}

func TestCancelCollaboration(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	_, err := cm.CancelCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("CancelCollaboration failed: %v", err)
	}

	c, _ := cm.GetCollaboration(collab.ID)
	if c.Phase != collaboration.PhaseCancelled {
		t.Errorf("expected cancelled, got %s", c.Phase)
	}

	// Can't cancel again
	_, err = cm.CancelCollaboration(collab.ID)
	if err == nil {
		t.Fatal("expected error cancelling already-cancelled collaboration")
	}
}

// ── Discussion Budget Tests ──────────────────────────────────────────

func TestDiscussionTurnTaking(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:  2,
		TurnBudget: 1,
	})

	// Agent 1 should be able to speak first
	if !cm.IsAgentTurn(collab.ID, "a1") {
		t.Error("expected agent a1 to have turn first")
	}
	if cm.IsAgentTurn(collab.ID, "a2") {
		t.Error("expected agent a2 to NOT have turn yet")
	}

	// Agent 1 sends a message
	msg1 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1", Type: protocol.AgentTypeBackend}, "My proposal...")
	msg1.SetCollaborationID(collab.ID)
	err := cm.RecordMessage(collab.ID, msg1)
	if err != nil {
		t.Fatalf("RecordMessage failed: %v", err)
	}

	// Now Agent 2 should have turn
	if !cm.IsAgentTurn(collab.ID, "a2") {
		t.Error("expected agent a2 to have turn after a1 spoke")
	}

	// Agent 2 sends a message
	msg2 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a2", Name: "Agent2", Type: protocol.AgentTypeFrontend}, "I agree...")
	msg2.SetCollaborationID(collab.ID)
	cm.RecordMessage(collab.ID, msg2)

	// Round 2: Agent 1 should have turn again
	if !cm.IsAgentTurn(collab.ID, "a1") {
		t.Error("expected agent a1 to have turn in round 2")
	}
}

func TestDiscussionBudgetExhausted(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:        1,
		TurnBudget:       1,
		MaxTotalMessages: 2,
	})

	msg1 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1"}, "Hello")
	msg1.SetCollaborationID(collab.ID)
	cm.RecordMessage(collab.ID, msg1)

	msg2 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a2", Name: "Agent2"}, "Hi")
	msg2.SetCollaborationID(collab.ID)
	cm.RecordMessage(collab.ID, msg2)

	status, _ := cm.GetDiscussionStatus(collab.ID)
	if status != collaboration.DiscussionBudgetExhausted {
		t.Errorf("expected budget_exhausted, got %s", status)
	}
}

func TestDiscussionTimeout(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		Timeout: 1 * time.Millisecond,
	})

	// Wait for timeout
	time.Sleep(5 * time.Millisecond)

	timedOut := cm.CheckTimeout(collab.ID)
	if !timedOut {
		t.Error("expected discussion to time out")
	}

	status, _ := cm.GetDiscussionStatus(collab.ID)
	if status != collaboration.DiscussionTimedOut {
		t.Errorf("expected timed_out, got %s", status)
	}
}

func TestDiscussionMentionAllowsOutOfTurn(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:  3,
		TurnBudget: 2,
	})

	// Agent 1's turn first
	msg1 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1"}, "@Agent2 what do you think?")
	msg1.SetCollaborationID(collab.ID)
	msg1.Mentions = []string{"a2"}
	cm.RecordMessage(collab.ID, msg1)

	// Agent 2 should be allowed to respond (out of turn via @mention)
	if !cm.IsAgentTurn(collab.ID, "a2") {
		t.Error("expected agent a2 to be allowed via @mention even if not strict turn")
	}
}

// ── Consensus Detection Tests ────────────────────────────────────────

func TestConsensusAgreement(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	msg1 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1"}, "I agree with the plan, looks good!")
	msg1.SetCollaborationID(collab.ID)
	state1 := cm.AnalyzeConsensus(collab.ID, msg1)
	if state1 != collaboration.ConsensusAgrees {
		t.Errorf("expected agrees, got %s", state1)
	}

	// Not full consensus yet (a2 hasn't agreed)
	if cm.CheckFullConsensus(collab.ID) {
		t.Error("expected no full consensus yet")
	}

	msg2 := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a2", Name: "Agent2"}, "LGTM, let's proceed")
	msg2.SetCollaborationID(collab.ID)
	cm.AnalyzeConsensus(collab.ID, msg2)

	if !cm.CheckFullConsensus(collab.ID) {
		t.Error("expected full consensus after both agents agreed")
	}
}

func TestConsensusDisagreement(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1"}, "I have concerns about this approach, we should reconsider")
	msg.SetCollaborationID(collab.ID)
	state := cm.AnalyzeConsensus(collab.ID, msg)
	if state != collaboration.ConsensusDisagrees {
		t.Errorf("expected disagrees, got %s", state)
	}

	if !cm.HasDisagreement(collab.ID) {
		t.Error("expected disagreement to be detected")
	}
}

// ── Task Assignment Tests ────────────────────────────────────────────

func TestSetAndUpdateTasks(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "Build API", AssignedTo: "a1", Status: collaboration.TaskPending},
		{ID: "t2", Title: "Build UI", AssignedTo: "a2", Status: collaboration.TaskPending},
	}
	err := cm.SetTasks(collab.ID, tasks)
	if err != nil {
		t.Fatalf("SetTasks failed: %v", err)
	}

	// Update task status
	err = cm.UpdateTaskStatus(collab.ID, "t1", collaboration.TaskCompleted, "API built successfully")
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	if cm.AllTasksComplete(collab.ID) {
		t.Error("not all tasks should be complete yet")
	}

	cm.UpdateTaskStatus(collab.ID, "t2", collaboration.TaskCompleted, "UI built")
	if !cm.AllTasksComplete(collab.ID) {
		t.Error("all tasks should be complete now")
	}
}

func TestAddParticipantsToPlanningCollaboration(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)
	hub.addAgent("a3", "DevOpsPro", protocol.AgentTypeDevOps, []string{"kubernetes"})

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration failed: %v", err)
	}

	added, err := cm.AddParticipants(collab.ID, []string{"a3"})
	if err != nil {
		t.Fatalf("AddParticipants failed: %v", err)
	}
	if len(added) != 1 || added[0].AgentID != "a3" {
		t.Fatalf("expected agent a3 to be added, got %+v", added)
	}

	updated, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration failed: %v", err)
	}
	if len(updated.Agents) != 3 {
		t.Fatalf("expected 3 participants after add, got %d", len(updated.Agents))
	}
	if updated.Discussion == nil || len(updated.Discussion.Participants) != 3 {
		t.Fatalf("expected discussion participants to include added agent, got %+v", updated.Discussion)
	}
}

// ── Artifact Tests ───────────────────────────────────────────────────

func TestUpdateArtifact(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	err := cm.UpdateArtifact(collab.ID, "a1", "Agent1", "## Plan\n\n- Task 1: Build API")
	if err != nil {
		t.Fatalf("UpdateArtifact failed: %v", err)
	}

	artifact, _ := cm.GetArtifact(collab.ID)
	if artifact.Version != 1 {
		t.Errorf("expected version 1, got %d", artifact.Version)
	}
	if artifact.Content != "## Plan\n\n- Task 1: Build API" {
		t.Errorf("unexpected artifact content: %s", artifact.Content)
	}
	if len(artifact.EditHistory) != 1 {
		t.Errorf("expected 1 edit in history, got %d", len(artifact.EditHistory))
	}

	// Second edit
	cm.UpdateArtifact(collab.ID, "a2", "Agent2", "## Plan\n\n- Task 1: Build API\n- Task 2: Build UI")
	artifact, _ = cm.GetArtifact(collab.ID)
	if artifact.Version != 2 {
		t.Errorf("expected version 2, got %d", artifact.Version)
	}
}

func TestExtractPlanFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPlan bool
	}{
		{
			name:     "response with plan heading",
			input:    "Here's my thinking.\n\n## Plan\n\n- Task 1: foo\n- Task 2: bar",
			wantPlan: true,
		},
		{
			name:     "response with tasks heading",
			input:    "### Tasks\n\n- Task 1: Build it",
			wantPlan: true,
		},
		{
			name:     "response without plan",
			input:    "I think we should use AES-256-GCM for encryption.",
			wantPlan: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collaboration.ExtractPlanFromResponse(tt.input)
			if tt.wantPlan && result == "" {
				t.Error("expected plan to be extracted, got empty string")
			}
			if !tt.wantPlan && result != "" {
				t.Errorf("expected no plan, got: %s", result)
			}
		})
	}
}

func TestExtractTasksFromPlan(t *testing.T) {
	agents := []collaboration.CollaborationAgent{
		{AgentID: "rust-1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust},
		{AgentID: "sec-1", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity},
	}

	planContent := `## Plan

- Task 1: @RustExpert - Build the CLI scaffold with clap
- Task 2: @SecurityExpert - Implement AES-256-GCM encryption module
- Task 3: @RustExpert - Integrate encryption into CLI commands
`

	tasks := collaboration.ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	if tasks[0].AssignedTo != "rust-1" {
		t.Errorf("task 0 should be assigned to rust-1, got %s", tasks[0].AssignedTo)
	}
	if tasks[1].AssignedTo != "sec-1" {
		t.Errorf("task 1 should be assigned to sec-1, got %s", tasks[1].AssignedTo)
	}
}

func TestExtractTasksFromPlanSupportsKebabMentions(t *testing.T) {
	agents := []collaboration.CollaborationAgent{
		{AgentID: "a-1", AgentName: "agent-a", AgentType: protocol.AgentTypeBackend},
		{AgentID: "b-1", AgentName: "agent-b", AgentType: protocol.AgentTypeFrontend},
	}

	planContent := `## Tasks

- @agent-a: implement backend parser support
- @agent-b: add UI wiring for collaborations
`

	tasks := collaboration.ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].AssignedName != "agent-a" || tasks[1].AssignedName != "agent-b" {
		t.Fatalf("expected assignments to resolve kebab-case mentions, got %+v", tasks)
	}
}

func TestExtractTasksFromPlanSupportsHeadingWithAssignedLine(t *testing.T) {
	agents := []collaboration.CollaborationAgent{
		{AgentID: "rust-1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust},
		{AgentID: "sec-1", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity},
	}

	planContent := `## Plan

### Task 1: Build CLI command interface
- Assigned to: @RustExpert
- Acceptance: command parses args and prints help

### Task 2: Add encryption key handling
- Assigned to: @SecurityExpert
`

	tasks := collaboration.ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].AssignedTo != "rust-1" {
		t.Fatalf("expected task 1 assigned to rust-1, got %s", tasks[0].AssignedTo)
	}
	if tasks[1].AssignedTo != "sec-1" {
		t.Fatalf("expected task 2 assigned to sec-1, got %s", tasks[1].AssignedTo)
	}
}

// ── Config Normalization Tests ───────────────────────────────────────

func TestDiscussionConfigDefaults(t *testing.T) {
	cfg := collaboration.DiscussionConfig{}.Normalized()

	if cfg.MaxRounds != collaboration.DefaultMaxRounds {
		t.Errorf("expected max rounds %d, got %d", collaboration.DefaultMaxRounds, cfg.MaxRounds)
	}
	if cfg.TurnBudget != collaboration.DefaultTurnBudget {
		t.Errorf("expected turn budget %d, got %d", collaboration.DefaultTurnBudget, cfg.TurnBudget)
	}
	if cfg.MaxTotalMessages != collaboration.DefaultMaxTotalMessages {
		t.Errorf("expected max total messages %d, got %d", collaboration.DefaultMaxTotalMessages, cfg.MaxTotalMessages)
	}
	if cfg.Timeout != collaboration.DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", collaboration.DefaultTimeout, cfg.Timeout)
	}
}

func TestDiscussionConfigClamp(t *testing.T) {
	cfg := collaboration.DiscussionConfig{
		MaxRounds:        100,
		TurnBudget:       100,
		MaxTotalMessages: 1000,
		Timeout:          24 * time.Hour,
	}.Normalized()

	if cfg.MaxRounds != collaboration.HardMaxRounds {
		t.Errorf("expected max rounds clamped to %d, got %d", collaboration.HardMaxRounds, cfg.MaxRounds)
	}
	if cfg.TurnBudget != collaboration.HardMaxTurnBudget {
		t.Errorf("expected turn budget clamped to %d, got %d", collaboration.HardMaxTurnBudget, cfg.TurnBudget)
	}
	if cfg.MaxTotalMessages != collaboration.HardMaxTotalMessages {
		t.Errorf("expected max total messages clamped to %d, got %d", collaboration.HardMaxTotalMessages, cfg.MaxTotalMessages)
	}
	if cfg.Timeout != collaboration.HardMaxTimeout {
		t.Errorf("expected timeout clamped to %v, got %v", collaboration.HardMaxTimeout, cfg.Timeout)
	}
}

// ── Participant Query Tests ──────────────────────────────────────────

func TestIsParticipant(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	if !cm.IsParticipant(collab.ID, "a1") {
		t.Error("a1 should be a participant")
	}
	if cm.IsParticipant(collab.ID, "a3") {
		t.Error("a3 should NOT be a participant")
	}
}

func TestGetCollaborationForAgent(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("test", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{})

	found := cm.GetCollaborationForAgent("a1")
	if found == nil {
		t.Fatal("expected to find collaboration for a1")
	}
	if found.ID != collab.ID {
		t.Errorf("expected collab ID %s, got %s", collab.ID, found.ID)
	}

	notFound := cm.GetCollaborationForAgent("unknown")
	if notFound != nil {
		t.Error("expected nil for unknown agent")
	}
}

// ── Role Suggestion Tests ────────────────────────────────────────────

func TestSuggestRole(t *testing.T) {
	tests := []struct {
		agentType protocol.AgentType
		expected  string
	}{
		{protocol.AgentTypeCLI, "Implementation & Code Generation"},
		{protocol.AgentTypeSecurity, "Security Review & Auth Design"},
		{protocol.AgentTypeRust, "Rust Architecture & Systems Design"},
		{protocol.AgentTypeBackend, "Backend Architecture & API Design"},
		{protocol.AgentTypeFrontend, "Frontend Architecture & UI Design"},
		{protocol.AgentTypeGeneral, "General Contributor"},
	}

	for _, tt := range tests {
		role := collaboration.SuggestRole(tt.agentType, nil)
		if role != tt.expected {
			t.Errorf("SuggestRole(%s) = %q, want %q", tt.agentType, role, tt.expected)
		}
	}
}

func TestExecutionPhaseDiscussionIgnoresBudget(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:        1,
		TurnBudget:       1,
		MaxTotalMessages: 2,
	})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}

	for i := 0; i < 12; i++ {
		agentID := "a1"
		if i%2 == 1 {
			agentID = "a2"
		}
		msg := protocol.NewMessage(
			protocol.MessageTypeCollabDiscussion,
			"general",
			protocol.AgentInfo{ID: agentID, Name: "x", Type: protocol.AgentTypeBackend},
			"execution chatter",
		)
		if err := cm.RecordMessage(collab.ID, msg); err != nil {
			t.Fatalf("RecordMessage %d: %v", i, err)
		}
	}

	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("GetCollaboration: %v", err)
	}
	if got.Discussion.Status != collaboration.DiscussionActive {
		t.Fatalf("expected active discussion during execution, got %s (counts %d/%d)",
			got.Discussion.Status, got.Discussion.TotalMessageCount, got.Discussion.MaxTotalMessages)
	}
}

func TestExtendDiscussionLimitsReopensExhausted(t *testing.T) {
	hub := newMockCollabHub()
	hub.addAgent("a1", "Agent1", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "Agent2", protocol.AgentTypeFrontend, nil)

	cm := collaboration.NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, "general", "user", collaboration.DiscussionConfig{
		MaxRounds:        1,
		TurnBudget:       1,
		MaxTotalMessages: 1,
	})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1", Type: protocol.AgentTypeBackend},
		"one",
	)
	if err := cm.RecordMessage(collab.ID, msg); err != nil {
		t.Fatalf("RecordMessage: %v", err)
	}
	got, _ := cm.GetCollaboration(collab.ID)
	if got.Discussion.Status != collaboration.DiscussionBudgetExhausted {
		t.Fatalf("expected exhausted, got %s", got.Discussion.Status)
	}

	if _, err := cm.ExtendDiscussionLimits(collab.ID, 1, 3); err != nil {
		t.Fatalf("ExtendDiscussionLimits: %v", err)
	}
	got2, _ := cm.GetCollaboration(collab.ID)
	if got2.Discussion.Status != collaboration.DiscussionActive {
		t.Fatalf("expected active after extend, got %s", got2.Discussion.Status)
	}
	if got2.Discussion.MaxRounds < 2 || got2.Discussion.MaxTotalMessages < 4 {
		t.Fatalf("expected bumped caps, got rounds=%d msgs=%d", got2.Discussion.MaxRounds, got2.Discussion.MaxTotalMessages)
	}
}

// ── Protocol Metadata Tests ──────────────────────────────────────────

func TestCollaborationMetadataHelpers(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general",
		protocol.AgentInfo{ID: "a1", Name: "Agent1"}, "test")

	// Initially empty
	if msg.GetCollaborationID() != "" {
		t.Error("expected empty collaboration ID")
	}
	if msg.IsCollaborationMessage() {
		t.Error("expected not a collaboration message")
	}

	// Set collaboration ID
	msg.SetCollaborationID("collab-123")
	if msg.GetCollaborationID() != "collab-123" {
		t.Errorf("expected collab-123, got %s", msg.GetCollaborationID())
	}
	if !msg.IsCollaborationMessage() {
		t.Error("expected collaboration message after setting ID")
	}

	// Set phase
	msg.SetCollaborationPhase("planning")
	if msg.GetCollaborationPhase() != "planning" {
		t.Errorf("expected planning, got %s", msg.GetCollaborationPhase())
	}

	// Set task ID
	msg.SetTaskID("task-456")
	if msg.GetTaskID() != "task-456" {
		t.Errorf("expected task-456, got %s", msg.GetTaskID())
	}

	// Set artifact action
	msg.SetArtifactAction("propose")
	if msg.GetArtifactAction() != "propose" {
		t.Errorf("expected propose, got %s", msg.GetArtifactAction())
	}
}
