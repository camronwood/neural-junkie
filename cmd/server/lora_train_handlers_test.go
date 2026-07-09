package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHandleLoraTrainRoute_packDisabled(t *testing.T) {
	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	req := httptest.NewRequest(http.MethodGet, "/api/lora/train/preview?source=channel&source_id=general", nil)
	rec := httptest.NewRecorder()
	handleLoraTrainRoute(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	appConfig = nil
}

func TestHandleLoraTrainExpertContext_repoAgent(t *testing.T) {
	config.SetupTestOfficialPackCatalog(t)
	chatHub = hub.NewHub()
	defer func() {
		chatHub = nil
		appConfig = nil
		loraTrainMgr = nil
	}()

	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	if err := appConfig.InstallPack(config.PackSpecialistTuning); err != nil {
		t.Fatal(err)
	}
	appConfig.Packs.Enabled[config.PackSpecialistTuning] = true

	agent := &protocol.AgentInfo{
		ID:             "repo-test-1",
		Name:           "MyAppExpert",
		Type:           protocol.AgentTypeRepo,
		RepositoryPath: "/Users/me/projects/My-App",
	}
	if err := chatHub.RegisterAgent(agent); err != nil {
		t.Fatal(err)
	}
	if err := chatHub.JoinChannel(agent.ID, "general"); err != nil {
		t.Fatal(err)
	}

	initLoraTrainManager()

	req := httptest.NewRequest(http.MethodGet, "/api/lora/train/expert-context?agent_id=repo-test-1", nil)
	rec := httptest.NewRecorder()
	handleLoraTrainRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["source"] != "repo" {
		t.Fatalf("expected source repo, got %v", body["source"])
	}
	if body["suggested_ollama_tag"] != "nj-repo-my-app:14b" {
		t.Fatalf("expected nj-repo-my-app:14b, got %v", body["suggested_ollama_tag"])
	}
	if body["suggested_base_ollama_tag"] != hfhub.DefaultLoRATrainingCodeBase {
		t.Fatalf("expected suggested base %q, got %v", hfhub.DefaultLoRATrainingCodeBase, body["suggested_base_ollama_tag"])
	}
	if body["source_id"] != "general" {
		t.Fatalf("expected source_id general, got %v", body["source_id"])
	}
}

func TestHandleLoraTrainExpertContext_assistantAgent(t *testing.T) {
	config.SetupTestOfficialPackCatalog(t)
	chatHub = hub.NewHub()
	defer func() {
		chatHub = nil
		appConfig = nil
		loraTrainMgr = nil
	}()

	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	if err := appConfig.InstallPack(config.PackSpecialistTuning); err != nil {
		t.Fatal(err)
	}
	appConfig.Packs.Enabled[config.PackSpecialistTuning] = true

	agent := &protocol.AgentInfo{
		ID:   "asst-test-1",
		Name: "CamronAssistant",
		Type: protocol.AgentTypeAssistant,
	}
	if err := chatHub.RegisterAgent(agent); err != nil {
		t.Fatal(err)
	}
	if err := chatHub.JoinChannel(agent.ID, "general"); err != nil {
		t.Fatal(err)
	}

	initLoraTrainManager()

	req := httptest.NewRequest(http.MethodGet, "/api/lora/train/expert-context?agent_id=asst-test-1", nil)
	rec := httptest.NewRecorder()
	handleLoraTrainRoute(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["source"] != "channel" {
		t.Fatalf("expected source channel, got %v", body["source"])
	}
	if body["suggested_ollama_tag"] != "nj-assistant-camronassistant:14b" {
		t.Fatalf("expected assistant tag, got %v", body["suggested_ollama_tag"])
	}
}

func TestHandleLoraTrainStart_rejectsQwenBase(t *testing.T) {
	config.SetupTestOfficialPackCatalog(t)
	chatHub = hub.NewHub()
	defer func() {
		chatHub = nil
		appConfig = nil
		loraTrainMgr = nil
	}()

	appConfig = config.DefaultConfig()
	appConfig.Packs = config.DefaultPacksConfig()
	if err := appConfig.InstallPack(config.PackSpecialistTuning); err != nil {
		t.Fatal(err)
	}
	appConfig.Packs.Enabled[config.PackSpecialistTuning] = true
	initLoraTrainManager()

	body := `{"source":"channel","source_id":"general","base_ollama_tag":"qwen2.5-coder:14b","ollama_tag":"nj-test:14b"}`
	req := httptest.NewRequest(http.MethodPost, "/api/lora/train", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLoraTrainRoute(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cannot be used for LoRA training") {
		t.Fatalf("expected LoRA training rejection, got %s", rec.Body.String())
	}
}
