package config

import "testing"

func TestOllamaConfigModelUsesReactTools(t *testing.T) {
	cfg := OllamaConfig{
		ReactToolsEnabled: true,
		ReactToolModels:   []string{"gemma3:12b"},
	}
	if !cfg.ModelUsesReactTools("gemma3:12b") {
		t.Fatal("expected gemma3:12b")
	}
	if cfg.ModelUsesReactTools("qwen3.5:9b") {
		t.Fatal("qwen should not use react")
	}
}

func TestOllamaConfigModelUsesReactToolsDisabled(t *testing.T) {
	cfg := OllamaConfig{ReactToolsEnabled: false, ReactToolModels: []string{"gemma3:12b"}}
	if cfg.ModelUsesReactTools("gemma3:12b") {
		t.Fatal("expected disabled")
	}
}

func TestOllamaConfigReactToolModelsDefault(t *testing.T) {
	cfg := OllamaConfig{}
	models := cfg.ReactToolModelsOrDefault()
	if len(models) != 1 || models[0] != "gemma3:12b" {
		t.Fatalf("models=%v", models)
	}
}
