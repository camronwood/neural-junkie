package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/camronwood/neural-junkie/internal/arenasidecar"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/packs"
)

func handleArenaSidecarStatus(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if packID != config.PackModelArena {
		http.Error(w, "arena sidecar status is only available for the Model Arena pack", http.StatusBadRequest)
		return
	}
	if !requireArenaPackInstalled(w, packID) {
		return
	}
	st := chatHub.ArenaSidecarStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

func handleArenaSidecarInstall(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if packID != config.PackModelArena {
		http.Error(w, "arena sidecar install is only available for the Model Arena pack", http.StatusBadRequest)
		return
	}
	if !requireArenaPackInstalled(w, packID) {
		return
	}
	dir, err := packs.InstalledPackDir(packID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	settings := map[string]string{}
	if appConfig != nil {
		settings = appConfig.ArenaSidecarSettings()
		if m, err := appConfig.InstalledPackManifestByID(packID); err == nil && m != nil {
			if resolved, err := packs.ResolveSettingsOverlay(m, dir); err == nil {
				for k, v := range resolved {
					settings[k] = v
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	if err := arenasidecar.InstallSidecarDeps(ctx, dir, settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := restartPackSidecar(packID); err != nil {
		http.Error(w, "installed deps but sidecar restart failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"pack_id": packID,
		"sidecar": chatHub.ArenaSidecarStatus(),
	})
}

func handleArenaSidecarRestart(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	if packID != config.PackModelArena {
		http.Error(w, "sidecar restart is only available for the Model Arena pack", http.StatusBadRequest)
		return
	}
	if appConfig == nil || !appConfig.IsPackEnabled(packID) {
		http.Error(w, "enable the Model Arena pack first", http.StatusBadRequest)
		return
	}
	if err := restartPackSidecar(packID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"pack_id": packID,
		"sidecar": chatHub.ArenaSidecarStatus(),
	})
}

func requireArenaPackInstalled(w http.ResponseWriter, packID string) bool {
	if appConfig == nil || !appConfig.IsPackInstalled(packID) {
		http.Error(w, "pack not installed", http.StatusBadRequest)
		return false
	}
	return true
}
