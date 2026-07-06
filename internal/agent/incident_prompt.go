package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func incidentPackPromptContext() string {
	dir, err := packs.InstalledPackDir(config.PackIncidentManagement)
	if err != nil {
		return ""
	}
	m, err := packs.LoadManifest(dir)
	if err != nil {
		return ""
	}
	return packs.IncidentPackContext(dir, m)
}

func appendIncidentPackContext(system *strings.Builder, agentType protocol.AgentType) {
	if agentType != protocol.AgentTypeIncident {
		return
	}
	if ctx := incidentPackPromptContext(); ctx != "" {
		system.WriteString(ctx)
	}
}
