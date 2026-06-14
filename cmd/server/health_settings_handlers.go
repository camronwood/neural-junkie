package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := chatHub.ListAgents()
	health := map[string]interface{}{
		"status":      "ok",
		"uptime_secs": int(time.Since(serverStartTime).Seconds()),
		"agent_count": len(agents),
		"version":     "1.0.0",
		"snapshot":    chatHub.GetSessionSaveHealth(),
		"features":    []string{"hub_data_read"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appConfig.Redacted())

	case http.MethodPut:
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Preserve API keys that are redacted in the incoming payload
		for i := range incoming.AI.Providers {
			ip := &incoming.AI.Providers[i]
			if strings.Contains(ip.APIKey, "...") || ip.APIKey == "***" {
				if existing := appConfig.GetProvider(ip.ID); existing != nil {
					ip.APIKey = existing.APIKey
				}
			}
		}
		if strings.Contains(incoming.HF.Token, "...") || incoming.HF.Token == "***" {
			incoming.HF.Token = appConfig.HF.Token
		}

		appConfig.Server = incoming.Server
		appConfig.AI = incoming.AI
		appConfig.Agents = incoming.Agents
		appConfig.Ollama = incoming.Ollama
		appConfig.HF = incoming.HF
		appConfig.Updates = incoming.Updates
		appConfig.Collaboration = incoming.Collaboration
		appConfig.Delegation = incoming.Delegation.Normalized()
		appConfig.Features = incoming.Features
		appConfig.Performance = incoming.Performance
		if incoming.Packs.Enabled != nil {
			appConfig.Packs = incoming.Packs
		}
		appConfig.MCP = incoming.MCP
		appConfig.EnsureMCPDefaults()
		appConfig.SyncAgentsFromPacks()
		syncMCPFromConfig()

		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}

		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		syncMCPFromConfig()
		reconcileConfiguredSpecialists()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleConfiguredAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appConfig.Agents)
}

func handleRestartAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	reconcileConfiguredSpecialists()
	// Re-run the configured agents initializer; existing agents keep running
	// (hub silently skips re-registration of duplicate IDs).
	initializeConfiguredAgents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "restarted"})
}
