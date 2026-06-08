package config

import "testing"

func TestProviderForAgent_biologyUsesOpenBioDespiteCoderProviderDefault(t *testing.T) {
	cfg := DefaultConfig()
	_ = cfg.InstallPack(PackSoftwareDevelopment)
	_ = cfg.InstallPack(PackLifeSciences)
	_ = cfg.SetPackEnabled(PackSoftwareDevelopment, true)
	_ = cfg.SetPackEnabled(PackLifeSciences, true)
	cfg.Packs.LayoutOwner = PackSoftwareDevelopment
	cfg.AI.Providers = []ProviderConfig{{
		ID:       "ollama-local",
		Type:     "ollama",
		Model:    DevOllamaCodeModel,
		Endpoint: "http://localhost:11434",
	}}
	cfg.AI.DefaultProviderID = "ollama-local"
	cfg.Agents = []AgentConfig{{
		Type:       "biology",
		Name:       "BiologyExpert",
		Enabled:    true,
		ProviderID: "ollama-local",
	}}

	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != BioOllamaChatModel {
		t.Fatalf("model = %q, want %q", p.Model, BioOllamaChatModel)
	}
}

func TestProviderForAgent_biologyRespectsExplicitAgentModelOverride(t *testing.T) {
	cfg := DefaultConfig()
	_ = cfg.InstallPack(PackLifeSciences)
	_ = cfg.SetPackEnabled(PackLifeSciences, true)
	cfg.AI.Providers = []ProviderConfig{{
		ID:       "ollama-local",
		Type:     "ollama",
		Model:    DevOllamaCodeModel,
		Endpoint: "http://localhost:11434",
	}}
	cfg.AI.DefaultProviderID = "ollama-local"
	cfg.Agents = []AgentConfig{{
		Type:       "biology",
		Name:       "BiologyExpert",
		Enabled:    true,
		ProviderID: "ollama-local",
		Model:      "qwen2.5-coder:14b",
	}}

	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != "qwen2.5-coder:14b" {
		t.Fatalf("model = %q, want explicit override", p.Model)
	}
}

func TestProviderForAgent_biologyUsesMCPChatModelOverride(t *testing.T) {
	cfg := DefaultConfig()
	_ = cfg.InstallPack(PackLifeSciences)
	_ = cfg.SetPackEnabled(PackLifeSciences, true)
	cfg.MCP.Biology.ChatModel = "nj-bio:8b"
	cfg.AI.Providers = []ProviderConfig{{
		ID:       "ollama-local",
		Type:     "ollama",
		Model:    DevOllamaCodeModel,
		Endpoint: "http://localhost:11434",
	}}
	cfg.AI.DefaultProviderID = "ollama-local"
	cfg.Agents = []AgentConfig{{
		Type:       "biology",
		Name:       "BiologyExpert",
		Enabled:    true,
		ProviderID: "ollama-local",
	}}

	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != "nj-bio:8b" {
		t.Fatalf("model = %q, want nj-bio:8b from mcp.biology.chat_model", p.Model)
	}
}

func TestProviderForAgent_biologyFallsBackToDelegationChatModel(t *testing.T) {
	cfg := DefaultConfig()
	_ = cfg.InstallPack(PackLifeSciences)
	_ = cfg.SetPackEnabled(PackLifeSciences, true)
	cfg.Delegation.BiologyChatModel = "nj-bio:8b"
	cfg.AI.Providers = []ProviderConfig{{
		ID:       "ollama-local",
		Type:     "ollama",
		Model:    DevOllamaCodeModel,
		Endpoint: "http://localhost:11434",
	}}
	cfg.AI.DefaultProviderID = "ollama-local"
	cfg.Agents = []AgentConfig{{
		Type:       "biology",
		Name:       "BiologyExpert",
		Enabled:    true,
		ProviderID: "ollama-local",
	}}

	p := cfg.ProviderForAgent(cfg.Agents[0])
	if p == nil {
		t.Fatal("nil provider")
	}
	if p.Model != "nj-bio:8b" {
		t.Fatalf("model = %q, want nj-bio:8b from legacy delegation settings", p.Model)
	}
}
