package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/runbooklibrary"
)

func setupRunbookLibraryHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(filepath.Join(home, ".neural-junkie", "runbook-library"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
}

func TestRunbookDefinitionExportImportRoundTrip(t *testing.T) {
	setupRunbookAPITest(t)
	setupRunbookLibraryHome(t)

	def := runbooklibrary.RunbookDefinition{
		Title:       "Composable smoke test",
		Description: "export/import roundtrip",
		Tasks: []collaboration.CollaborationTask{
			{ID: "t1", Title: "Step 1", AssignedTo: "a1", AssignedName: "RustExpert"},
		},
	}
	raw, _ := json.Marshal(def)
	createReq := loopbackRequest(http.MethodPost, "/api/runbook-definitions", bytes.NewReader(raw))
	createRec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status %d: %s", createRec.Code, createRec.Body.String())
	}
	var created runbooklibrary.RunbookDefinition
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected assigned ID")
	}

	exportReq := loopbackRequest(http.MethodGet, "/api/runbook-definitions/"+created.ID+"/export", nil)
	exportRec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("export status %d: %s", exportRec.Code, exportRec.Body.String())
	}
	var bundle runbooklibrary.DefinitionBundle
	if err := json.Unmarshal(exportRec.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.SchemaVersion != runbooklibrary.DefinitionBundleSchemaVersion {
		t.Fatalf("schema version = %d", bundle.SchemaVersion)
	}
	if bundle.Definition.Title != "Composable smoke test" {
		t.Fatalf("exported title = %q", bundle.Definition.Title)
	}

	bundleRaw, _ := json.Marshal(bundle)
	importReq := loopbackRequest(http.MethodPost, "/api/runbook-definitions/import", bytes.NewReader(bundleRaw))
	importRec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(importRec, importReq)
	if importRec.Code != http.StatusOK {
		t.Fatalf("import status %d: %s", importRec.Code, importRec.Body.String())
	}
	var imported runbooklibrary.RunbookDefinition
	if err := json.Unmarshal(importRec.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.ID == "" || imported.ID == created.ID {
		t.Fatalf("expected a freshly minted ID distinct from %q, got %q", created.ID, imported.ID)
	}
	if imported.Title != "Composable smoke test" {
		t.Fatalf("imported title = %q", imported.Title)
	}
	if imported.Version != 1 {
		t.Fatalf("expected v1 for fresh import, got %d", imported.Version)
	}
}

func TestRunbookDefinitionImportRejectsEmptyTasks(t *testing.T) {
	setupRunbookAPITest(t)
	setupRunbookLibraryHome(t)

	raw, _ := json.Marshal(runbooklibrary.RunbookDefinition{Title: "No tasks"})
	req := loopbackRequest(http.MethodPost, "/api/runbook-definitions/import", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRunbookRunProvenanceLinksDefinitionAndRun(t *testing.T) {
	h := setupRunbookAPITest(t)
	setupRunbookLibraryHome(t)

	def := runbooklibrary.RunbookDefinition{
		Title:       "Provenance smoke test",
		Description: "Verifies run -> definition -> events provenance linking",
		Tasks: []collaboration.CollaborationTask{
			{ID: "t1", Title: "Step 1", AssignedTo: "a1", AssignedName: "RustExpert"},
		},
	}
	raw, _ := json.Marshal(def)
	createReq := loopbackRequest(http.MethodPost, "/api/runbook-definitions", bytes.NewReader(raw))
	createRec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status %d: %s", createRec.Code, createRec.Body.String())
	}
	var created runbooklibrary.RunbookDefinition
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	instBody, _ := json.Marshal(map[string]any{
		"agent_ids":  []string{"a1"},
		"channel":    "general",
		"created_by": "api-tester",
	})
	instReq := loopbackRequest(http.MethodPost, "/api/runbook-definitions/"+created.ID+"/instantiate", bytes.NewReader(instBody))
	instRec := httptest.NewRecorder()
	handleRunbookDefinitionsRoute(instRec, instReq)
	if instRec.Code != http.StatusOK {
		t.Fatalf("instantiate status %d: %s", instRec.Code, instRec.Body.String())
	}
	var inst struct {
		CollaborationID string `json:"collaboration_id"`
	}
	if err := json.Unmarshal(instRec.Body.Bytes(), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.CollaborationID == "" {
		t.Fatal("missing collaboration_id")
	}
	// syncRunbookRunIndex is normally invoked by internal phase-transition
	// hooks; call it directly here so the run index has an entry to link
	// against without needing a full collaboration lifecycle.
	snap, err := h.GetRunbookSnapshot(inst.CollaborationID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefinitionID != created.ID {
		t.Fatalf("snapshot definition_id = %q, want %q", snap.DefinitionID, created.ID)
	}

	provReq := loopbackRequest(http.MethodGet, "/api/runbook-runs/"+inst.CollaborationID+"/provenance", nil)
	provRec := httptest.NewRecorder()
	handleRunbookRunsRoute(provRec, provReq)
	if provRec.Code != http.StatusOK {
		t.Fatalf("provenance status %d: %s", provRec.Code, provRec.Body.String())
	}
	var out struct {
		RunID      string                            `json:"run_id"`
		Definition *runbooklibrary.RunbookDefinition `json:"definition"`
	}
	if err := json.Unmarshal(provRec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.RunID != inst.CollaborationID {
		t.Fatalf("run_id = %q", out.RunID)
	}
	if out.Definition == nil || out.Definition.ID != created.ID {
		t.Fatalf("expected linked definition %q, got %+v", created.ID, out.Definition)
	}
}
