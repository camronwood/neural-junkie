package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/arenasidecar"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/packs"
)

// ArenaSidecarStatus reports chess dependency readiness for the Model Arena pack.
func (h *Hub) ArenaSidecarStatus() arenasidecar.SidecarStatus {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return arenasidecar.SidecarStatus{}
	}
	return arenaSidecarStatusForConfig(h.commandHandler.appConfig, config.PackModelArena)
}

func arenaSidecarStatusForConfig(cfg *config.Config, packID string) arenasidecar.SidecarStatus {
	if cfg == nil || !cfg.IsPackInstalled(packID) {
		return arenasidecar.SidecarStatus{}
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		return arenasidecar.SidecarStatusFromSettings(nil, "")
	}
	settings := map[string]string{}
	if m, err := cfg.InstalledPackManifestByID(packID); err == nil && m != nil {
		if resolved, err := packs.ResolveSettingsOverlay(m, dir); err == nil {
			settings = resolved
		}
	}
	for k, v := range cfg.ArenaSidecarSettings() {
		if strings.TrimSpace(v) != "" {
			settings[k] = v
		}
	}
	return arenasidecar.SidecarStatusFromSettings(settings, dir)
}
