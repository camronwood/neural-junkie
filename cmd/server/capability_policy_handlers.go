package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type capabilityPolicyAgentResponse struct {
	Agent protocol.AgentInfo          `json:"agent"`
	State config.AgentCapabilityState `json:"state"`
}

func handleCapabilityPolicy(w http.ResponseWriter, r *http.Request) {
	cfg := config.AppConfig()
	switch r.Method {
	case http.MethodGet:
		agents := chatHub.ListAgents()
		rows := make([]capabilityPolicyAgentResponse, 0, len(agents))
		for _, info := range agents {
			if info == nil {
				continue
			}
			rows = append(rows, capabilityPolicyAgentResponse{
				Agent: *info,
				State: cfg.ResolveAgentCapabilities(info.ID, string(info.Type), info.Name),
			})
		}
		writeCapabilityPolicyResponse(w, cfg, rows)
	case http.MethodPut, http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var req struct {
			AllowSensitiveByDefault *bool                           `json:"allow_sensitive_by_default,omitempty"`
			HandoffsEnabled         *bool                           `json:"handoffs_enabled,omitempty"`
			AgentKey                string                          `json:"agent_key,omitempty"`
			Override                *config.AgentCapabilityOverride `json:"override,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.AllowSensitiveByDefault != nil {
			cfg.SetCapabilityDefaults(*req.AllowSensitiveByDefault)
		}
		if req.HandoffsEnabled != nil {
			cfg.SetCapabilityHandoffsEnabled(*req.HandoffsEnabled)
		}
		if strings.TrimSpace(req.AgentKey) != "" {
			agentKey := strings.TrimSpace(req.AgentKey)
			for _, info := range chatHub.ListAgents() {
				if info != nil && info.ID == agentKey {
					agentKey = strings.ToLower(string(info.Type)) + ":" + strings.ToLower(info.Name)
					break
				}
			}
			override := config.AgentCapabilityOverride{}
			if req.Override != nil {
				override = *req.Override
			}
			if err := cfg.SetAgentCapabilityOverride(agentKey, override); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if req.AllowSensitiveByDefault == nil && req.HandoffsEnabled == nil && strings.TrimSpace(req.AgentKey) == "" {
			http.Error(w, "global default or agent override required", http.StatusBadRequest)
			return
		}
		if err := cfg.Save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		config.SetAppConfig(cfg)
		writeCapabilityPolicyResponse(w, cfg, nil)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeCapabilityPolicyResponse(w http.ResponseWriter, cfg *config.Config, agents []capabilityPolicyAgentResponse) {
	registry := cfg.ResolvedCapabilityRegistry()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"allow_sensitive_by_default": cfg.SensitiveCapabilitiesAllowedByDefault(),
		"handoffs_enabled":           cfg.CapabilityHandoffsEnabled(),
		"capability_registry":        registry.CapabilityRegistry,
		"agents":                     agents,
	})
}

func handleCapabilityHandoffs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"handoffs": chatHub.ListHandoffs(),
	})
}
