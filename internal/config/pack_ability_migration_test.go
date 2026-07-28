package config

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/packs"
)

func TestMigrateRetiredAbilityPackExperts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents = []AgentConfig{
		{Type: "maps", Name: "MapsExpert", Enabled: true, Implementation: "builtin/maps"},
		{Type: "music", Name: "MusicExpert", Enabled: true, Implementation: "builtin/music"},
		{Type: "browser", Name: "WebBrowserExpert", Enabled: true, Implementation: "builtin/browser"},
		{Type: "code-review", Name: "CodeReviewer", Enabled: true, Implementation: "builtin/code-review"},
		{Type: "backend", Name: "BackendEngineer", Enabled: true, Implementation: "builtin/backend"},
		{Type: "assistant", Name: "Assistant", Enabled: true},
	}
	cfg.SpecialistCompose = map[string]SpecialistComposeEntry{
		"maps":    {ChatModel: "qwen2.5:7b"},
		"music":   {ChatModel: "qwen2.5:7b"},
		"backend": {ChatModel: "qwen3.5:27b"},
	}
	cfg.MCP.Agents = map[string]bool{
		"maps": true, "music": true, "browser": true, "code-review": true, "backend": true,
	}

	cfg.migrateRetiredAbilityPackExperts()

	for _, a := range cfg.Agents {
		if packs.IsRetiredAbilityPackAgentType(a.Type) {
			t.Fatalf("retired agent still present: %+v", a)
		}
	}
	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents remaining, got %d", len(cfg.Agents))
	}
	if _, ok := cfg.SpecialistCompose["maps"]; ok {
		t.Fatal("expected maps compose removed")
	}
	if _, ok := cfg.SpecialistCompose["backend"]; !ok {
		t.Fatal("expected backend compose preserved")
	}
	if cfg.MCP.Agents["maps"] || cfg.MCP.Agents["browser"] || cfg.MCP.Agents["code-review"] {
		t.Fatalf("expected retired mcp agent keys cleared: %+v", cfg.MCP.Agents)
	}
	if !cfg.MCP.Agents["backend"] {
		t.Fatal("expected backend mcp agent preserved")
	}
}

func TestPackToolGrantedTo(t *testing.T) {
	SetupTestOfficialPackCatalog(t)
	cfg := DefaultConfig()
	installTestPack(t, cfg, PackMaps)
	if err := cfg.SetPackEnabled(PackMaps, true); err != nil {
		t.Fatal(err)
	}
	cfg.UpsertPackToolGrant(PackToolGrant{
		CapabilityID:  "maps-tools",
		GrantedAgents: []string{"Trip Planner"},
	})
	SetAppConfig(cfg)
	t.Cleanup(func() { SetAppConfig(nil) })

	if !cfg.PackToolGrantedTo("Trip Planner", "maps-tools") {
		t.Fatal("expected grant")
	}
	if cfg.PackToolGrantedTo("Other", "maps-tools") {
		t.Fatal("ungranted expert must not match")
	}
	if err := cfg.SetPackEnabled(PackMaps, false); err != nil {
		t.Fatal(err)
	}
	if cfg.PackToolGrantedTo("Trip Planner", "maps-tools") {
		t.Fatal("grant must require pack enabled")
	}
}
