package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNoteAndUnloadTrackedOllamaModels(t *testing.T) {
	clearTrackedOllamaModels()
	t.Cleanup(clearTrackedOllamaModels)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NEURAL_JUNKIE_SESSION_OLLAMA_MODELS", filepath.Join(home, "session-ollama-models.json"))

	var unloadHits int
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		unloadHits++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"done": true, "done_reason": "unload"})
	}))
	defer srv.Close()

	NoteOllamaModelUsed(srv.URL, "qwen3.5:9b")
	NoteOllamaModelUsed(srv.URL, "qwen3.5:9b") // dedupe
	tracked := TrackedOllamaModels()
	if len(tracked) != 1 {
		t.Fatalf("tracked = %#v want 1", tracked)
	}

	path := filepath.Join(home, "session-ollama-models.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted session file: %v", err)
	}

	unloaded, errs := UnloadTrackedOllamaModels(context.Background())
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if len(unloaded) != 1 || unloaded[0] != "qwen3.5:9b" {
		t.Fatalf("unloaded = %#v", unloaded)
	}
	if unloadHits != 1 || gotModel != "qwen3.5:9b" {
		t.Fatalf("hits=%d model=%q", unloadHits, gotModel)
	}
	if len(TrackedOllamaModels()) != 0 {
		t.Fatal("expected tracker cleared")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected session file removed, err=%v", err)
	}
}

func TestUnloadTrackedFallsBackToPersistedFile(t *testing.T) {
	clearTrackedOllamaModels()
	t.Cleanup(clearTrackedOllamaModels)
	home := t.TempDir()
	t.Setenv("HOME", home)
	sessionPath := filepath.Join(home, "session-ollama-models.json")
	t.Setenv("NEURAL_JUNKIE_SESSION_OLLAMA_MODELS", sessionPath)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"done": true})
	}))
	defer srv.Close()

	doc := sessionModelsFile{}
	doc.Models = append(doc.Models, struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	}{Endpoint: srv.URL, Model: "nomic-embed-text"})
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(sessionPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	unloaded, errs := UnloadTrackedOllamaModels(context.Background())
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if len(unloaded) != 1 || unloaded[0] != "nomic-embed-text" || hits != 1 {
		t.Fatalf("unloaded=%#v hits=%d", unloaded, hits)
	}
}
