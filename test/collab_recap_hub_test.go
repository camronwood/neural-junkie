package test

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHubEnterReviewingBlocksApproveUntilRecap(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)
	cm := h.GetCollaborationManager()

	collab, err := cm.CreateCollaboration("recap gate", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err == nil {
		t.Fatal("expected ApprovePlan blocked while planning recap pending")
	}
	ensurePlanningRecapComplete(t, cm, collab.ID)
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("ApprovePlan after recap: %v", err)
	}
}

func TestHubSlashApprovePlanAfterPlanningRecap(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("slash recap", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	ensurePlanningRecapComplete(t, cm, collab.ID)

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", humanTester(), "/approve-plan "+collab.ID[:8])
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("approve-plan: %v", err)
	}
	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.Phase != collaboration.PhaseExecuting && snap.Phase != collaboration.PhaseApproved {
		t.Fatalf("expected approved or executing after slash approve, got %s", snap.Phase)
	}
}

func TestHubCompleteCollabRequiresFinalRecap(t *testing.T) {
	h := hub.NewHub()
	collabID, ch := setupExecutingCollabWithTasks(t, h, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Work", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskCompleted},
	})
	cm := h.GetCollaborationManager()

	completeMsg := protocol.NewMessage(protocol.MessageTypeQuestion, ch, humanTester(), "/complete-collab "+collabID[:8])
	if err := h.SendMessage(completeMsg); err != nil {
		t.Fatalf("complete-collab: %v", err)
	}

	snap, _ := cm.GetCollaborationSnapshot(collabID)
	if snap.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected executing until final recap, got %s", snap.Phase)
	}

	finishExecutingCollabViaRecap(t, h, collabID)

	snap, _ = cm.GetCollaborationSnapshot(collabID)
	if snap.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed, got %s", snap.Phase)
	}
	if !strings.Contains(snap.SessionRecap, "Final session summary") {
		t.Fatalf("session_recap=%q", snap.SessionRecap)
	}
}

func TestHubTransitionToExecutingArchivesPlanningDiscussion(t *testing.T) {
	h := hub.NewHub()
	registerTwoCollabAgents(t, h)
	cm := h.GetCollaborationManager()

	collab, err := cm.CreateCollaboration("archive", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	discMsg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "general",
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust}, "planning line")
	discMsg.SetCollaborationID(collab.ID)
	_ = cm.RecordMessage(collab.ID, discMsg)

	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	ensurePlanningRecapComplete(t, cm, collab.ID)
	_, _ = cm.ApprovePlan(collab.ID)
	_, _ = cm.TransitionToExecuting(collab.ID)

	snap, _ := cm.GetCollaborationSnapshot(collab.ID)
	if snap.PlanningDiscussion == nil || len(snap.PlanningDiscussion.Messages) == 0 {
		t.Fatal("expected planning discussion archived")
	}
	if !strings.Contains(snap.PlanningDiscussion.Messages[0].Content, "planning line") {
		t.Fatalf("archived content=%q", snap.PlanningDiscussion.Messages[0].Content)
	}
	if snap.Discussion == nil || snap.Discussion.Topic == "" || !strings.Contains(snap.Discussion.Topic, "Execution Q&A") {
		t.Fatalf("expected fresh execution discussion, got %+v", snap.Discussion)
	}
}
