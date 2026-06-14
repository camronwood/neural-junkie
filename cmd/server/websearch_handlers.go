package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/integrations/websearch"
)

func handleWebSearchConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		pub := map[string]interface{}{
			"enabled":     appConfig.WebSearch.Enabled,
			"provider":    appConfig.WebSearch.ProviderName(),
			"max_results": appConfig.WebSearch.MaxResultsOrDefault(),
			"api_key_set": appConfig.WebSearch.APIKey != "",
			"keyless":     appConfig.WebSearch.Keyless,
			"ready":       appConfig.WebSearch.Ready(),
		}
		writeJSON(w, http.StatusOK, pub)
	case http.MethodPut, http.MethodPost:
		var body struct {
			Enabled    *bool  `json:"enabled"`
			Provider   string `json:"provider"`
			APIKey     string `json:"api_key"`
			MaxResults *int   `json:"max_results"`
			Keyless    *bool  `json:"keyless"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Enabled != nil {
			appConfig.WebSearch.Enabled = *body.Enabled
		}
		if p := strings.TrimSpace(body.Provider); p != "" {
			appConfig.WebSearch.Provider = p
		}
		if t := strings.TrimSpace(body.APIKey); t != "" && !strings.Contains(t, "...") && t != "***" {
			appConfig.WebSearch.APIKey = t
		}
		if body.MaxResults != nil && *body.MaxResults > 0 {
			appConfig.WebSearch.MaxResults = *body.MaxResults
		}
		if body.Keyless != nil {
			appConfig.WebSearch.Keyless = *body.Keyless
		}
		if err := appConfig.Save(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		applyCollabActionConfig()
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWebSearchTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	client := websearch.NewClient(appConfig.WebSearch)
	results, err := client.Search(r.Context(), "Neural Junkie multi-agent", 1)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"results": results,
	})
}
