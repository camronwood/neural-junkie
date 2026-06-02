// Package routing selects AI providers for implementation sessions (local-first).
package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

const DefaultLocalToolModel = "qwen2.5-coder:7b"

// Input is the routing decision context for an implementation session.
type Input struct {
	RoutingEnabled      bool
	LocalProviderID     string
	LocalToolModel      string
	FallbackProviderIDs []string
	Providers           []config.ProviderConfig
	DefaultProviderID   string
}

// SelectProviderID returns provider id, tool-loop model tag, and reason code.
func SelectProviderID(in Input) (id string, toolModel string, reason string) {
	toolModel = strings.TrimSpace(in.LocalToolModel)
	if toolModel == "" {
		toolModel = DefaultLocalToolModel
	}
	if !in.RoutingEnabled || len(in.Providers) == 0 {
		if in.DefaultProviderID != "" {
			return in.DefaultProviderID, toolModel, "routing_disabled_default"
		}
		return "", toolModel, "no_providers"
	}

	localID := strings.TrimSpace(in.LocalProviderID)
	if localID == "" {
		localID = pickProviderByType(in.Providers, "ollama")
	}
	if localID != "" && providerExists(in.Providers, localID) {
		return localID, toolModel, "local_ollama_first"
	}

	for _, fid := range in.FallbackProviderIDs {
		fid = strings.TrimSpace(fid)
		if fid != "" && providerExists(in.Providers, fid) {
			return fid, toolModel, "fallback_provider"
		}
	}

	if in.DefaultProviderID != "" && providerExists(in.Providers, in.DefaultProviderID) {
		return in.DefaultProviderID, toolModel, "default_agent_provider"
	}
	return "", toolModel, "no_eligible_provider"
}

func providerExists(providers []config.ProviderConfig, id string) bool {
	for _, p := range providers {
		if p.ID == id {
			return true
		}
	}
	return false
}

func pickProviderByType(providers []config.ProviderConfig, typ string) string {
	for _, p := range providers {
		if strings.EqualFold(strings.TrimSpace(p.Type), typ) {
			return p.ID
		}
	}
	return ""
}
