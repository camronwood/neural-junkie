package config

import "testing"

func TestMCPEnabledForAgentBiologyPack(t *testing.T) {
	cfg := DefaultConfig()
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
	cfg.Packs.Enabled[PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	if !cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP when software-development pack on")
	}
	cfg.Packs.Enabled[PackSoftwareDevelopment] = false
	cfg.SyncAgentsFromPacks()
	if cfg.MCPEnabledForAgent("backend") {
		t.Fatal("expected backend MCP off when pack off")
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
