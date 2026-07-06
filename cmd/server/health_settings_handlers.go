package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
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
		payload := settingsResponse()
		json.NewEncoder(w).Encode(payload)

	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
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
		if strings.Contains(incoming.Jira.APIToken, "...") || incoming.Jira.APIToken == "***" {
			incoming.Jira.APIToken = appConfig.Jira.APIToken
		}
		config.PreserveRedactedSecrets(&incoming, appConfig)

		prevMusic := appConfig.MCP.Music
		prevBaseline := appConfig.CaptureSettingsRestartBaseline()
		appConfig.Server = incoming.Server
		appConfig.AI = incoming.AI
		appConfig.Agents = incoming.Agents
		appConfig.Ollama = incoming.Ollama
		appConfig.HF = incoming.HF
		appConfig.Updates = incoming.Updates
		appConfig.Collaboration = incoming.Collaboration
		appConfig.Implementation = incoming.Implementation
		appConfig.Routing = incoming.Routing.Normalized()
		appConfig.Delegation = incoming.Delegation.Normalized()
		appConfig.Features = incoming.Features
		appConfig.Performance = incoming.Performance
		appConfig.Slack = incoming.Slack
		appConfig.WebSearch = incoming.WebSearch
		appConfig.Phoenix = incoming.Phoenix
		appConfig.WorkspaceIndex = incoming.WorkspaceIndex
		appConfig.SpecialistCompose = incoming.SpecialistCompose
		appConfig.Security = incoming.Security
		appConfig.Session = incoming.Session
		appConfig.SessionSummary = incoming.SessionSummary
		appConfig.ImageGen = incoming.ImageGen
		appConfig.CLIAgents = incoming.CLIAgents
		appConfig.MCPResources = incoming.MCPResources
		appConfig.Debug = incoming.Debug
		appConfig.Automation = incoming.Automation
		appConfig.Storage = incoming.Storage
		if incoming.Packs.Enabled != nil {
			if appConfig.Packs.Enabled == nil {
				appConfig.Packs.Enabled = make(map[string]bool)
			}
			for id, enabled := range incoming.Packs.Enabled {
				appConfig.Packs.Enabled[id] = enabled
			}
			if len(incoming.Packs.Installed) > 0 {
				appConfig.Packs.Installed = incoming.Packs.Installed
			}
			if incoming.Packs.LayoutOwner != "" {
				appConfig.Packs.LayoutOwner = incoming.Packs.LayoutOwner
			}
			if incoming.Packs.CatalogURL != "" {
				appConfig.Packs.CatalogURL = incoming.Packs.CatalogURL
			}
		}
		appConfig.MCP = incoming.MCP
		appConfig.AWS = incoming.AWS
		appConfig.Jira = incoming.Jira
		appConfig.EnsureMCPDefaults()
		appConfig.SyncAgentsFromPacks()
		syncMCPFromConfig()
		applyRuntimeConfigSideEffects(nil)

		globalProviderCache.Clear()
		if ch, ok := chatHub.GetCommandHandler().(*hub.CommandHandler); ok {
			ch.SetProviderRegistry(appConfig, globalProviderCache)
		}

		if err := appConfig.Save(); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		config.SetAppConfig(appConfig)

		syncMCPFromConfig()
		if musicSettingsChanged(prevMusic, appConfig.MCP.Music) {
			syncPackSidecars()
		}
		reconcileConfiguredSpecialists()

		restartReasons := config.SettingsRestartReasons(prevBaseline, appConfig)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":           "saved",
			"requires_restart": len(restartReasons) > 0,
			"restart_reasons":  restartReasons,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func settingsResponse() map[string]interface{} {
	redacted := appConfig.Redacted()
	out := map[string]interface{}{}
	data, err := json.Marshal(redacted)
	if err == nil {
		_ = json.Unmarshal(data, &out)
	}
	if p := capabilities.Global(); p != nil {
		out["capability_profiles"] = p.Status()
	}
	return out
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

func musicSettingsChanged(before, after config.MusicMCPConfig) bool {
	return before.Normalized() != after.Normalized()
}
