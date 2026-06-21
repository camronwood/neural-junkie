package config

import "testing"

func TestProviderForAgent_devSpecialistAvoidsBiologyModel(t *testing.T) {
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, true)
	cfg.AI.Providers = []ProviderConfig{{
		ID:       "ollama-local",
		Type:     "ollama",
		Model:    BioOllamaChatModel,
		Endpoint: "http://localhost:11434",
	}}
	cfg.AI.DefaultProviderID = "ollama-local"
	cfg.Agents = []AgentConfig{{
		Type:       "frontend",
		Name:       "FrontendEngineer",
		Enabled:    true,
		ProviderID: "ollama-local",
	}}

	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != DevOllamaCodeModel {
		t.Fatalf("model = %q, want %q", p.Model, DevOllamaCodeModel)
	}
}
