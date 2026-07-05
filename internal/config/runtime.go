package config

import "sync"

var (
	appConfigMu sync.RWMutex
	appConfigGlobal *Config
)

// SetAppConfig updates the process-wide config snapshot (call after load and on every save).
func SetAppConfig(c *Config) {
	appConfigMu.Lock()
	defer appConfigMu.Unlock()
	appConfigGlobal = c
}

// AppConfig returns the live hub config or a default when unset (tests).
func AppConfig() *Config {
	appConfigMu.RLock()
	defer appConfigMu.RUnlock()
	if appConfigGlobal != nil {
		return appConfigGlobal
	}
	return DefaultConfig()
}
