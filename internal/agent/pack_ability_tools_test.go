package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestMapsToolsEnabledForAssistantWhenPackOn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackMaps)
	if err := cfg.SetPackEnabled(config.PackMaps, true); err != nil {
		t.Fatal(err)
	}
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	a := &Agent{Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := &protocol.Message{Content: "walk from A to B"}
	if !a.mapsToolsEnabledForMessage(msg) {
		t.Fatal("Assistant should get maps tools when maps pack is enabled")
	}

	if err := cfg.SetPackEnabled(config.PackMaps, false); err != nil {
		t.Fatal(err)
	}
	if a.mapsToolsEnabledForMessage(msg) {
		t.Fatal("Assistant must not get maps tools when pack is off")
	}
}

func TestMapsToolsEnabledForGrantedCustomExpert(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackMaps)
	if err := cfg.SetPackEnabled(config.PackMaps, true); err != nil {
		t.Fatal(err)
	}
	cfg.UpsertPackToolGrant(config.PackToolGrant{
		CapabilityID:  "maps-tools",
		GrantedAgents: []string{"Trip Planner"},
	})
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	granted := &Agent{Info: protocol.AgentInfo{Name: "Trip Planner", Type: protocol.AgentTypeExpert}}
	ungranted := &Agent{Info: protocol.AgentInfo{Name: "Other Expert", Type: protocol.AgentTypeExpert}}
	msg := &protocol.Message{Content: "route me downtown"}
	if !granted.mapsToolsEnabledForMessage(msg) {
		t.Fatal("granted custom expert should get maps tools")
	}
	if ungranted.mapsToolsEnabledForMessage(msg) {
		t.Fatal("ungranted custom expert must not get maps tools")
	}
}

func TestAttachPackToolGrantsOnCustomExpert(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	config.InstallTestPack(t, cfg, config.PackMaps)
	config.InstallTestPack(t, cfg, config.PackSoftwareDevelopment)
	config.InstallTestPack(t, cfg, config.PackWebBrowser)
	if err := cfg.SetPackEnabled(config.PackMaps, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackSoftwareDevelopment, true); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetPackEnabled(config.PackWebBrowser, true); err != nil {
		t.Fatal(err)
	}
	cfg.UpsertPackToolGrant(config.PackToolGrant{
		CapabilityID:  "maps-tools",
		GrantedAgents: []string{"Widget Expert"},
	})
	cfg.UpsertPackToolGrant(config.PackToolGrant{
		CapabilityID:  "web-browser",
		GrantedAgents: []string{"Widget Expert"},
	})
	config.SetAppConfig(cfg)
	t.Cleanup(func() { config.SetAppConfig(nil) })

	ag := NewCustomExpertAgent("Widget Expert", []string{"Travel"}, ai.NewMockProvider(), shouldRespondTestHub{})
	if ag.MCPServer == nil {
		t.Fatal("expected MCP server after pack tool grants")
	}
	names := map[string]bool{}
	for _, tool := range claudeToolsFromMCPServer(mcpServerFromInterface(ag.MCPServer), nil) {
		names[tool.Name] = true
	}
	if !names["maps_geocode"] || !names["browser_screenshot"] {
		t.Fatalf("expected maps + browser tools, got %v", names)
	}
}
