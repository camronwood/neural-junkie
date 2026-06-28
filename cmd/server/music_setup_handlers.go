package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/music"
	"github.com/camronwood/neural-junkie/internal/packs"
)

func handleMusicACEStepStatus(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if packID != config.PackMusicCreation {
		http.Error(w, "ACE-Step status is only available for the Music creation pack", http.StatusBadRequest)
		return
	}
	if !requireMusicPackInstalled(w, packID) {
		return
	}
	st := musicACEStepStatus(packID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func handleMusicACEStepInstall(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if packID != config.PackMusicCreation {
		http.Error(w, "ACE-Step install is only available for the Music creation pack", http.StatusBadRequest)
		return
	}
	if !requireMusicPackInstalled(w, packID) {
		return
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Minute)
	defer cancel()
	if err := music.InstallACEStep(ctx, dir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	syncPackSidecars()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pack_id": packID,
		"acestep": musicACEStepStatus(packID),
	})
}

func requireMusicPackInstalled(w http.ResponseWriter, packID string) bool {
	if appConfig == nil || !appConfig.IsPackInstalled(packID) {
		http.Error(w, "pack not installed", http.StatusBadRequest)
		return false
	}
	return true
}

func musicACEStepStatus(packID string) music.ACEStepStatus {
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		return music.ACEStepStatusFromSettings(nil, "")
	}
	settings := map[string]string{}
	if m, err := appConfig.InstalledPackManifestByID(packID); err == nil && m != nil {
		if resolved, err := packs.ResolveSettingsOverlay(m, dir); err == nil {
			settings = resolved
		}
	}
	return music.ACEStepStatusFromSettings(settings, dir)
}
