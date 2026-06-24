package config

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/packs"
	"github.com/camronwood/neural-junkie/internal/packs/sidecar"
)

// ContextOrBackground returns a background context (hub has no request-scoped config ctx).
func (c *Config) ContextOrBackground() context.Context {
	return context.Background()
}

// CollectPackSidecarEnvs returns sidecar env for enabled packs that declare hub-sidecar capabilities.
func (c *Config) CollectPackSidecarEnvs() []sidecar.SidecarEnv {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	var manifests []*packs.Manifest
	packDirs := make(map[string]string)
	settings := make(map[string]map[string]string)
	for _, id := range c.Packs.Installed {
		if !c.packEnabledLocked(id) {
			continue
		}
		m, err := c.installedManifestLocked(id)
		if err != nil || m == nil {
			continue
		}
		if !sidecar.PackNeedsSidecar(m) {
			continue
		}
		dir, err := packs.InstalledPackDir(id)
		if err != nil {
			continue
		}
		manifests = append(manifests, m)
		packDirs[m.ID] = dir
		resolved, _ := packs.ResolveSettingsOverlay(m, dir)
		settings[m.ID] = resolved
	}
	return sidecar.CollectSidecarEnvs(manifests, packDirs, settings)
}
