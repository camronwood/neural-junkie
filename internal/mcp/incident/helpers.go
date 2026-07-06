package incident

import (
	"fmt"
	"strings"

	mcp "github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/integrations/ticketing"
)

func writeAllowed() bool {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return false
	}
	return cfg.IncidentSettings().WriteModeEnabled()
}

func requireWrite(tool string) error {
	if writeAllowed() {
		return nil
	}
	return fmt.Errorf("%s: incident write mode is disabled — enable Allow ticket mutations in Settings → Integrations → Incident", tool)
}

func registry() (*ticketing.Registry, error) {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return nil, fmt.Errorf("hub config unavailable")
	}
	return ticketing.NewRegistry(cfg), nil
}

func providerFromRequest(provider string) (ticketing.Provider, error) {
	reg, err := registry()
	if err != nil {
		return nil, err
	}
	if p := strings.TrimSpace(provider); p != "" {
		return reg.Provider(p)
	}
	return reg.DefaultProvider()
}
