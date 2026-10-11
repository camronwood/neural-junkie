package collaboration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestTransitionToExecutingSandboxFailureKeepsApproved(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badRoot := filepath.Join(blocker, "collabs")

	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)
	cm.SetAssetsRootResolver(func() string { return badRoot })

	collab, err := cm.CreateCollaboration("fail sandbox", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatal(err)
	}

	_, err = cm.TransitionToExecuting(collab.ID)
	if err == nil {
		t.Fatal("expected sandbox mkdir failure")
	}
	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != PhaseApproved {
		t.Fatalf("phase=%s want approved after sandbox failure", got.Phase)
	}
	if got.WorkingDirectory != "" {
		t.Fatalf("WorkingDirectory should stay empty on failure, got %q", got.WorkingDirectory)
	}
}

func TestTransitionToExecutingFailureDoesNotCancelSibling(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NEURAL_JUNKIE_COLLAB_ASSETS_DIR", "")

	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)

	first, err := cm.CreateCollaboration("first", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToReviewing(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApprovePlan(first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToExecuting(first.ID); err != nil {
		t.Fatal(err)
	}

	blocker := filepath.Join(tmp, "blocker-file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cm.SetAssetsRootResolver(func() string {
		return filepath.Join(blocker, "nested")
	})

	second, err := cm.CreateCollaboration("second", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToReviewing(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApprovePlan(second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToExecuting(second.ID); err == nil {
		t.Fatal("expected second TransitionToExecuting to fail")
	}

	gotFirst, _ := cm.GetCollaboration(first.ID)
	gotSecond, _ := cm.GetCollaboration(second.ID)
	if gotFirst.Phase != PhaseExecuting {
		t.Fatalf("sibling should remain executing, got %s", gotFirst.Phase)
	}
	if gotSecond.Phase != PhaseApproved {
		t.Fatalf("failed transition should stay approved, got %s", gotSecond.Phase)
	}
}

func TestRevisePlanClearsPlanningRecap(t *testing.T) {
	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)

	collab, err := cm.CreateCollaboration("revise", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatal(err)
	}
	if err := cm.CompletePlanningRecap(collab.ID, "a1", "prior recap"); err != nil {
		t.Fatal(err)
	}
	before, _ := cm.GetCollaboration(collab.ID)
	if before.PlanningRecapStatus != RecapStatusComplete || before.PlanningRecap == "" {
		t.Fatalf("setup: recap status=%q text=%q", before.PlanningRecapStatus, before.PlanningRecap)
	}

	if _, err := cm.RevisePlan(collab.ID, "add tests"); err != nil {
		t.Fatal(err)
	}
	after, _ := cm.GetCollaboration(collab.ID)
	if after.Phase != PhasePlanning {
		t.Fatalf("phase=%s", after.Phase)
	}
	if after.PlanningRecap != "" || after.PlanningRecapStatus != "" || after.PlanningRecapAgentID != "" {
		t.Fatalf("recap fields not cleared: status=%q agent=%q text=%q",
			after.PlanningRecapStatus, after.PlanningRecapAgentID, after.PlanningRecap)
	}
}

func TestRevisePlanWrongPhase(t *testing.T) {
	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("planning", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if _, err := cm.RevisePlan(collab.ID, "x"); err == nil {
		t.Fatal("expected error revising while still planning")
	}
}

func TestListActiveAndGetCollaborationReturnClones(t *testing.T) {
	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("clone", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}

	listed := cm.ListActive()
	if len(listed) != 1 {
		t.Fatalf("len=%d", len(listed))
	}
	listed[0].Phase = PhaseCancelled
	listed[0].Title = "mutated"

	got, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase == PhaseCancelled || got.Title == "mutated" {
		t.Fatal("ListActive must return clones; mutation leaked into manager state")
	}

	forAgent := cm.GetCollaborationForAgent("a1")
	if forAgent == nil {
		t.Fatal("expected collab for a1")
	}
	forAgent.Phase = PhaseCompleted
	again, _ := cm.GetCollaboration(collab.ID)
	if again.Phase == PhaseCompleted {
		t.Fatal("GetCollaborationForAgent must return clones")
	}
}

func TestListActiveCloneRaceWithUpdateTaskStatus(t *testing.T) {
	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)
	collab, err := cm.CreateCollaboration("race", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_ = cm.SetTasks(collab.ID, []CollaborationTask{
		{ID: "t1", Title: "T", AssignedTo: "a1", AssignedName: "A", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				_ = cm.ListActive()
				_, _ = cm.GetCollaboration(collab.ID)
				_ = cm.GetCollaborationForAgent("a1")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 40; j++ {
			status := TaskInProgress
			if j%2 == 0 {
				status = TaskPending
			}
			_ = cm.UpdateTaskStatus(collab.ID, "t1", status, "")
		}
	}()
	wg.Wait()
}

func TestExecutionMessageBudgetExceededCheckOnly(t *testing.T) {
	hub := newRunbookMockHub()
	hub.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	hub.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(hub)
	collab, _ := cm.CreateCollaboration("budget", []string{"a1", "a2"}, "general", "user", DiscussionConfig{})
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatal(err)
	}

	limit := collab.MaxExecutionMessagesLimit()
	for i := 0; i < limit; i++ {
		if _, _, err := cm.IncrementExecutionMessageCount(collab.ID); err != nil {
			t.Fatal(err)
		}
	}
	over, err := cm.ExecutionMessageBudgetExceeded(collab.ID)
	if err != nil || !over {
		t.Fatalf("over=%v err=%v", over, err)
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.ExecutionMessageCount != limit {
		t.Fatalf("check-only must not increment; count=%d want %d", snap.ExecutionMessageCount, limit)
	}
}
