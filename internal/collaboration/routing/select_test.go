package routing

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestSelectProviderID_default(t *testing.T) {
	providers := []config.ProviderConfig{
		{ID: "local", Type: "ollama"},
		{ID: "cloud", Type: "anthropic"},
	}
	id, reason := SelectProviderID(Input{
		TaskText:          "Implement the full payment reconciliation service.",
		HasUserImages:     false,
		Providers:         providers,
		DefaultProviderID: "cloud",
	})
	if id != "cloud" || reason != "default_agent_provider" {
		t.Fatalf("got id=%q reason=%q", id, reason)
	}
}

func TestSelectProviderID_security(t *testing.T) {
	providers := []config.ProviderConfig{
		{ID: "local", Type: "ollama"},
		{ID: "cloud", Type: "anthropic"},
	}
	id, reason := SelectProviderID(Input{
		TaskText:          "Review OWASP auth and JWT handling for this API.",
		HasUserImages:     false,
		Providers:         providers,
		DefaultProviderID: "local",
	})
	if id != "cloud" || reason != "security_premium" {
		t.Fatalf("got id=%q reason=%q", id, reason)
	}
}

func TestSelectProviderID_cheap(t *testing.T) {
	providers := []config.ProviderConfig{
		{ID: "local", Type: "ollama"},
		{ID: "cloud", Type: "anthropic"},
	}
	id, reason := SelectProviderID(Input{
		TaskText:          "Please fix a small typo in the label wording.",
		HasUserImages:     false,
		Providers:         providers,
		DefaultProviderID: "cloud",
	})
	if id != "local" || reason != "cheap_local" {
		t.Fatalf("got id=%q reason=%q", id, reason)
	}
}

func TestSelectProviderID_visionCheapest(t *testing.T) {
	providers := []config.ProviderConfig{
		{ID: "local", Type: "ollama"},
		{ID: "cloud", Type: "anthropic"},
	}
	id, reason := SelectProviderID(Input{
		TaskText:          "Describe this screenshot.",
		HasUserImages:     true,
		Providers:         providers,
		DefaultProviderID: "cloud",
	})
	if id != "local" || reason != "vision_cheapest" {
		t.Fatalf("got id=%q reason=%q", id, reason)
	}
}

func TestSelectProviderID_emptyProviders(t *testing.T) {
	id, reason := SelectProviderID(Input{
		TaskText:          "anything",
		Providers:         nil,
		DefaultProviderID: "only",
	})
	if id != "only" || reason != "no_providers_or_default" {
		t.Fatalf("got id=%q reason=%q", id, reason)
	}
}
