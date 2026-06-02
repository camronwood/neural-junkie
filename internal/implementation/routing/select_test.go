package routing

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSelectProviderIDLocalFirst(t *testing.T) {
	in := Input{
		RoutingEnabled:  true,
		LocalProviderID: "ollama-local",
		LocalToolModel:  "qwen2.5-coder:7b",
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
			{ID: "claude", Type: "anthropic"},
		},
		DefaultProviderID: "claude",
	}
	id, model, reason := SelectProviderID(in)
	if id != "ollama-local" {
		t.Fatalf("id=%q", id)
	}
	if model != "qwen2.5-coder:7b" {
		t.Fatalf("model=%q", model)
	}
	if reason != "local_ollama_first" {
		t.Fatalf("reason=%q", reason)
	}
}

func TestSelectProviderIDFallback(t *testing.T) {
	in := Input{
		RoutingEnabled:      true,
		LocalProviderID:     "missing",
		FallbackProviderIDs: []string{"claude"},
		Providers: []config.ProviderConfig{
			{ID: "claude", Type: "anthropic"},
		},
	}
	id, _, reason := SelectProviderID(in)
	if id != "claude" || reason != "fallback_provider" {
		t.Fatalf("id=%q reason=%q", id, reason)
	}
}
