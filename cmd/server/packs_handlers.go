package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/packs"
)

func handlePacksRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/packs")
	path = strings.Trim(path, "/")
	if path == "" {
		handlePacksList(w, r)
		return
	}
	if path == "catalog" {
		handlePacksCatalog(w, r)
		return
	}
	parts := strings.Split(path, "/")
	packID := parts[0]
	if len(parts) == 2 && parts[1] == "install" {
		handlePackInstall(w, r, packID)
		return
	}
	handlePackByID(w, r, packID)
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

func handlePacksCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := appConfig.ListPackCatalogStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"packs":       rows,
		"catalog_url": packs.CatalogURL(),
	})
}

func handlePackInstall(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := appConfig.InstallPack(packID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writePacksMutationResponse(w, packID, nil)
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
		writePacksMutationResponse(w, packID, map[string]any{"enabled": body.Enabled})
	case http.MethodDelete:
		if err := appConfig.UninstallPack(packID); err != nil {
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
		writePacksMutationResponse(w, packID, nil)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writePacksMutationResponse(w http.ResponseWriter, packID string, extra map[string]any) {
	st := appConfig.ListPackStatus()
	out := map[string]any{
		"status":         "ok",
		"pack_id":        packID,
		"packs":          st.Packs,
		"layout_owner":   st.LayoutOwner,
		"layout_profile": st.LayoutProfile,
		"capabilities":   st.Capabilities,
	}
	for k, v := range extra {
		out[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func handleExpertPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appConfig.AvailableExpertPresets())
}
