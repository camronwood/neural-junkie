package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handlePacksRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/packs")
	path = strings.Trim(path, "/")
	if path == "" {
		handlePacksList(w, r)
		return
	}
	handlePackByID(w, r, path)
}

func handlePacksList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appConfig.ListPackStatus())
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePackByID(w http.ResponseWriter, r *http.Request, packID string) {
	switch r.Method {
	case http.MethodPut:
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := appConfig.SetPackEnabled(packID, body.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		syncMCPFromConfig()
		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}
		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		reconcileConfiguredSpecialists()
		initializeConfiguredAgents()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"pack_id": packID,
			"enabled": body.Enabled,
			"packs":   appConfig.ListPackStatus(),
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleExpertPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appConfig.AvailableExpertPresets())
}
