package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func setupRunbookAPITest(t *testing.T) *hub.Hub {
	t.Helper()
	chatHub = hub.NewHub()
	chatHub.CreateChannel("general", "General", "")
	a1 := &protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SecurityExpert", Type: protocol.AgentTypeSecurity, Status: "active"}
	if err := chatHub.RegisterAgent(a1); err != nil {
		t.Fatal(err)
	}
	if err := chatHub.RegisterAgent(a2); err != nil {
		t.Fatal(err)
	}
	return chatHub
}

func TestHandleRunbooksCreateAndSubmit(t *testing.T) {
	setupRunbookAPITest(t)

	body := map[string]any{
		"description": "API runbook",
		"agent_ids":   []string{"a1", "a2"},
		"channel":     "general",
		"created_by":  "api-tester",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleRunbooksRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		CollaborationID string `json:"collaboration_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.CollaborationID == "" {
		t.Fatal("missing collaboration_id")
	}

	now := time.Now()
	tasks := []collaboration.CollaborationTask{
		{ID: "t1", Title: "One", Description: "first", AssignedTo: "a1", AssignedName: "RustExpert", Status: collaboration.TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Two", Description: "second", AssignedTo: "a2", AssignedName: "SecurityExpert", Status: collaboration.TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	}
	putBody, _ := json.Marshal(map[string]any{"tasks": tasks})
	putReq := httptest.NewRequest(http.MethodPut, "/api/runbooks/"+created.CollaborationID, bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	handleRunbooksRoute(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status %d: %s", putRec.Code, putRec.Body.String())
	}

	subReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/submit", nil)
	subRec := httptest.NewRecorder()
	handleRunbooksRoute(subRec, subReq)
	if subRec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", subRec.Code, subRec.Body.String())
	}
	var snap collaboration.Collaboration
	if err := json.Unmarshal(subRec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Phase != collaboration.PhaseReviewing {
		t.Fatalf("phase = %s", snap.Phase)
	}
}

func TestHandleRunbookParsePlan(t *testing.T) {
	setupRunbookAPITest(t)

	createBody, _ := json.Marshal(map[string]any{
		"description": "parse",
		"agent_ids":   []string{"a1", "a2"},
		"channel":     "general",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", bytes.NewReader(createBody))
	rec := httptest.NewRecorder()
	handleRunbooksRoute(rec, req)
	var created struct {
		CollaborationID string `json:"collaboration_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	md := "## Plan\n- Task 1: @RustExpert - Build\n- Task 2: @SecurityExpert - Review\n  - depends: 1\n"
	parseBody, _ := json.Marshal(map[string]string{"markdown": md})
	parseReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/parse-plan", bytes.NewReader(parseBody))
	parseRec := httptest.NewRecorder()
	handleRunbooksRoute(parseRec, parseReq)
	if parseRec.Code != http.StatusOK {
		t.Fatalf("parse-plan status %d: %s", parseRec.Code, parseRec.Body.String())
	}
	var out struct {
		Tasks []collaboration.CollaborationTask `json:"tasks"`
	}
	if err := json.Unmarshal(parseRec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Tasks) != 2 || len(out.Tasks[1].Dependencies) != 1 {
		t.Fatalf("tasks: %#v", out.Tasks)
	}
}

func TestHandleRunbookSuggestAssignee(t *testing.T) {
	setupRunbookAPITest(t)

	createBody, _ := json.Marshal(map[string]any{
		"description": "suggest",
		"agent_ids":   []string{"a1", "a2"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/runbooks", bytes.NewReader(createBody))
	rec := httptest.NewRecorder()
	handleRunbooksRoute(rec, req)
	var created struct {
		CollaborationID string `json:"collaboration_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	suggestBody, _ := json.Marshal(map[string]string{
		"title":       "Security review",
		"description": "Audit JWT handling and OAuth flows",
	})
	suggestReq := httptest.NewRequest(http.MethodPost, "/api/runbooks/"+created.CollaborationID+"/suggest-assignee", bytes.NewReader(suggestBody))
	suggestRec := httptest.NewRecorder()
	handleRunbooksRoute(suggestRec, suggestReq)
	if suggestRec.Code != http.StatusOK {
		t.Fatalf("suggest status %d: %s", suggestRec.Code, suggestRec.Body.String())
	}
	var s struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.Unmarshal(suggestRec.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.AgentID != "a2" {
		t.Fatalf("expected security expert a2, got %q", s.AgentID)
	}
}
