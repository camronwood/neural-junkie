package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
	st := chatHub.ACEStepStatus()
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
	var body struct {
		ModelVariant string `json:"model_variant"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	variant := music.NormalizeModelVariant(body.ModelVariant)
	if appConfig != nil {
		cfgVariant := appConfig.MusicMCPSettings().ModelVariant
		if variant == "sft" && strings.TrimSpace(body.ModelVariant) == "" && cfgVariant != "" {
			variant = music.NormalizeModelVariant(cfgVariant)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Minute)
	defer cancel()
	if err := music.InstallACEStep(ctx, dir, variant); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	syncPackSidecars()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pack_id": packID,
		"acestep": chatHub.ACEStepStatus(),
	})
}

func musicACEStepStatus(packID string) music.ACEStepStatus {
	if chatHub != nil {
		return chatHub.ACEStepStatus()
	}
	return music.ACEStepStatus{}
}

func handleMusicSidecarRestart(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if packID != config.PackMusicCreation {
		http.Error(w, "sidecar restart is only available for the Music creation pack", http.StatusBadRequest)
		return
	}
	if appConfig == nil || !appConfig.IsPackEnabled(packID) {
		http.Error(w, "enable the Music creation pack first", http.StatusBadRequest)
		return
	}
	syncPackSidecars()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"pack_id": packID,
		"acestep": chatHub.ACEStepStatus(),
	})
}

func requireMusicPackInstalled(w http.ResponseWriter, packID string) bool {
	if appConfig == nil || !appConfig.IsPackInstalled(packID) {
		http.Error(w, "pack not installed", http.StatusBadRequest)
		return false
	}
	return true
}
