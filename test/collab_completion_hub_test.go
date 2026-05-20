package test

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func setupExecutingCollabWithTasks(t *testing.T, h *hub.Hub, tasks []collaboration.CollaborationTask) (collabID, channel string) {
	t.Helper()
	registerTwoCollabAgents(t, h)
	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"completion test",
		[]string{"a1", "a2"},
		"general",
		"tester",
		collaboration.DiscussionConfig{},
	)
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if len(tasks) > 0 {
		if err := cm.SetTasks(collab.ID, tasks); err != nil {
			t.Fatalf("SetTasks: %v", err)
		}
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	ensurePlanningRecapComplete(t, cm, collab.ID)
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("TransitionToExecuting: %v", err)
	}
	return collab.ID, collab.Channel
}

func TestHubAgentTASKSTATUSCompletesCollaboration(t *testing.T) {
	h := hub.NewHub()
	taskID := "task-aaa-bbbb"
	collabID, ch := setupExecutingCollabWithTasks(t, h, []collaboration.CollaborationTask{
		{
			ID:           taskID,
			Title:        "Build landing page",
			AssignedTo:   "a1",
			AssignedName: "RustExpert",
			Status:       collaboration.TaskPending,
		},
	})

	reply := protocol.NewMessage(
		protocol.MessageTypeAnswer,
		ch,
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"index.html is in place.\nTASK_STATUS: completed\n",
	)
	reply.SetCollaborationID(collabID)
	reply.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	reply.SetTaskID(taskID)

	if err := h.SendMessage(reply); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	cm := h.GetCollaborationManager()
	got, err := cm.GetCollaboration(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected executing while final recap pending, got %s", got.Phase)
	}
	if got.SessionRecapStatus != collaboration.RecapStatusPending {
		t.Fatalf("expected session recap pending, got %s", got.SessionRecapStatus)
	}

	finishExecutingCollabViaRecap(t, h, collabID)

	got, err = cm.GetCollaboration(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected phase completed, got %s", got.Phase)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].Status != collaboration.TaskCompleted {
		t.Fatalf("expected task completed, got %+v", got.Tasks)
	}
	if got.Discussion == nil || got.Discussion.Status != collaboration.DiscussionConverged {
		t.Fatalf("expected discussion converged, got %+v", got.Discussion)
	}
	if !strings.Contains(got.SessionRecap, "Final session summary") {
		t.Fatalf("expected session_recap stored, got %q", got.SessionRecap)
	}
}

func TestHubCollabTaskPromptDoesNotRegressTaskToPending(t *testing.T) {
	h := hub.NewHub()
	taskID := "task-ccc-dddd"
	collabID, ch := setupExecutingCollabWithTasks(t, h, []collaboration.CollaborationTask{
		{
			ID:           taskID,
			Title:        "Review",
			AssignedTo:   "a1",
			AssignedName: "RustExpert",
			Status:       collaboration.TaskInProgress,
		},
	})

	taskMsg := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		ch,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@RustExpert please complete your task",
	)
	taskMsg.SetCollaborationID(collabID)
	taskMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	taskMsg.SetTaskID(taskID)
	taskMsg.SetTaskStatus(string(collaboration.TaskPending))

	if err := h.SendMessage(taskMsg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	got, err := h.GetCollaborationManager().GetCollaboration(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tasks[0].Status != collaboration.TaskInProgress {
		t.Fatalf("collaboration_task must not update task status via lifecycle; got %s", got.Tasks[0].Status)
	}
}

func TestHubPlanHandoffSyncCompletesCollaboration(t *testing.T) {
	h := hub.NewHub()
	collabID, ch := setupExecutingCollabWithTasks(t, h, []collaboration.CollaborationTask{
		{ID: "t1", Title: "Task one", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending},
		{ID: "t2", Title: "Task two", AssignedTo: "a2", AssignedName: "SecurityExpert", Status: collaboration.TaskPending},
	})

	handoff := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		ch,
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"PLAN (V2)\n\nTask 1 (@RustExpert) — Complete\nTask 2 (@SecurityExpert) - Complete\n",
	)
	handoff.SetCollaborationID(collabID)
	handoff.SetCollaborationPhase(string(collaboration.PhaseExecuting))

	if err := h.SendMessage(handoff); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	got, err := h.GetCollaborationManager().GetCollaboration(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseExecuting {
		t.Fatalf("expected executing pending final recap, got %s", got.Phase)
	}

	finishExecutingCollabViaRecap(t, h, collabID)

	got, err = h.GetCollaborationManager().GetCollaboration(collabID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected completed after handoff sync, got %s", got.Phase)
	}
	for _, task := range got.Tasks {
		if task.Status != collaboration.TaskCompleted {
			t.Fatalf("task %s status %s", task.ID, task.Status)
		}
	}

	msgs, _ := h.GetMessages(ch, 30)
	var sawCompletion bool
	for _, m := range msgs {
		if m != nil && strings.Contains(m.Content, "Collaboration") && strings.Contains(m.Content, "completed") {
			sawCompletion = true
			break
		}
	}
	if !sawCompletion {
		t.Fatal("expected completion broadcast in channel messages")
	}
}
