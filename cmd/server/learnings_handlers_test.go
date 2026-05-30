package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	learningpkg "github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func enablePersonalLearningForTest(t *testing.T) {
	t.Helper()
	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	if err := appConfig.InstallPack(config.PackSpecialistTuning); err != nil {
		t.Fatal(err)
	}
	appConfig.Packs.Enabled[config.PackSpecialistTuning] = true
	appConfig.Features.PersonalLearningEnabled = true
	learningStore = nil
	initPersonalLearningStore()
	if learningStore == nil {
		t.Fatal("learning store not initialized")
	}
}

func TestHandleLearningsRoute_gates(t *testing.T) {
	appConfig = config.DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "/api/learnings", nil)
	rec := httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without pack, got %d", rec.Code)
	}

	enablePersonalLearningForTest(t)
	defer func() {
		appConfig = nil
		learningStore = nil
		chatHub = nil
	}()

	req = httptest.NewRequest(http.MethodGet, "/api/learnings", nil)
	rec = httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleLearningsCRUD(t *testing.T) {
	dir := t.TempDir()
	chatHub = hub.NewHub()
	defer func() {
		chatHub = nil
		appConfig = nil
		learningStore = nil
	}()

	enablePersonalLearningForTest(t)
	storePath := filepath.Join(dir, "learnings.json")
	var err error
	learningStore, err = learningpkg.NewStore(storePath)
	if err != nil {
		t.Fatal(err)
	}

	body := map[string]string{
		"agent_id":   "agent-1",
		"agent_name": "BackendEngineer",
		"content":    "Use structured logging",
		"category":   "preference",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/learnings", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected id in response")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/learnings?agent_id=agent-1", nil)
	rec = httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(list))
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/learnings/"+id, nil)
	rec = httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE expected 200, got %d", rec.Code)
	}
}

func TestHandleLearningsStats(t *testing.T) {
	chatHub = hub.NewHub()
	defer func() {
		chatHub = nil
		appConfig = nil
		learningStore = nil
		loraTrainMgr = nil
	}()

	enablePersonalLearningForTest(t)
	initLoraTrainManager()

	agent := &protocol.AgentInfo{
		ID:   "asst-1",
		Name: "Assistant",
		Type: protocol.AgentTypeAssistant,
	}
	if err := chatHub.RegisterAgent(agent); err != nil {
		t.Fatal(err)
	}
	if err := chatHub.JoinChannel(agent.ID, "general"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/learnings/stats?agent_id=asst-1", nil)
	rec := httptest.NewRecorder()
	handleLearningsRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["agent_id"] != "asst-1" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

// TestLearningLoRASmoke is Layer 2 CI smoke: learnings API + expert-context without GPU/Python.
func TestLearningLoRASmoke(t *testing.T) {
	TestHandleLearningsRoute_gates(t)
	TestHandleLearningsCRUD(t)
	TestHandleLearningsStats(t)
	TestHandleLoraTrainExpertContext_assistantAgent(t)
}
