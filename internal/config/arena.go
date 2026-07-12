package config

import (
	"os"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/arenasidecar"
)

// ArenaSidecarSettings returns overlay values merged into the model-arena sidecar env.
func (c *Config) ArenaSidecarSettings() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.arenaSidecarSettingsLocked()
}

func (c *Config) arenaSidecarSettingsLocked() map[string]string {
	venv := arenasidecar.ExpandHomePath("~/.neural-junkie/arena/venv")
	python := filepath.Join(venv, "bin", "python")
	out := map[string]string{
		"arena_venv": venv,
	}
	if info, err := os.Stat(python); err == nil && !info.IsDir() {
		out["python_executable"] = python
	}
	return out
}
