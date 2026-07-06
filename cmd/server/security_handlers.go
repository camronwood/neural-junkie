package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
)

type systemSecuritySnapshot struct {
	HubTokenConfigured  bool   `json:"hub_token_configured"`
	AuthRequired        bool   `json:"auth_required"`
	RelaxedLocal        bool   `json:"relaxed_local"`
	BootstrapConfigured bool   `json:"bootstrap_configured"`
	ListenAll           bool   `json:"listen_all"`
	LoopbackOnly        bool   `json:"loopback_only"`
	ConfigKeyManaged    bool   `json:"config_key_managed"`
	Security            any    `json:"security,omitempty"`
	Server              any    `json:"server,omitempty"`
	Session             any    `json:"session,omitempty"`
	Debug               any    `json:"debug,omitempty"`
	MCPResources        any    `json:"mcp_resources,omitempty"`
	Automation          any    `json:"automation,omitempty"`
	CLIAgents           any    `json:"cli_agents,omitempty"`
	ImageGen            any    `json:"image_gen,omitempty"`
	SessionSummary      any    `json:"session_summary,omitempty"`
}

func securitySnapshot() systemSecuritySnapshot {
	srv := appConfig.ResolvedServer()
	return systemSecuritySnapshot{
		HubTokenConfigured:  hub.HubTokenConfigured(),
		AuthRequired:        hub.AuthRequired(),
		RelaxedLocal:        hub.RelaxedLocal(),
		BootstrapConfigured: hub.BootstrapConfigured(),
		ListenAll:           srv.ListenAll,
		LoopbackOnly:        !srv.ListenAll,
		ConfigKeyManaged:    true,
		Security:            appConfig.Redacted().Security,
		Server:              appConfig.Redacted().Server,
		Session:             appConfig.Session,
		Debug:               appConfig.Debug,
		MCPResources:        appConfig.MCPResources,
		Automation:          appConfig.Automation,
		CLIAgents:           appConfig.CLIAgents,
		ImageGen:            appConfig.Redacted().ImageGen,
		SessionSummary:      appConfig.SessionSummary,
	}
}

func handleSystemSecurity(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(securitySnapshot())
	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var body struct {
			Security       *config.SecurityConfig       `json:"security"`
			Server         *config.ServerConfig         `json:"server"`
			Session        *config.SessionConfig        `json:"session"`
			Debug          *config.DebugSettings        `json:"debug"`
			MCPResources   *config.MCPResourcesConfig   `json:"mcp_resources"`
			Automation     *config.AutomationConfig     `json:"automation"`
			CLIAgents      *config.CLIAgentsConfig      `json:"cli_agents"`
			ImageGen       *config.ImageGenSettings     `json:"image_gen"`
			SessionSummary *config.SessionSummaryConfig `json:"session_summary"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		prevBaseline := appConfig.CaptureSettingsRestartBaseline()
		if body.Security != nil {
			in := *body.Security
			config.PreserveRedactedSecrets(&config.Config{Security: in}, appConfig)
			if isRedactedPlaceholder(in.HubToken) {
				in.HubToken = appConfig.Security.HubToken
			}
			if isRedactedPlaceholder(in.FullMetadataSecret) {
				in.FullMetadataSecret = appConfig.Security.FullMetadataSecret
			}
			appConfig.Security = in
		}
		if body.Server != nil {
			appConfig.Server = *body.Server
		}
		if body.Session != nil {
			appConfig.Session = *body.Session
		}
		if body.Debug != nil {
			appConfig.Debug = *body.Debug
		}
		if body.MCPResources != nil {
			appConfig.MCPResources = *body.MCPResources
		}
		if body.Automation != nil {
			appConfig.Automation = *body.Automation
		}
		if body.CLIAgents != nil {
			appConfig.CLIAgents = *body.CLIAgents
		}
		if body.ImageGen != nil {
			in := *body.ImageGen
			if isRedactedPlaceholder(in.OpenAIAPIKey) {
				in.OpenAIAPIKey = appConfig.ImageGen.OpenAIAPIKey
			}
			appConfig.ImageGen = in
		}
		if body.SessionSummary != nil {
			appConfig.SessionSummary = *body.SessionSummary
		}
		applyRuntimeConfigSideEffects(nil)
		if err := appConfig.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		config.SetAppConfig(appConfig)
		restartReasons := config.SettingsRestartReasons(prevBaseline, appConfig)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":           "saved",
			"requires_restart": len(restartReasons) > 0,
			"restart_reasons":  restartReasons,
			"snapshot":         securitySnapshot(),
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func isRedactedPlaceholder(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "***" || strings.Contains(s, "...")
}
