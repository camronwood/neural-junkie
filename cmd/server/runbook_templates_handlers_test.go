package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleRunbookTemplatesListAndInstantiate(t *testing.T) {
	h := setupRunbookAPITest(t)
	dir := t.TempDir()
	tplDir := filepath.Join(dir, "runbook-templates")
	if err := os.MkdirAll(tplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := `{"name":"health-check-alert","title":"Health","description":"check","tasks":[{"id":"t1","title":"Ping","kind":"action","action":{"type":"web_search","config":{"query":"up"}}}]}`
	if err := os.WriteFile(filepath.Join(tplDir, "health-check-alert.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	h.SetCollaborationAssetsRootResolver(func() string { return dir })

	listReq := httptest.NewRequest(http.MethodGet, "/api/runbook-templates", nil)
	listRec := httptest.NewRecorder()
	handleRunbookTemplatesRoute(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", listRec.Code, listRec.Body.String())
	}
	var templates []map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &templates); err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("expected at least one bundled template")
	}

	body, _ := json.Marshal(map[string]any{
		"channel":     "general",
		"created_by":  "tester",
		"agent_ids":   []string{"a1", "a2"},
	})
	instReq := httptest.NewRequest(http.MethodPost, "/api/runbook-templates/health-check-alert/instantiate", bytes.NewReader(body))
	instRec := httptest.NewRecorder()
	handleRunbookTemplatesRoute(instRec, instReq)
	if instRec.Code != http.StatusOK {
		t.Fatalf("instantiate status %d: %s", instRec.Code, instRec.Body.String())
	}
	var out struct {
		CollaborationID string `json:"collaboration_id"`
	}
	if err := json.Unmarshal(instRec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.CollaborationID == "" {
		t.Fatal("missing collaboration_id from template instantiate")
	}
}
