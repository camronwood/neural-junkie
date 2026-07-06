package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/cad"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func enableCADPackForTest(t *testing.T) {
	t.Helper()
	config.SetupTestOfficialPackCatalog(t)
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	if err := cfg.InstallPack(config.PackCAD); err != nil {
		t.Fatal(err)
	}
	cfg.Packs.Enabled[config.PackCAD] = true
	cfg.SyncAgentsFromPacks()
	mcp.SetAppConfig(cfg)
}

func TestCADAgentSkipsWorkspaceTools(t *testing.T) {
	enableCADPackForTest(t)

	agent := &Agent{}
	cadMCP, err := cad.NewCADMCP()
	if err != nil {
		t.Fatal(err)
	}
	startDomainAgentMCP(agent, "CAD", cadMCP)

	srv := mcpServerFromInterface(agent.MCPServer)
	if srv == nil {
		t.Fatal("expected MCP server")
	}
	if srv.GetTool("list_dir") != nil {
		t.Fatal("CAD agent should not register workspace list_dir")
	}
	if srv.GetTool("write_openscad") == nil {
		t.Fatal("CAD agent should register write_openscad")
	}
}

func TestAppendMCPToolsPromptCADGreetingGuidance(t *testing.T) {
	enableCADPackForTest(t)

	agent := &Agent{}
	cadMCP, err := cad.NewCADMCP()
	if err != nil {
		t.Fatal(err)
	}
	startDomainAgentMCP(agent, "CAD", cadMCP)

	var system strings.Builder
	appendMCPToolsPrompt(&system, mcpServerFromInterface(agent.MCPServer), protocol.AgentTypeCAD, nil)
	prompt := system.String()
	if !strings.Contains(prompt, "without calling tools") {
		t.Fatalf("expected greeting guidance in CAD tools prompt, got:\n%s", prompt)
	}
}
