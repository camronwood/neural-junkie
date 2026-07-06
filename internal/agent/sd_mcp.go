package agent

import (
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp/packremote"
	"github.com/camronwood/neural-junkie/internal/packsidecar"
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

// attachSDDomainMCP wires pack sidecar MCP when available, otherwise runs localFn for in-core MCP.
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
	if localFn == nil {
		return
	}
	srv, err := localFn()
	if err != nil {
		log.Printf("Failed to create %s MCP server: %v", label, err)
		return
	}
	startAgentMCPWithOptions(agent, label, srv, attachWorkspace)
}
