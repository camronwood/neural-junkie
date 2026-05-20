package mcp

import (
	"sync"

	"github.com/camronwood/neural-junkie/internal/config"
)

var (
	appConfigMu sync.RWMutex
	appConfig   *config.Config
)

// SetAppConfig wires hub config into MCP enablement and biology tool limits.
func SetAppConfig(cfg *config.Config) {
	appConfigMu.Lock()
	appConfig = cfg
	appConfigMu.Unlock()
}

// AppConfig returns the current hub config snapshot pointer (may be nil).
func AppConfig() *config.Config {
	appConfigMu.RLock()
	defer appConfigMu.RUnlock()
	return appConfig
}
