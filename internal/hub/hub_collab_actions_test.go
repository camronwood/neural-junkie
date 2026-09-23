package hub

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/collaboration/actions"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMaxConcurrentTasksCapsDispatchWave(t *testing.T) {
	h := newTestHub(t)
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
	if _, err := h.StartRunbook(result.CollaborationID, nil); err != nil {
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
	h := newTestHub(t)
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")
	h.SetCollabActionRunnerConfig(actions.Config{
		WebSearchQuery: func(_ context.Context, q string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{{"title": "ok", "query": q}}, nil
		},
	})

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
	if _, err := h.StartRunbook(result.CollaborationID, nil); err != nil {
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

func TestApproveGatedWebhookExecutesAction(t *testing.T) {
	h := newTestHub(t)
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")

	var posts int
	h.SetCollabActionRunnerConfig(actions.Config{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			posts++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     make(http.Header),
			}, nil
		})},
	})

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "wh1", Title: "Notify", Kind: collaboration.TaskKindAction, Status: collaboration.TaskPending,
			Action: &collaboration.TaskActionSpec{
				Type:   "webhook",
				Config: map[string]interface{}{"url": "https://example.com/hook", "payload": map[string]interface{}{"ping": true}},
			},
			CreatedAt: now, UpdatedAt: now,
		},
	}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "gated webhook",
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
	if _, err := h.StartRunbook(result.CollaborationID, nil); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := h.AcknowledgeCollaborationWorkspace(result.CollaborationID, ""); err != nil {
		t.Fatalf("ack: %v", err)
	}

	cm := h.GetCollaborationManager()
	snap, _ := cm.GetCollaborationSnapshot(result.CollaborationID)
	var wh *collaboration.CollaborationTask
	for i := range snap.Tasks {
		if snap.Tasks[i].ID == "wh1" {
			wh = &snap.Tasks[i]
			break
		}
	}
	if wh == nil || !wh.AwaitingApproval || wh.Status != collaboration.TaskInProgress {
		t.Fatalf("expected gated in_progress, got %#v", wh)
	}
	if posts != 0 {
		t.Fatalf("webhook should not POST before approve, posts=%d", posts)
	}

	if _, err := cm.ApproveTaskDispatch(result.CollaborationID, "wh1"); err != nil {
		t.Fatal(err)
	}
	h.AfterCollabTaskApproved(result.CollaborationID, "wh1")

	snap2, _ := cm.GetCollaborationSnapshot(result.CollaborationID)
	for i := range snap2.Tasks {
		if snap2.Tasks[i].ID == "wh1" {
			wh = &snap2.Tasks[i]
			break
		}
	}
	if wh == nil || wh.Status != collaboration.TaskCompleted {
		t.Fatalf("expected completed after approve, got %#v", wh)
	}
	if posts != 1 {
		t.Fatalf("expected 1 POST after approve, got %d", posts)
	}

	// Second AfterCollabTaskApproved must not re-gate forever / double-post when already complete.
	h.AfterCollabTaskApproved(result.CollaborationID, "wh1")
	if posts != 1 {
		t.Fatalf("completed task should not POST again, posts=%d", posts)
	}
}

func TestApproveGatedEmailExecutesAction(t *testing.T) {
	h := newTestHub(t)
	chName := "general"
	_ = h.CreateChannel(chName, "General", "")

	var sent int
	h.SetCollabActionRunnerConfig(actions.Config{
		EmailSend: func(_ context.Context, msg actions.EmailMessage) error {
			sent++
			if msg.To != "camronwood@gmail.com" {
				t.Fatalf("to = %q", msg.To)
			}
			return nil
		},
	})

	a1 := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "em1", Title: "Email Camron", Kind: collaboration.TaskKindAction, Status: collaboration.TaskPending,
			Action: &collaboration.TaskActionSpec{Type: "email", Config: map[string]interface{}{
				"to": "camronwood@gmail.com", "subject": "test", "body": "hi",
				"host": "smtp.example.com", "from": "nj@example.com",
			}},
			CreatedAt: now, UpdatedAt: now,
		},
	}
	result, err := h.CreateRunbookSession(RunbookCreateRequest{
		Description: "gated email",
		AgentIDs:    []string{"a1"},
		Channel:     chName,
		CreatedBy:   "tester",
		Tasks:       tasks,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.StartRunbook(result.CollaborationID, nil); err != nil {
		t.Fatal(err)
	}
	_ = h.AcknowledgeCollaborationWorkspace(result.CollaborationID, "")

	cm := h.GetCollaborationManager()
	snap, _ := cm.GetCollaborationSnapshot(result.CollaborationID)
	for _, task := range snap.Tasks {
		if task.ID == "em1" && !task.AwaitingApproval {
			t.Fatal("email should gate before approve")
		}
	}
	if sent != 0 {
		t.Fatal("must not send before approve")
	}
	if _, err := cm.ApproveTaskDispatch(result.CollaborationID, "em1"); err != nil {
		t.Fatal(err)
	}
	h.AfterCollabTaskApproved(result.CollaborationID, "em1")
	if sent != 1 {
		t.Fatalf("sent=%d", sent)
	}
	snap2, _ := cm.GetCollaborationSnapshot(result.CollaborationID)
	for _, task := range snap2.Tasks {
		if task.ID == "em1" && task.Status != collaboration.TaskCompleted {
			t.Fatalf("status=%s", task.Status)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
