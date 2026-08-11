package config

import "testing"

func TestMCPEnabledForAgentBiologyPack(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackLifeSciences)
	cfg.Packs.Enabled[PackLifeSciences] = true
	cfg.SyncAgentsFromPacks()
	if !cfg.MCPEnabledForAgent("biology") {
		t.Fatal("expected biology MCP when life-sciences pack on")
	}
	cfg.Packs.Enabled[PackLifeSciences] = false
	cfg.SyncAgentsFromPacks()
	if cfg.MCPEnabledForAgent("biology") {
		t.Fatal("expected biology MCP off when pack off")
	}
}

func TestMCPEnabledForAgentSoftwareDevelopmentPack(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	if !cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP when default coder is running")
	}
	for _, typ := range []string{"devops", "database", "frontend", "security"} {
		if cfg.MCPEnabledForAgent(typ) {
			t.Fatalf("expected %s MCP off while specialist is disabled", typ)
		}
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	cfg.SyncAgentsFromPacks()
	if cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP off when pack off")
	}
}

func TestMCPEnabledForAgentUserOverride(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackSoftwareDevelopment)
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	cfg.MCP.Agents["backend"] = false
	if cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected user override to disable backend MCP")
	}
}

func TestBiologyMCPSettingsDefaults(t *testing.T) {
	b := BiologyMCPConfig{}
	if b.ESMFoldModelOrDefault() != defaultESMFoldModel {
		t.Fatalf("esmfold default: %s", b.ESMFoldModelOrDefault())
	}
	if b.MaxFoldLengthOrDefault() != defaultMaxFoldLength {
		t.Fatalf("fold len default: %d", b.MaxFoldLengthOrDefault())
	}
	if b.ChatModelOrDefault() != BioOllamaChatModel {
		t.Fatalf("chat model default: %s", b.ChatModelOrDefault())
	}
	if b.ToolModelOrDefault() != BioOllamaToolModel {
		t.Fatalf("tool model default: %s", b.ToolModelOrDefault())
	}
}

func TestBiologyChatModelOrDefaultPrefersMCPOverDelegation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP.Biology.ChatModel = "nj-bio:8b"
	cfg.Delegation.BiologyChatModel = "koesn/llama3-openbiollm-8b:latest"
	if got := cfg.BiologyChatModelOrDefault(); got != "nj-bio:8b" {
		t.Fatalf("got %q", got)
	}
}

func TestMigrateBiologyMCPModels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Delegation.BiologyChatModel = "nj-bio:8b"
	cfg.Delegation.BiologyToolModel = "custom-tool:7b"
	cfg.MigrateBiologyMCPModels()
	if cfg.MCP.Biology.ChatModel != "nj-bio:8b" {
		t.Fatalf("chat model = %q", cfg.MCP.Biology.ChatModel)
	}
	if cfg.MCP.Biology.ToolModel != "custom-tool:7b" {
		t.Fatalf("tool model = %q", cfg.MCP.Biology.ToolModel)
	}
	cfg.MigrateBiologyMCPModels()
	if cfg.MCP.Biology.ChatModel != "nj-bio:8b" {
		t.Fatal("migration should not overwrite existing mcp.biology values")
	}
}

func TestMCPEnabledForAgentCADPack(t *testing.T) {
	setupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackCAD)
	cfg.Packs.Enabled[PackCAD] = true
	cfg.SyncAgentsFromPacks()
	if !cfg.MCPEnabledForAgent("cad") {
		t.Fatal("expected cad MCP when CAD pack on")
	}
	cfg.Packs.Enabled[PackCAD] = false
	cfg.SyncAgentsFromPacks()
	if cfg.MCPEnabledForAgent("cad") {
		t.Fatal("expected cad MCP off when pack off")
	}
}

func TestCadMCPSettingsDefaults(t *testing.T) {
	c := CadMCPConfig{}
	if c.OpenSCADPathOrDefault() != "openscad" {
		t.Fatalf("openscad path default: %s", c.OpenSCADPathOrDefault())
	}
	if c.ChatModelOrDefault() != CadOllamaChatModel {
		t.Fatalf("chat model default: %s", c.ChatModelOrDefault())
	}
	if c.ToolModelOrDefault() != CadOllamaToolModel {
		t.Fatalf("tool model default: %s", c.ToolModelOrDefault())
	}
}
