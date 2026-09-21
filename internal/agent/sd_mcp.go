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
	return packMCPSidecarActive(config.PackSoftwareDevelopment)
}

func packMCPSidecarActive(packID string) bool {
	mgr := packsidecar.GlobalManager()
	if mgr == nil {
		return false
	}
	if !mgr.MCPSidecarActive(packID) {
		return false
	}
	inst := mgr.InstanceForPack(packID)
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
	attachPackOwnedMCP(agent, config.PackSoftwareDevelopment, agentType, label, attachWorkspace, localFn)
}

// attachPackOwnedMCP wires pack-owned MCP tools:
//  1. mcp-sidecar binary (SD pattern) via packremote.NewRemoteMCP
//  2. hub-sidecar tools.json catalog via packremote.NewHubPackMCP
//  3. optional localFn fallback
//  4. workspace-only MCP when attachWorkspace is set
func attachPackOwnedMCP(agent *Agent, packID, agentType, label string, attachWorkspace bool, localFn func() (MCPServerInterface, error)) {
	if agent == nil {
		return
	}
	packID = strings.TrimSpace(packID)
	agentType = strings.TrimSpace(agentType)

	if packID == config.PackSoftwareDevelopment && isSDPackDomainAgentType(agentType) && packMCPSidecarActive(packID) {
		remote, err := packremote.NewRemoteMCP(agentType)
		if err != nil {
			log.Printf("Failed to create pack remote MCP for %s: %v", label, err)
		} else {
			startAgentMCPWithOptions(agent, label, remote, attachWorkspace)
			return
		}
	}

	if packID != "" && packID != config.PackSoftwareDevelopment {
		if hub, err := packremote.NewHubPackMCP(packID, label); err == nil {
			startAgentMCPWithOptions(agent, label, hub, attachWorkspace)
			return
		} else {
			log.Printf("Pack hub MCP for %s (%s): %v", label, packID, err)
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
	log.Printf("Workspace-only MCP attached for %s (pack %s unavailable)", label, packID)
}
