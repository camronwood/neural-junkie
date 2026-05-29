package security

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
)

func TestNewSecurityMCPRegistersTools(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	if err := cfg.InstallPack(config.PackSoftwareDevelopment); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)

	s, err := NewSecurityMCP()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.GetMCPServer().ListTools()) < 5 {
		t.Fatalf("expected 5 security tools")
	}
}
