package config

import "testing"

func TestProviderForAgentModelOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents = []AgentConfig{
		{
			Type:       "security",
			Name:       "SecurityReviewer",
			Enabled:    true,
			ProviderID: "ollama-local",
			Model:      "nj-security:14b",
		},
	}
	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != "nj-security:14b" {
		t.Fatalf("model = %q", p.Model)
	}
}

func TestSetAgentRuntimeProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents = []AgentConfig{
		{Type: "backend", Name: "BackendEngineer", Enabled: true, ProviderID: "ollama-local"},
	}
	if !cfg.SetAgentRuntimeProvider("BackendEngineer", "backend", "ollama-local", "nj-backend:14b") {
		t.Fatal("expected match")
	}
	if cfg.Agents[0].Model != "nj-backend:14b" {
		t.Fatalf("model = %q", cfg.Agents[0].Model)
	}
	cfg.ClearAllAgentModels()
	if cfg.Agents[0].Model != "" {
		t.Fatalf("expected cleared model, got %q", cfg.Agents[0].Model)
	}
}
