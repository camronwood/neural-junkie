package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

func createExecutingRunbookForCollabAPI(t *testing.T) (collabID, taskID string) {
	t.Helper()
	setupRunbookAPITest(t)
	body, _ := json.Marshal(map[string]any{
		"description": "collab API test",
		"agent_ids":   []string{"a1", "a2"},
		"channel":     "general",
		"created_by":  "tester",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleRunbooksRoute(rec, req)
	var created struct {
		CollaborationID string `json:"collaboration_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "First", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Second", AssignedTo: "a2", AssignedName: "SecurityExpert", Status: collaboration.TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	}
	putBody, _ := json.Marshal(map[string]any{"tasks": tasks})
	putReq := httptest.NewRequest(http.MethodPut, "/api/runbooks/"+created.CollaborationID, bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handleRunbooksRoute(putRec, putReq)

	subReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/submit", nil)
	subRec := httptest.NewRecorder()
	handleRunbooksRoute(subRec, subReq)
	if subRec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", subRec.Code, subRec.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/start", nil)
	startRec := httptest.NewRecorder()
	handleRunbooksRoute(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status %d: %s", startRec.Code, startRec.Body.String())
	}
	if err := chatHub.AcknowledgeCollaborationWorkspace(created.CollaborationID, ""); err != nil {
		t.Fatalf("AcknowledgeCollaborationWorkspace: %v", err)
	}
	return created.CollaborationID, "t1"
}

func TestHandleCollabTaskCompleteDispatchesWave(t *testing.T) {
	collabID, taskID := createExecutingRunbookForCollabAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+collabID+"/tasks/"+taskID+"/complete", nil)
	rec := httptest.NewRecorder()
	handleCollaborationsSubRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete status %d: %s", rec.Code, rec.Body.String())
	}

	var snap collaboration.Collaboration
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	var t1, t2 *collaboration.CollaborationTask
	for i := range snap.Tasks {
		switch snap.Tasks[i].ID {
		case "t1":
			t1 = &snap.Tasks[i]
		case "t2":
			t2 = &snap.Tasks[i]
		}
	}
	if t1 == nil || t1.Status != collaboration.TaskCompleted {
		t.Fatalf("t1 status = %v", t1)
	}
	if t2 == nil || !t2.PromptDispatched {
		t.Fatal("t2 should be dispatched after t1 complete via API")
	}
}

func TestHandleCollabTaskSkipDispatchesWave(t *testing.T) {
	collabID, taskID := createExecutingRunbookForCollabAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+collabID+"/tasks/"+taskID+"/skip", nil)
	rec := httptest.NewRecorder()
	handleCollaborationsSubRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("skip status %d: %s", rec.Code, rec.Body.String())
	}
	var snap collaboration.Collaboration
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	var t1, t2 *collaboration.CollaborationTask
	for i := range snap.Tasks {
		switch snap.Tasks[i].ID {
		case "t1":
			t1 = &snap.Tasks[i]
		case "t2":
			t2 = &snap.Tasks[i]
		}
	}
	if t1 == nil || t1.Status != collaboration.TaskCompleted {
		t.Fatalf("t1 should be completed after skip, got %#v", t1)
	}
	if t2 == nil || !t2.PromptDispatched {
		t.Fatal("t2 should dispatch after skip")
	}
}

func TestHandleCollabTaskReassign(t *testing.T) {
	collabID, taskID := createExecutingRunbookForCollabAPI(t)

	body, _ := json.Marshal(map[string]string{"agent_id": "a2"})
	req := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+collabID+"/tasks/"+taskID+"/reassign", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleCollaborationsSubRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reassign status %d: %s", rec.Code, rec.Body.String())
	}
	var snap collaboration.Collaboration
	_ = json.Unmarshal(rec.Body.Bytes(), &snap)
	for _, task := range snap.Tasks {
		if task.ID == taskID && task.AssignedTo != "a2" {
			t.Fatalf("task assignee = %q", task.AssignedTo)
		}
	}
}

func TestHandleCollabTaskApproveDispatchesGatedTask(t *testing.T) {
	setupRunbookAPITest(t)
	body, _ := json.Marshal(map[string]any{
		"description": "gated",
		"agent_ids":   []string{"a1"},
		"channel":     "general",
		"created_by":  "tester",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleRunbooksRoute(rec, req)
	var created struct {
		CollaborationID string `json:"collaboration_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{
			ID: "t1", Title: "Gated", AssignedTo: "a1", AssignedName: "RustExpert",
			Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now,
			Options: &collaboration.TaskExecutionOptions{RequiresApproval: true},
		},
	}
	putBody, _ := json.Marshal(map[string]any{"tasks": tasks})
	putReq := httptest.NewRequest(http.MethodPut, "/api/runbooks/"+created.CollaborationID, bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handleRunbooksRoute(putRec, putReq)

	subReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/submit", nil)
	handleRunbooksRoute(httptest.NewRecorder(), subReq)
	startReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/start", nil)
	startRec := httptest.NewRecorder()
	handleRunbooksRoute(startRec, startReq)
	_ = chatHub.AcknowledgeCollaborationWorkspace(created.CollaborationID, "")

	snap, _ := chatHub.GetRunbookSnapshot(created.CollaborationID)
	if len(snap.Tasks) != 1 || !snap.Tasks[0].AwaitingApproval {
		t.Fatalf("gated task should require approval: %#v", snap.Tasks[0])
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+created.CollaborationID+"/tasks/t1/approve", nil)
	approveRec := httptest.NewRecorder()
	handleCollaborationsSubRoute(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve status %d: %s", approveRec.Code, approveRec.Body.String())
	}
	var approved collaboration.Collaboration
	_ = json.Unmarshal(approveRec.Body.Bytes(), &approved)
	if len(approved.Tasks) != 1 || approved.Tasks[0].AwaitingApproval {
		t.Fatalf("approve should clear awaiting_approval: %#v", approved.Tasks[0])
	}
}

func TestHandleCollabPauseResume(t *testing.T) {
	collabID, _ := createExecutingRunbookForCollabAPI(t)

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+collabID+"/pause", nil)
	pauseRec := httptest.NewRecorder()
	handleCollaborationsSubRoute(pauseRec, pauseReq)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("pause: %d %s", pauseRec.Code, pauseRec.Body.String())
	}
	var paused collaboration.Collaboration
	_ = json.Unmarshal(pauseRec.Body.Bytes(), &paused)
	if !paused.DispatchPaused {
		t.Fatal("expected dispatch_paused")
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/collaborations/"+collabID+"/resume", nil)
	resumeRec := httptest.NewRecorder()
	handleCollaborationsSubRoute(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("resume: %d", resumeRec.Code)
	}
	var resumed collaboration.Collaboration
	_ = json.Unmarshal(resumeRec.Body.Bytes(), &resumed)
	if resumed.DispatchPaused {
		t.Fatal("expected dispatch resumed")
	}
}
