package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/plans"
)

const testPlanMarkdown = `---
name: HelloWorld plan
overview: Add a HelloWorld helper.
todos:
  - id: add-fn
    content: Add HelloWorld in core/sample/main.go
    status: pending
isProject: false
---

# HelloWorld

## Out of scope

- Tests
`

func TestHandlePlansGetPut(t *testing.T) {
	restore := plans.OverrideForTest(plans.NewStore(t.TempDir()))
	defer restore()
	hubSessions = hub.NewSessionManager()
	sess := hubSessions.CreateSession("Camron", "admin")

	recIn, err := plans.Active().SaveFromMarkdown(testPlanMarkdown)
	if err != nil || recIn == nil {
		t.Fatalf("seed: %v %+v", err, recIn)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plans", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec := httptest.NewRecorder()
	handlePlans(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET list status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/plans/"+recIn.ID, nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec = httptest.NewRecorder()
	handlePlansSubRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET one status %d: %s", rec.Code, rec.Body.String())
	}

	updated := bytes.ReplaceAll([]byte(testPlanMarkdown), []byte("status: pending"), []byte("status: completed"))
	body, _ := json.Marshal(map[string]string{"markdown": string(updated)})
	req = httptest.NewRequest(http.MethodPut, "/api/plans/"+recIn.ID, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-NJ-Session", sess.Token)
	rec = httptest.NewRecorder()
	handlePlansSubRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d: %s", rec.Code, rec.Body.String())
	}
	got, err := plans.Active().Get(recIn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Todos[0].Status != "completed" {
		t.Fatalf("status=%q", got.Todos[0].Status)
	}
}
