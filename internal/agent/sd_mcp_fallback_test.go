package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAttachSDDomainMCPFallsBackToWorkspaceTools(t *testing.T) {
	agent := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), nil)
	if agent.MCPServer != nil {
		t.Fatal("expected no MCP before attach")
	}
	attachSDDomainMCP(agent, "frontend", "Frontend", true, nil)
	if agent.MCPServer == nil {
		t.Fatal("expected workspace-only MCP fallback when sidecar is down")
	}
	srv := mcpServerFromInterface(agent.MCPServer)
	if srv == nil || srv.GetTool("read_file") == nil {
		t.Fatal("expected read_file on workspace-only MCP")
	}
	if srv.GetTool("list_dir") == nil {
		t.Fatal("expected list_dir on workspace-only MCP")
	}
	if !agent.hasWorkspaceTools() {
		t.Fatal("hasWorkspaceTools should be true")
	}
}

func TestNewFrontendAgentRegistersWorkspaceToolsWithoutSidecar(t *testing.T) {
	agent := NewFrontendAgent("FrontendEngineer", ai.NewMockProvider(), nil)
	if !agent.hasWorkspaceTools() {
		t.Fatal("FrontendEngineer must expose read_file even when SD sidecar is down")
	}
}
