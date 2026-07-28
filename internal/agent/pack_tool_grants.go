package agent

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// packToolGrantedToAgent reports whether Composition Model granted the named
// custom expert access to the given pack capability id (e.g. maps-tools).
func packToolGrantedToAgent(agentName, capabilityID string) bool {
	cfg := config.AppConfig()
	if cfg == nil {
		return false
	}
	return cfg.PackToolGrantedTo(agentName, capabilityID)
}

// musicPackToolGranted is true when a custom expert was granted music-generation.
func musicPackToolGranted(agentName string) bool {
	return packToolGrantedToAgent(agentName, "music-generation")
}

func agentDisplayNameMatches(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
