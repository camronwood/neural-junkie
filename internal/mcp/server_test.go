package mcp

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
)

func TestGetMCPServerConfigFromHubConfig(t *testing.T) {
	SetAppConfig(nil)
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	cfg.MCP.Agents["backend"] = true
	cfg.Packs.Enabled[config.PackSoftwareDevelopment] = true
	cfg.SyncAgentsFromPacks()
	SetAppConfig(cfg)

	got := GetMCPServerConfig("BACKEND")
	if !got.Enabled {
		t.Fatal("expected backend MCP enabled from config")
	}
	if got.Port != 8081 {
		t.Fatalf("expected port 8081, got %d", got.Port)
	}
}

func TestGetMCPServerConfigDisabledWhenMasterOff(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = false
	SetAppConfig(cfg)

	got := GetMCPServerConfig("BACKEND")
	if got.Enabled {
		t.Fatal("expected MCP disabled when master switch off")
	}
}

func TestGetMCPServerConfigCustomPort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Ports = map[string]int{"backend": 9099}
	SetAppConfig(cfg)

	got := GetMCPServerConfig("BACKEND")
	if got.Port != 9099 {
		t.Fatalf("expected port 9099, got %d", got.Port)
	}
}

func TestNewMCPServerDisabled(t *testing.T) {
	cfg := &MCPServerConfig{Enabled: false, Name: "test"}
	_, _, err := NewMCPServer(cfg)
	if err == nil {
		t.Fatal("expected error when disabled")
	}
}
