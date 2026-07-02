package routing

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

func TestSelectMainModel_bootFixFirstAttemptStaysOnLocalToolModel(t *testing.T) {
	prev := capabilities.Global()
	t.Cleanup(func() { capabilities.SetGlobal(prev) })
	capabilities.SetGlobal(&capabilities.Profiles{
		TaskClasses: map[string][]string{
			"implement":       {"codegemma:7b", "qwen2.5-coder:14b", "devstral:24b"},
			"implement_heavy":   {"qwen2.5-coder:14b", "devstral:24b"},
		},
	})

	in := Input{
		RoutingEnabled:                true,
		ModelCapabilityRoutingEnabled: true,
		LocalToolModel:                "qwen2.5-coder:14b",
		TaskText:                      "the app won't boot — make start-all fails",
		AgentType:                     "frontend",
		BootFixIntent:                 true,
		InstalledOllamaTags: map[string]struct{}{
			"qwen2.5-coder:14b": {},
			"devstral:24b":        {},
		},
		OllamaTagToolFilter: func(tag string) bool {
			return tag == "qwen2.5-coder:14b" || tag == "devstral:24b"
		},
	}
	model, reason := SelectMainModel(in)
	if model != "qwen2.5-coder:14b" {
		t.Fatalf("model=%q reason=%q, want qwen2.5-coder:14b for first boot-fix attempt", model, reason)
	}
}

func TestSelectMainModel_repairTierPrefersFastHeavyModel(t *testing.T) {
	prev := capabilities.Global()
	t.Cleanup(func() { capabilities.SetGlobal(prev) })
	capabilities.SetGlobal(&capabilities.Profiles{
		TaskClasses: map[string][]string{
			"implement_heavy": {"qwen2.5-coder:14b", "devstral:24b"},
		},
	})

	in := Input{
		RoutingEnabled:                true,
		ModelCapabilityRoutingEnabled: true,
		LocalToolModel:                "qwen2.5-coder:14b",
		TaskText:                      "fix the compile error in src/App.tsx",
		AgentType:                     "frontend",
		RepairAttempts:                1,
		InstalledOllamaTags: map[string]struct{}{
			"qwen2.5-coder:14b": {},
			"devstral:24b":        {},
		},
		OllamaTagToolFilter: func(tag string) bool {
			return tag == "qwen2.5-coder:14b" || tag == "devstral:24b"
		},
	}
	model, _ := SelectMainModel(in)
	if model != "qwen2.5-coder:14b" {
		t.Fatalf("model=%q, want qwen2.5-coder:14b first in implement_heavy profile", model)
	}
}
