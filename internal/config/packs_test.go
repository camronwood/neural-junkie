package config

import "testing"

func TestSyncAgentsFromPacksLifeSciences(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Packs.Enabled[PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()

	if !cfg.AgentTypeEnabled("biology") {
		t.Fatal("expected biology enabled")
	}
	foundChat := false
	foundTool := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == BioOllamaChatModel {
			foundChat = true
		}
		if m == BioOllamaToolModel {
			foundTool = true
		}
	}
	if !foundChat {
		t.Fatalf("expected %s in models_to_ensure", BioOllamaChatModel)
	}
	if !foundTool {
		t.Fatalf("expected %s in models_to_ensure", BioOllamaToolModel)
	}
}

func TestSyncAgentsFromPacksDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Agents = append(cfg.Agents, AgentConfig{Type: "biology", Name: "BiologyExpert", Enabled: true})
	cfg.Packs.Enabled[PackLifeSciences] = false
	cfg.SyncAgentsFromPacks()

	if cfg.AgentTypeEnabled("biology") {
		t.Fatal("expected biology disabled when pack off")
	}
}

func TestAvailableExpertPresets(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Packs.Enabled[PackLifeSciences] = true
	presets := cfg.AvailableExpertPresets()
	hasBio := false
	for _, p := range presets {
		if p.Slug == "biology" && p.FromPack == PackLifeSciences {
			hasBio = true
		}
	}
	if !hasBio {
		t.Fatal("expected biology preset from life-sciences pack")
	}
}

func TestSyncAgentsFromPacksSoftwareDevelopment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()

	for _, typ := range devSpecialistTypes {
		if !cfg.AgentTypeEnabled(typ) {
			t.Fatalf("expected %s enabled", typ)
		}
	}
	foundCoder := false
	for _, m := range cfg.Ollama.ModelsToEnsure {
		if m == DevOllamaCodeModel {
			foundCoder = true
		}
	}
	if !foundCoder {
		t.Fatalf("expected %s in models_to_ensure", DevOllamaCodeModel)
	}
}

func TestSyncAgentsFromPacksSoftwareDevelopmentDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Agents = append(cfg.Agents, AgentConfig{Type: "backend", Name: "GoExpert", Enabled: true})
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	cfg.SyncAgentsFromPacks()

	if cfg.AgentTypeEnabled("backend") {
		t.Fatal("expected backend disabled when software-development pack off")
	}
}

func TestAvailableExpertPresetsDevPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	presets := cfg.AvailableExpertPresets()
	hasRust := false
	for _, p := range presets {
		if p.Slug == "rust" && p.FromPack == PackSoftwareDevelopment {
			hasRust = true
		}
	}
	if !hasRust {
		t.Fatal("expected rust preset from software-development pack")
	}
}

func TestPresetExpertAllowed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	if !cfg.PresetExpertAllowed("assistant") {
		t.Fatal("assistant should always be allowed")
	}
	if cfg.PresetExpertAllowed("rust") {
		t.Fatal("rust should be blocked when dev pack off")
	}
	if cfg.PresetExpertAllowed("biology") {
		t.Fatal("biology should be blocked when life-sciences off")
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	if !cfg.PresetExpertAllowed("rust") {
		t.Fatal("rust should be allowed when dev pack on")
	}
	if !cfg.PresetExpertAllowed("guitar") {
		t.Fatal("custom slugs should be allowed")
	}
}

func TestMigrateSoftwareDevelopmentPackIfNeeded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Agents = []AgentConfig{{Type: "backend", Name: "GoExpert", Enabled: true}}
	delete(cfg.Packs.Enabled, PackSoftwareDevelopment)
	cfg.migrateSoftwareDevelopmentPackIfNeeded()
	if !cfg.Packs.Enabled[PackSoftwareDevelopment] {
		t.Fatal("expected migration to enable software-development pack")
	}
}

func TestSpecialistShouldBeRunningPackOffOverridesConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Packs = DefaultPacksConfig()
	cfg.Agents = []AgentConfig{{Type: "backend", Name: "GoExpert", Enabled: true}}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	if cfg.SpecialistShouldBeRunning("backend") {
		t.Fatal("expected backend not running when software-development pack off even if config enabled")
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	if !cfg.SpecialistShouldBeRunning("backend") {
		t.Fatal("expected backend running when pack on and config enabled")
	}
}

func TestDefaultConfigNoDevAgents(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Agents) != 0 {
		t.Fatalf("expected no default agents, got %d", len(cfg.Agents))
	}
	if cfg.IsPackEnabled(PackSoftwareDevelopment) {
		t.Fatal("expected dev pack off in defaults")
	}
	presets := cfg.AvailableExpertPresets()
	for _, p := range presets {
		if p.Slug == "rust" {
			t.Fatal("rust preset should not appear when dev pack off")
		}
	}
}
