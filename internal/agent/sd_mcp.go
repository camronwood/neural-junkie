package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/mcp/packremote"
	"github.com/camronwood/neural-junkie/internal/packsidecar"
	"github.com/mark3labs/mcp-go/server"
)

var sdPackDomainAgentTypes = map[string]struct{}{
	"backend":      {},
	"frontend":     {},
	"devops":       {},
	"database":     {},
	"security":     {},
	"architecture": {},
	"code-review":  {},
	"rust":         {},
	"sre":          {},
	"mobile":       {},
	"data-ml":      {},
}

func isSDPackDomainAgentType(agentType string) bool {
	_, ok := sdPackDomainAgentTypes[strings.ToLower(strings.TrimSpace(agentType))]
	return ok
}

func sdPackMCPSidecarActive() bool {
	mgr := packsidecar.GlobalManager()
	if mgr == nil {
		return false
	}
	if !mgr.MCPSidecarActive(config.PackSoftwareDevelopment) {
		return false
	}
	inst := mgr.InstanceForPack(config.PackSoftwareDevelopment)
	if inst == nil || inst.BaseURL == "" {
		return false
	}
	return packremote.SidecarHealthOK(inst.BaseURL)
}

// workspaceOnlyMCP is an in-process MCP host used when the software-development
// pack sidecar is unavailable so specialists still get read_file / list_dir.
type workspaceOnlyMCP struct {
	mcpServer *server.MCPServer
}

func newWorkspaceOnlyMCP(label string) (*workspaceOnlyMCP, error) {
	name := fmt.Sprintf("%s-workspace-mcp", strings.ToLower(strings.ReplaceAll(strings.TrimSpace(label), " ", "-")))
	if name == "-workspace-mcp" {
		name = "specialist-workspace-mcp"
	}
	srv, err := mcp.NewInProcessMCPServer(name, "1.0.0")
	if err != nil {
		return nil, err
	}
	return &workspaceOnlyMCP{mcpServer: srv}, nil
}

func (w *workspaceOnlyMCP) GetMCPServer() *server.MCPServer {
	if w == nil {
		return nil
	}
	return w.mcpServer
}

func (w *workspaceOnlyMCP) Start() error { return nil }

// attachSDDomainMCP wires pack sidecar MCP when available, otherwise runs localFn for in-core MCP.
// When neither sidecar nor localFn is available but attachWorkspace is true, attaches a
// workspace-only MCP so file tools keep working (avoids "tool read_file not found" loops).
func attachSDDomainMCP(agent *Agent, agentType, label string, attachWorkspace bool, localFn func() (MCPServerInterface, error)) {
	if agent == nil {
		return
	}
	if isSDPackDomainAgentType(agentType) && sdPackMCPSidecarActive() {
		remote, err := packremote.NewRemoteMCP(agentType)
		if err != nil {
			log.Printf("Failed to create pack remote MCP for %s: %v", label, err)
		} else {
			startAgentMCPWithOptions(agent, label, remote, attachWorkspace)
			return
		}
	}
	if localFn != nil {
		srv, err := localFn()
		if err != nil {
			log.Printf("Failed to create %s MCP server: %v", label, err)
			return
		}
		startAgentMCPWithOptions(agent, label, srv, attachWorkspace)
		return
	}
	if !attachWorkspace {
		return
	}
	srv, err := newWorkspaceOnlyMCP(label)
	if err != nil {
		log.Printf("Failed to create workspace-only MCP for %s: %v", label, err)
		return
	}
	startAgentMCPWithOptions(agent, label, srv, true)
	log.Printf("Workspace-only MCP attached for %s (software-development sidecar unavailable)", label)
}
