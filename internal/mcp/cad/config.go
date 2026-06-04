package cad

import (
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
)

func cadSettings() config.CadMCPConfig {
	if cfg := mcp.AppConfig(); cfg != nil {
		return cfg.CadMCPSettings()
	}
	return config.CadMCPConfig{}
}
