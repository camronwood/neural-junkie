package routing

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSelectProviderIDLocalFirst(t *testing.T) {
	in := Input{
		RoutingEnabled:  true,
		LocalProviderID: "ollama-local",
		LocalToolModel:  "qwen3.5:9b",
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
	if model != "qwen3.5:9b" {
		t.Fatalf("model=%q", model)
	}
	if reason != "local_ollama_first" {
		t.Fatalf("reason=%q", reason)
	}
}

func TestSelectProviderIDFallback(t *testing.T) {
	in := Input{
		RoutingEnabled:            true,
		LocalProviderID:           "missing",
		FallbackProviderIDs:       []string{"claude"},
		FrontierEscalationEnabled: true,
		Providers: []config.ProviderConfig{
			{ID: "claude", Type: "anthropic"},
		},
	}
	id, _, reason := SelectProviderID(in)
	if id != "claude" || reason != "fallback_provider" {
		t.Fatalf("id=%q reason=%q", id, reason)
	}
}

func TestSelectProviderIDReliableRepairTier(t *testing.T) {
	in := Input{
		RoutingEnabled:            true,
		LocalProviderID:           "ollama-local",
		ReliableProviderID:        "claude",
		ReliableToolModel:         "qwen2.5-coder:14b",
		RepairAttempts:            2,
		LocalEscalationEnabled:    true,
		FrontierEscalationEnabled: true,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
			{ID: "claude", Type: "anthropic"},
		},
	}
	id, model, reason := SelectProviderID(in)
	if id != "claude" || reason != "frontier_after_local_exhaustion" {
		t.Fatalf("id=%q reason=%q", id, reason)
	}
	if model != "qwen2.5-coder:14b" {
		t.Fatalf("model=%q", model)
	}
}

func TestResolveToolModelRepairUsesReliable(t *testing.T) {
	in := Input{
		LocalToolModel:         "qwen3.5:9b",
		ReliableToolModel:      "qwen2.5-coder:14b",
		RepairAttempts:         1,
		LocalEscalationEnabled: true,
	}
	got := resolveToolModel(in)
	if got != "qwen2.5-coder:14b" {
		t.Fatalf("got %q", got)
	}
}

func TestFrontierBlockedWithoutExplicitConsent(t *testing.T) {
	in := Input{
		RoutingEnabled: true, LocalProviderID: "ollama-local",
		ReliableProviderID: "claude", RepairAttempts: 2,
		LocalEscalationEnabled: true,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
			{ID: "claude", Type: "anthropic"},
		},
	}
	id, _, reason := SelectProviderID(in)
	if id != "ollama-local" || reason != "local_ollama_first" {
		t.Fatalf("id=%q reason=%q; configured frontier must not imply consent", id, reason)
	}
}

func TestFrontierRequiresLocalExhaustionAndConsent(t *testing.T) {
	in := Input{
		RoutingEnabled: true, LocalProviderID: "ollama-local",
		ReliableProviderID: "claude", RepairAttempts: 1,
		LocalEscalationEnabled: true, FrontierEscalationEnabled: true,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
			{ID: "claude", Type: "anthropic"},
		},
	}
	id, _, _ := SelectProviderID(in)
	if id != "ollama-local" {
		t.Fatalf("repair 1 id=%q, want local tier", id)
	}
	in.RepairAttempts = 2
	id, _, reason := SelectProviderID(in)
	if id != "claude" || reason != "frontier_after_local_exhaustion" {
		t.Fatalf("repair 2 id=%q reason=%q, want consented frontier", id, reason)
	}
}
