package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestOllamaImageUnloadAfterUseDefault(t *testing.T) {
	t.Setenv("OLLAMA_IMAGE_KEEP_ALIVE", "")
	if !ollamaImageUnloadAfterUse() {
		t.Fatal("expected unload by default")
	}
}

func TestOllamaImageUnloadAfterUseKeepLoaded(t *testing.T) {
	t.Setenv("OLLAMA_IMAGE_KEEP_ALIVE", "-1")
	if ollamaImageUnloadAfterUse() {
		t.Fatal("expected keep loaded when keep_alive is -1")
	}
}

func TestUnloadOllamaModel(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"done": true, "done_reason": "unload"})
	}))
	defer srv.Close()

	if err := UnloadOllamaModel(context.Background(), srv.URL, "x/flux2-klein:4b"); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "x/flux2-klein:4b" {
		t.Fatalf("model = %#v", got["model"])
	}
	if got["keep_alive"] != float64(0) && got["keep_alive"] != 0 {
		t.Fatalf("keep_alive = %#v", got["keep_alive"])
	}
}

func TestOllamaImageGeneratorUnloadsAfterGenerate(t *testing.T) {
	var unloadCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/generations":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{{"b64_json": "aGVsbG8="}},
			})
		case "/api/generate":
			unloadCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"done": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_IMAGE_KEEP_ALIVE", "0")
	gen := NewOllamaImageGenerator(srv.URL, "x/flux2-klein:4b")
	mime, data, err := gen.GenerateImage(context.Background(), "a ship", "")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/png" || data != "aGVsbG8=" {
		t.Fatalf("mime=%q data=%q", mime, data)
	}
	if unloadCalls != 1 {
		t.Fatalf("unload calls = %d want 1", unloadCalls)
	}
}

func TestOllamaImageGeneratorSkipsUnloadWhenKeepAlive(t *testing.T) {
	clearTrackedOllamaModels()
	t.Cleanup(clearTrackedOllamaModels)
	t.Setenv("NEURAL_JUNKIE_SESSION_OLLAMA_MODELS", filepath.Join(t.TempDir(), "session-ollama-models.json"))

	var unloadCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/images/generations":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{{"b64_json": "aGVsbG8="}},
			})
		case "/api/generate":
			unloadCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"done": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_IMAGE_KEEP_ALIVE", "-1")
	gen := NewOllamaImageGenerator(srv.URL, "x/flux2-klein:4b")
	if _, _, err := gen.GenerateImage(context.Background(), "a ship", ""); err != nil {
		t.Fatal(err)
	}
	if unloadCalls != 0 {
		t.Fatalf("unload calls = %d want 0", unloadCalls)
	}
	if len(TrackedOllamaModels()) != 1 {
		t.Fatalf("expected image model tracked for session unload, got %#v", TrackedOllamaModels())
	}
}
