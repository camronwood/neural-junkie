// Package routing selects AI providers for implementation sessions (local-first).
package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
	unified "github.com/camronwood/neural-junkie/internal/routing"
)

const DefaultLocalToolModel = "qwen2.5-coder:14b"

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
	OllamaTagToolFilter           func(tag string) bool
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
	fallback := toolModelFallback(in)
	if in.ModelCapabilityRoutingEnabled {
		if p := capabilities.Global(); p != nil {
			_, toolClass := capabilities.ClassifyImpl(capabilities.ImplInput{
				TaskText:  in.TaskText,
				AgentType: in.AgentType,
			})
			if sel := capabilities.SelectOllamaTagWithFilter(p, toolClass, in.InstalledOllamaTags, fallback, implTagFilter(in, toolClass)); sel.Tag != "" {
				return sel.Tag
			}
		}
	}
	if text := strings.TrimSpace(in.TaskText); text != "" {
		dec := unified.ClassifyRules(unified.Input{Text: text, AgentType: in.AgentType})
		if dec.CostTier == unified.CostCheap {
			return pickToolCapableModel(in, DefaultLocalToolModel)
		}
	}
	return pickToolCapableModel(in, fallback)
}

func resolveMainModel(in Input) (string, string) {
	fallback := toolModelFallback(in)
	if in.ModelCapabilityRoutingEnabled {
		if p := capabilities.Global(); p != nil {
			mainClass, _ := capabilities.ClassifyImpl(capabilities.ImplInput{
				TaskText:  in.TaskText,
				AgentType: in.AgentType,
			})
			if sel := capabilities.SelectOllamaTagWithFilter(p, mainClass, in.InstalledOllamaTags, fallback, implTagFilter(in, mainClass)); sel.Tag != "" {
				return sel.Tag, sel.Reason
			}
		}
	}
	model := pickToolCapableModel(in, fallback)
	return model, "default_agent_provider"
}

func toolModelFallback(in Input) string {
	fallback := strings.TrimSpace(in.LocalToolModel)
	if fallback == "" {
		fallback = DefaultLocalToolModel
	}
	return fallback
}

func pickToolCapableModel(in Input, preferred string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		preferred = DefaultLocalToolModel
	}
	if in.OllamaTagToolFilter == nil || in.OllamaTagToolFilter(preferred) {
		return preferred
	}
	for _, candidate := range []string{DefaultLocalToolModel, preferred} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || candidate == preferred {
			continue
		}
		if in.OllamaTagToolFilter(candidate) {
			return candidate
		}
	}
	return preferred
}

func implTagFilter(in Input, class capabilities.TaskClass) func(string) bool {
	if in.OllamaTagToolFilter == nil || !capabilities.RequiresToolCapableModel(class) {
		return nil
	}
	return in.OllamaTagToolFilter
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
