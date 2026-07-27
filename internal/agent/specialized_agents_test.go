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

func TestNewCustomExpertAgent_CombinesUserToolsAndExternalMediaOnOneServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	cfg.MCP.UserTools = []config.UserMCPTool{
		{ID: "abc12345", Name: "Read My Site", URL: "https://example.com/api", GrantedAgents: []string{"Media Widget Expert"}},
	}
	cfg.MCP.ExternalMedia = config.ExternalMediaConfig{
		BaseURL:       "https://media.example.com/v1",
		GrantedAgents: []string{"Media Widget Expert"},
	}
	mcp.SetAppConfig(cfg)
	defer mcp.SetAppConfig(nil)

	agent := &Agent{}
	attachUserToolsMCP(agent, "Media Widget Expert")
	if agent.MCPServer == nil {
		t.Fatal("expected a combined MCP server to be attached")
	}
	srv := mcpServerFromInterface(agent.MCPServer)
	if srv == nil {
		t.Fatal("expected non-nil underlying MCP server")
	}
	if srv.GetTool("media_submit") == nil {
		t.Fatal("expected media_submit to be registered on the shared server")
	}
	if srv.GetTool("media_status") == nil {
		t.Fatal("expected media_status to be registered on the shared server")
	}
	found := false
	for name := range srv.ListTools() {
		if strings.HasPrefix(name, "user_read_my_site") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the granted user tool to be registered on the same shared server")
	}
}

func TestNewCustomExpertAgent_NoGrantsLeavesMCPServerNil(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Enabled = true
	mcp.SetAppConfig(cfg)
	defer mcp.SetAppConfig(nil)

	agent := &Agent{}
	attachUserToolsMCP(agent, "Ungranted Expert")
	if agent.MCPServer != nil {
		t.Fatal("expected nil MCPServer when no tools/media are granted")
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
