package frontend

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
)

func TestNewFrontendMCPDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = false
	mcp.SetAppConfig(cfg)

	_, err := NewFrontendMCP()
	if err == nil {
		t.Fatal("expected error when MCP disabled")
	}
}

func TestFrontendMCPRegistersTools(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	if err := cfg.InstallPack(config.PackSoftwareDevelopment); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)

	f, err := NewFrontendMCP()
	if err != nil {
		t.Fatal(err)
	}
	tools := f.GetMCPServer().ListTools()
	if len(tools) < 4 {
		t.Fatalf("expected at least 4 tools, got %d", len(tools))
	}
}
