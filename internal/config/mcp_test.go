package config

import "testing"

func TestMCPEnabledForAgentBiologyPack(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.InstallPack(PackLifeSciences); err != nil {
		t.Fatal(err)
	}
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
	cfg := DefaultConfig()
	if err := cfg.InstallPack(PackSoftwareDevelopment); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	for _, typ := range []string{"backend", "devops", "database", "frontend", "security"} {
		if !cfg.MCPEnabledForAgent(typ) {
			t.Fatalf("expected %s MCP when software-development pack on", typ)
		}
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	cfg.SyncAgentsFromPacks()
	if cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP off when pack off")
	}
}

func TestMCPEnabledForAgentUserOverride(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.InstallPack(PackSoftwareDevelopment); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	cfg.MCP.Agents["frontend"] = false
	if cfg.MCPEnabledForAgent("frontend") {
		t.Fatal("expected user override to disable frontend MCP")
	}
	if !cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP still on")
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
}

func TestMCPEnabledForAgentCADPack(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.InstallPack(PackCAD); err != nil {
		t.Fatal(err)
	}
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
