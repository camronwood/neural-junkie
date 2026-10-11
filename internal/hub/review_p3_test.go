package hub

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClaimTaskPromptDispatchClearsOnSendFailure(t *testing.T) {
	h := NewHub()
	chName := "test-claim-clear"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("claim clear", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatal(err)
	}
	completePlanningRecapForHubTest(t, cm, collab.ID)
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "task-1", Title: "Work", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
	})
	if _, _, err := cm.AcknowledgeWorkspace(collab.ID); err != nil {
		t.Fatal(err)
	}

	// Remove channel so SendMessage fails after claim.
	h.mu.Lock()
	delete(h.channels, chName)
	h.mu.Unlock()

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	sent := h.dispatchCollabTaskMessagesFilter(snap, nil, nil, false)
	if sent != 0 {
		t.Fatalf("sent=%d want 0", sent)
	}
	after, _ := cm.GetCollaborationSnapshot(collab.ID)
	if after.Tasks[0].PromptDispatched {
		t.Fatal("claim should be cleared after send failure")
	}
	if after.Tasks[0].Status != collaboration.TaskPending {
		t.Fatalf("status=%s want pending after clear", after.Tasks[0].Status)
	}
}

func TestClaimTaskPromptDispatchSingleFlightPreventsDuplicateSend(t *testing.T) {
	h := NewHub()
	chName := "test-claim-dup"
	_ = h.CreateChannel(chName, "collab", "test")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("claim dup", []string{"a1", "a2"}, chName, "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)
	now := time.Now()
	_ = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{ID: "task-dup", Title: "Work", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
	})
	if _, _, err := cm.AcknowledgeWorkspace(collab.ID); err != nil {
		t.Fatal(err)
	}

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	n1 := h.dispatchCollabTaskMessagesFilter(snap, nil, nil, false)
	snap2, _ := cm.GetCollaborationSnapshot(collab.ID)
	n2 := h.dispatchCollabTaskMessagesFilter(snap2, nil, nil, false)
	if n1 != 1 {
		t.Fatalf("first dispatch sent=%d want 1", n1)
	}
	if n2 != 0 {
		t.Fatalf("second dispatch sent=%d want 0 (already claimed)", n2)
	}
}

func TestResolveCollabIDUniquePrefixAndAmbiguous(t *testing.T) {
	h := NewHub()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	cm := h.GetCollaborationManager()

	c1, err := cm.CreateCollaboration("one", []string{"a1", "a2"}, "general", "u", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}

	ch := h.commandHandler
	got, err := ch.resolveCollabID(c1.ID[:8])
	if err != nil || got != c1.ID {
		t.Fatalf("unique prefix: got=%q err=%v", got, err)
	}

	shared := "abcdef12-1111-2222-3333-444444444444"
	other := "abcdef12-9999-8888-7777-666666666666"
	now := time.Now()
	cm.Restore(map[string]*collaboration.Collaboration{
		shared: {
			ID: shared, Title: "A", Description: "a", Phase: collaboration.PhasePlanning,
			Channel: "general", CreatedAt: now, UpdatedAt: now,
			Agents: []collaboration.CollaborationAgent{{AgentID: "a1"}, {AgentID: "a2"}},
		},
		other: {
			ID: other, Title: "B", Description: "b", Phase: collaboration.PhasePlanning,
			Channel: "general", CreatedAt: now, UpdatedAt: now,
			Agents: []collaboration.CollaborationAgent{{AgentID: "a1"}, {AgentID: "a2"}},
		},
	})

	_, err = ch.resolveCollabID("abcdef12")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
	_, err = ch.resolveCollabID("missing-prefix")
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestExecutionBudgetIncrementsOnlyOnAccept(t *testing.T) {
	h := NewHub()
	chName := "budget-accept"
	_ = h.CreateChannel(chName, "collab", "test")
	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("budget", []string{"a1", "a2"}, chName, "u", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	approveAndExecuteCollabForTest(t, cm, collab.ID)

	// Rejected: unknown channel — must not increment.
	bad := protocol.NewMessage(protocol.MessageTypeChat, "no-such-channel", *a1, "hello")
	bad.SetCollaborationID(collab.ID)
	if err := h.SendMessage(bad); err == nil {
		t.Fatal("expected channel not found")
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.ExecutionMessageCount != 0 {
		t.Fatalf("rejected message incremented budget to %d", snap.ExecutionMessageCount)
	}

	okMsg := protocol.NewMessage(protocol.MessageTypeChat, chName, *a1, "hello accepted")
	okMsg.SetCollaborationID(collab.ID)
	if err := h.SendMessage(okMsg); err != nil {
		t.Fatal(err)
	}
	snap, _ = cm.GetCollaborationSnapshot(collab.ID)
	if snap.ExecutionMessageCount != 1 {
		t.Fatalf("accepted message count=%d want 1", snap.ExecutionMessageCount)
	}
}

func TestRegisterAgentIssuesWSToken(t *testing.T) {
	h := NewHub()
	a := &protocol.AgentInfo{ID: "tok-a1", Name: "TokAgent", Type: protocol.AgentTypeBackend, Status: "active"}
	token, err := h.RegisterAgentWithToken(a)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty agent token")
	}
	if !h.ValidateAgentWSToken(a.ID, token) {
		t.Fatal("token should validate")
	}
	if h.ValidateAgentWSToken(a.ID, "wrong") {
		t.Fatal("wrong token must not validate")
	}
}
