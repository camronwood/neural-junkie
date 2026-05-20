package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaCapabilitiesIncludeTools(t *testing.T) {
	if !ollamaCapabilitiesIncludeTools([]string{"completion", "tools"}) {
		t.Fatal("expected tools capability")
	}
	if ollamaCapabilitiesIncludeTools([]string{"completion"}) {
		t.Fatal("expected no tools")
	}
}

func TestOllamaModelLikelyNoNativeTools(t *testing.T) {
	if !ollamaModelLikelyNoNativeTools("nj-bio:8b") {
		t.Fatal("nj-bio should skip native tools")
	}
	if !ollamaModelLikelyNoNativeTools("koesn/llama3-openbiollm-8b:latest") {
		t.Fatal("koesn openbiollm should skip native tools")
	}
	if ollamaModelLikelyNoNativeTools("llama3.1") {
		t.Fatal("llama3.1 should not match fast path")
	}
}

func TestOllamaSupportsToolsNjBio(t *testing.T) {
	p := NewOllamaProviderWithConfig("http://localhost:11434", "nj-bio:8b")
	if p.SupportsTools() {
		t.Fatal("nj-bio should not report native tool support")
	}
}

func TestOllamaFetchCapabilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(ollamaShowResponse{Capabilities: []string{"completion"}})
	}))
	defer srv.Close()

	caps, err := ollamaFetchCapabilities(context.Background(), srv.URL, "m")
	if err != nil {
		t.Fatal(err)
	}
	if len(caps) != 1 || caps[0] != "completion" {
		t.Fatalf("caps=%v", caps)
	}
	p := NewOllamaProviderWithConfig(srv.URL, "custom:7b")
	if p.SupportsTools() {
		t.Fatal("completion-only model should not support tools")
	}
}
