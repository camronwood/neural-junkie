package hub

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaxConcurrentTasksCapsDispatchWave(t *testing.T) {
	h := NewHub()
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "a3", Name: "AgentC", Type: protocol.AgentTypeRust, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "A", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "B", AssignedTo: "a2", AssignedName: "AgentB", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t3", Title: "C", AssignedTo: "a3", AssignedName: "AgentC", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
	}
	policy := collaboration.ExecutionPolicy{MaxConcurrentTasks: 1}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "parallel cap",
		AgentIDs:    []string{"a1", "a2", "a3"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks:       tasks,
	})
	if err != nil {
		t.Fatalf("CreateRunbookSession: %v", err)
	}
	if _, err := h.UpdateRunbookSession(result.CollaborationID, collaboration.RunbookUpdatePayload{ExecutionPolicy: &policy}); err != nil {
		t.Fatalf("UpdateRunbookSession: %v", err)
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		t.Fatalf("SubmitRunbookForReview: %v", err)
	}
	if _, err := h.StartRunbook(result.CollaborationID); err != nil {
		t.Fatalf("StartRunbook: %v", err)
	}
	if err := h.AcknowledgeCollaborationWorkspace(result.CollaborationID, ""); err != nil {
		t.Fatalf("AcknowledgeCollaborationWorkspace: %v", err)
	}

	snap, _ := h.GetRunbookSnapshot(result.CollaborationID)
	dispatched := 0
	for _, task := range snap.Tasks {
		if task.PromptDispatched {
			dispatched++
		}
	}
	if dispatched != 1 {
		t.Fatalf("max_concurrent_tasks=1: expected 1 dispatched task, got %d", dispatched)
	}
}

func TestExecuteCollabActionTaskDispatchesDependent(t *testing.T) {
	h := NewHub()
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "act1", Title: "Search", Kind: collaboration.TaskKindAction, Status: collaboration.TaskPending,
			Action:    &collaboration.TaskActionSpec{Type: "web_search", Config: map[string]interface{}{"query": "health"}},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "t2", Title: "Review", AssignedTo: "a1", AssignedName: "AgentA", Status: collaboration.TaskPending,
			Dependencies: []string{"act1"}, CreatedAt: now, UpdatedAt: now,
		},
	}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "action dag",
		AgentIDs:    []string{"a1"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks:       tasks,
	})
	if err != nil {
		t.Fatalf("CreateRunbookSession: %v", err)
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := h.StartRunbook(result.CollaborationID); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := h.AcknowledgeCollaborationWorkspace(result.CollaborationID, ""); err != nil {
		t.Fatalf("AcknowledgeCollaborationWorkspace: %v", err)
	}

	cm := h.GetCollaborationManager()
	snap2, _ := cm.GetCollaborationSnapshot(result.CollaborationID)
	var act, follow *collaboration.CollaborationTask
	for i := range snap2.Tasks {
		switch snap2.Tasks[i].ID {
		case "act1":
			act = &snap2.Tasks[i]
		case "t2":
			follow = &snap2.Tasks[i]
		}
	}
	if act == nil || act.Status != collaboration.TaskCompleted {
		t.Fatalf("action task should be completed, got %#v", act)
	}
	if follow == nil || !follow.PromptDispatched {
		t.Fatal("dependent agent task should dispatch after action completes")
	}
}
