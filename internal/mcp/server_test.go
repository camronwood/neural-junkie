package mcp

import (
	"os"
	"testing"
)

func TestGetMCPServerConfigDisabled(t *testing.T) {
	os.Unsetenv("ENABLE_MCP")
	os.Unsetenv("ENABLE_BACKEND_MCP")
	cfg := GetMCPServerConfig("BACKEND")
	if cfg.Enabled {
		t.Fatal("expected MCP disabled without env flags")
	}
	if cfg.Port != 8081 {
		t.Fatalf("expected default port 8081, got %d", cfg.Port)
	}
}

func TestGetMCPServerConfigEnabled(t *testing.T) {
	t.Setenv("ENABLE_MCP", "true")
	t.Setenv("ENABLE_BACKEND_MCP", "true")
	t.Setenv("MCP_BACKEND_PORT", "9099")
	cfg := GetMCPServerConfig("BACKEND")
	if !cfg.Enabled {
		t.Fatal("expected MCP enabled")
	}
	if cfg.Port != 9099 {
		t.Fatalf("expected port 9099, got %d", cfg.Port)
	}
}

func TestNewMCPServerDisabled(t *testing.T) {
	cfg := &MCPServerConfig{Enabled: false, Name: "test"}
	_, _, err := NewMCPServer(cfg)
	if err == nil {
		t.Fatal("expected error when disabled")
	}
}
