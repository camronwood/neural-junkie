package config

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/packs"
)

// ContextOrBackground returns a background context (hub has no request-scoped config ctx).
func (c *Config) ContextOrBackground() context.Context {
	return context.Background()
}

// CollectPackSidecarEnvs returns sidecar env for enabled packs that declare hub-sidecar capabilities.
func (c *Config) CollectPackSidecarEnvs() []packs.SidecarEnv {
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
		if !packs.PackNeedsSidecar(m) {
			continue
		}
		dir, err := packs.InstalledPackDir(id)
		if err != nil {
			continue
		}
		manifests = append(manifests, m)
		packDirs[m.ID] = dir
		resolved, _ := packs.ResolveSettingsOverlay(m, dir)
		if m.ID == PackMusicCreation {
			for k, v := range c.musicSidecarSettingsLocked() {
				if strings.TrimSpace(v) != "" {
					resolved[k] = v
				}
			}
		}
		if m.ID == PackAWS {
			for k, v := range c.awsSidecarSettingsLocked() {
				if strings.TrimSpace(v) != "" {
					resolved[k] = v
				}
			}
		}
		settings[m.ID] = resolved
	}
	return packs.CollectSidecarEnvs(manifests, packDirs, settings)
}
