package routing

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestPlanTaskLightModelWhenInstalled(t *testing.T) {
	installed := map[string]struct{}{"qwen2.5:3b": {}}
	got := PlanTask(PlanInput{
		TaskText:            "Identify schema files in the repo",
		AgentModel:          "qwen2.5:7b",
		DefaultProviderID:   "ollama-local",
		InstalledOllamaTags: installed,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
		},
	})
	if got.OllamaModel != "qwen2.5:3b" {
		t.Fatalf("model = %q, want qwen2.5:3b", got.OllamaModel)
	}
	if got.ModelReason != "light_local_model" {
		t.Fatalf("model reason = %q, want light_local_model", got.ModelReason)
	}
}

func TestPlanTaskSynthesisUsesAgentDefault(t *testing.T) {
	installed := map[string]struct{}{"qwen2.5:3b": {}}
	got := PlanTask(PlanInput{
		TaskText:            "Compile findings from the above tasks into a markdown document",
		AgentModel:          "qwen2.5:7b",
		DefaultProviderID:   "ollama-local",
		InstalledOllamaTags: installed,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
		},
	})
	if got.OllamaModel != "qwen2.5:7b" {
		t.Fatalf("model = %q, want agent default qwen2.5:7b", got.OllamaModel)
	}
	if got.ModelReason != "agent_default_model" {
		t.Fatalf("model reason = %q, want agent_default_model", got.ModelReason)
	}
}

func TestPlanTaskSecurityLoRA(t *testing.T) {
	lora := map[string]struct{}{"nj-security:14b": {}}
	got := PlanTask(PlanInput{
		TaskText:          "Review OAuth security controls",
		AgentModel:        "qwen2.5:7b",
		DefaultProviderID: "ollama-local",
		HasLoRACapability: true,
		InstalledLoRATags: lora,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
		},
	})
	if got.OllamaModel != "nj-security:14b" {
		t.Fatalf("model = %q, want nj-security:14b", got.OllamaModel)
	}
	if got.ModelReason != "security_lora_tag" {
		t.Fatalf("model reason = %q, want security_lora_tag", got.ModelReason)
	}
}

func TestPlanTaskFileDeliverableKeepsAgentDefault(t *testing.T) {
	installed := map[string]struct{}{"qwen2.5:3b": {}}
	got := PlanTask(PlanInput{
		TaskText:            "Identify schema files in resource-api/ and Write collabs/abc123/schema.md defining the API schema",
		AgentModel:          "qwen2.5:7b",
		DefaultProviderID:   "ollama-local",
		InstalledOllamaTags: installed,
		Providers: []config.ProviderConfig{
			{ID: "ollama-local", Type: "ollama"},
		},
	})
	if got.OllamaModel != "qwen2.5:7b" {
		t.Fatalf("model = %q, want agent default qwen2.5:7b", got.OllamaModel)
	}
	if got.ModelReason != "deliverable_task_keep_agent_model" {
		t.Fatalf("model reason = %q, want deliverable_task_keep_agent_model", got.ModelReason)
	}
}

func TestExpectedModelNonOllamaProvider(t *testing.T) {
	providers := []config.ProviderConfig{
		{ID: "claude-main", Type: "anthropic"},
		{ID: "ollama-local", Type: "ollama"},
	}
	got := PlanTask(PlanInput{
		TaskText:              "Review OAuth security controls",
		DefaultProviderID:     "claude-main",
		SmartRoutingEnabled:   true,
		Providers:             providers,
		InstalledLoRATags:     map[string]struct{}{"nj-security:14b": {}},
		HasLoRACapability:     true,
	})
	got.ProviderID = "claude-main"
	if model := got.ExpectedModel(providers); model != "" {
		t.Fatalf("expected empty model for anthropic provider, got %q", model)
	}
}
