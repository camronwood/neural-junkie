package hub

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/music"
	"github.com/camronwood/neural-junkie/internal/packs"
)

// ACEStepStatus reports local ACE-Step readiness for the Music creation pack.
func (h *Hub) ACEStepStatus() music.ACEStepStatus {
	if h == nil || h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return music.ACEStepStatus{}
	}
	return aceStepStatusForConfig(h.commandHandler.appConfig, config.PackMusicCreation)
}

func aceStepStatusForConfig(cfg *config.Config, packID string) music.ACEStepStatus {
	if cfg == nil || !cfg.IsPackInstalled(packID) {
		return music.ACEStepStatus{}
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		return music.ACEStepStatusFromSettings(nil, "")
	}
	settings := map[string]string{}
	if m, err := cfg.InstalledPackManifestByID(packID); err == nil && m != nil {
		if resolved, err := packs.ResolveSettingsOverlay(m, dir); err == nil {
			settings = resolved
		}
	}
	for k, v := range cfg.MusicSidecarSettings() {
		if strings.TrimSpace(v) != "" {
			settings[k] = v
		}
	}
	return music.ACEStepStatusFromSettings(settings, dir)
}
