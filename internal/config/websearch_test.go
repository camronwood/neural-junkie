package config

import "testing"

func TestWebSearchReady(t *testing.T) {
	if (WebSearchConfig{}).Ready() {
		t.Fatal("empty config should not be ready")
	}
	cfg := WebSearchConfig{Enabled: true, APIKey: "key"}
	if !cfg.Ready() {
		t.Fatal("expected ready with key")
	}
	if got := cfg.ProviderName(); got != "tavily" {
		t.Fatalf("default provider = %q", got)
	}
	if got := cfg.MaxResultsOrDefault(); got != 5 {
		t.Fatalf("max = %d", got)
	}
}

func TestWebSearchTavilyKeylessReady(t *testing.T) {
	cfg := WebSearchConfig{Enabled: true, Provider: "tavily", Keyless: true}
	if !cfg.Ready() {
		t.Fatal("expected tavily keyless to be ready without api key")
	}
}

func TestWebSearchBraveRequiresKey(t *testing.T) {
	cfg := WebSearchConfig{Enabled: true, Provider: "brave"}
	if cfg.Ready() {
		t.Fatal("brave should require api key")
	}
}
