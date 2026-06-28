package main

import (
	"encoding/json"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

func handlePackUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	updates, err := appConfig.ListPackUpdates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if updates == nil {
		updates = []config.PackUpdateInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"updates": updates,
		"count":   len(updates),
	})
}

func handlePackUpgrade(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := ensureMutationAccess(w, r, ""); !ok {
		return
	}
	wasEnabled, err := appConfig.UpgradePack(packID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if wasEnabled {
		syncMCPFromConfig()
		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}
		reconcileConfiguredSpecialists()
		initializeConfiguredAgents()
		triggerEnsurePackLoRAs(packID)
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, packID, map[string]any{
		"upgraded":    true,
		"was_enabled": wasEnabled,
	})
}
