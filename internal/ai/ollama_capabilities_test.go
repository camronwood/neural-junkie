package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaTagSupportsTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(ollamaShowResponse{Capabilities: []string{"completion", "tools"}})
	}))
	defer srv.Close()

	if !OllamaTagSupportsTools(context.Background(), srv.URL, "qwen3.5:9b") {
		t.Fatal("expected tools support")
	}
}

func TestOllamaTagSupportsToolsNoToolsCapability(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaShowResponse{Capabilities: []string{"completion"}})
	}))
	defer srv.Close()

	if OllamaTagSupportsTools(context.Background(), srv.URL, "deepseek-coder:6.7b") {
		t.Fatal("expected no tools support")
	}
}

func TestOllamaTagSupportsToolsKnownNoTools(t *testing.T) {
	if OllamaTagSupportsTools(context.Background(), "http://unused", "nj-bio:8b") {
		t.Fatal("nj-bio should be treated as non-tool")
	}
}
