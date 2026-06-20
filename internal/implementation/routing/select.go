// Package routing selects AI providers for implementation sessions (local-first).
package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

const DefaultLocalToolModel = "qwen3.5:9b"

// Input is the routing decision context for an implementation session.
type Input struct {
	RoutingEnabled                bool
	ModelCapabilityRoutingEnabled bool
	LocalProviderID               string
	LocalToolModel                string
	FallbackProviderIDs           []string
	Providers                     []config.ProviderConfig
	DefaultProviderID             string
	TaskText                      string
	AgentType                     string
	InstalledOllamaTags           map[string]struct{}
}

// SelectProviderID returns provider id, tool-loop model tag, and reason code.
func SelectProviderID(in Input) (id string, toolModel string, reason string) {
	toolModel = resolveToolModel(in)
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

// SelectMainModel returns the Ollama chat model for an implementation session.
func SelectMainModel(in Input) (model string, reason string) {
	return resolveMainModel(in)
}

func resolveToolModel(in Input) string {
	fallback := strings.TrimSpace(in.LocalToolModel)
	if fallback == "" {
		fallback = DefaultLocalToolModel
	}
	if in.ModelCapabilityRoutingEnabled {
		if p := capabilities.Global(); p != nil {
			_, toolClass := capabilities.ClassifyImpl(capabilities.ImplInput{
				TaskText:  in.TaskText,
				AgentType: in.AgentType,
			})
			if sel := capabilities.SelectOllamaTag(p, toolClass, in.InstalledOllamaTags, fallback); sel.Tag != "" {
				return sel.Tag
			}
		}
	}
	if text := strings.TrimSpace(in.TaskText); text != "" {
		dec := unified.ClassifyRules(unified.Input{Text: text, AgentType: in.AgentType})
		if dec.CostTier == unified.CostCheap {
			return DefaultLocalToolModel
		}
	}
	return fallback
}

func resolveMainModel(in Input) (string, string) {
	fallback := strings.TrimSpace(in.LocalToolModel)
	if fallback == "" {
		fallback = DefaultLocalToolModel
	}
	if in.ModelCapabilityRoutingEnabled {
		if p := capabilities.Global(); p != nil {
			mainClass, _ := capabilities.ClassifyImpl(capabilities.ImplInput{
				TaskText:  in.TaskText,
				AgentType: in.AgentType,
			})
			if sel := capabilities.SelectOllamaTag(p, mainClass, in.InstalledOllamaTags, fallback); sel.Tag != "" {
				return sel.Tag, sel.Reason
			}
		}
	}
	return fallback, "default_agent_provider"
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
