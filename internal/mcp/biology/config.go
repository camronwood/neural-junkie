package biology

import (
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
)

func biologySettings() config.BiologyMCPConfig {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.BiologyMCPSettings()
	}
	return config.BiologyMCPConfig{}
}
