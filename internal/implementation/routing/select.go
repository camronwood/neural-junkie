// Package routing selects AI providers for implementation sessions (local-first).
package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
	semantic "github.com/camronwood/neural-junkie/internal/intent"
	unified "github.com/camronwood/neural-junkie/internal/routing"
	"github.com/camronwood/neural-junkie/internal/routing/capabilities"
)

const DefaultLocalToolModel = "qwen2.5-coder:14b"

// Input is the routing decision context for an implementation session.
type Input struct {
	RoutingEnabled                bool
	ModelCapabilityRoutingEnabled bool
	LocalProviderID               string
	LocalToolModel                string
	ReliableToolModel             string
	ReliableProviderID            string
	FallbackProviderIDs           []string
	LocalEscalationEnabled        bool
	FrontierEscalationEnabled     bool
	Providers                     []config.ProviderConfig
	DefaultProviderID             string
	TaskText                      string
	AgentType                     string
	RepairAttempts                int
	VerifyFailed                  bool
	BootFixIntent                 bool
	InstalledOllamaTags           map[string]struct{}
	OllamaTagToolFilter           func(tag string) bool
	SemanticDecision              *semantic.TurnDecision
}

// SelectProviderID returns provider id, tool-loop model tag, and reason code.
func SelectProviderID(in Input) (id string, toolModel string, reason string) {
	toolModel = resolveToolModel(in)
	if in.RepairAttempts >= 2 {
		rid := strings.TrimSpace(in.ReliableProviderID)
		if rid != "" && providerExists(in.Providers, rid) {
			if providerIsLocal(in.Providers, rid) && in.LocalEscalationEnabled {
				return rid, toolModel, "reliable_local_repair_tier"
			}
			if !providerIsLocal(in.Providers, rid) && in.FrontierEscalationEnabled {
				return rid, toolModel, "frontier_after_local_exhaustion"
			}
		}
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
		if fid != "" && providerExists(in.Providers, fid) &&
			(providerIsLocal(in.Providers, fid) || in.FrontierEscalationEnabled) {
			return fid, toolModel, "fallback_provider"
		}
	}

	if in.DefaultProviderID != "" && providerExists(in.Providers, in.DefaultProviderID) &&
		(providerIsLocal(in.Providers, in.DefaultProviderID) || in.FrontierEscalationEnabled) {
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
	if in.RepairAttempts >= 1 && in.LocalEscalationEnabled {
		if m := strings.TrimSpace(in.ReliableToolModel); m != "" {
			if len(in.InstalledOllamaTags) == 0 || installedTag(in.InstalledOllamaTags, m) {
				return pickToolCapableModel(in, m)
			}
		}
	}
	if in.ModelCapabilityRoutingEnabled {
		if p := capabilities.Global(); p != nil {
			_, toolClass := implementationClasses(in)
			if sel := capabilities.SelectOllamaTagWithFilter(p, toolClass, in.InstalledOllamaTags, fallback, implTagFilter(in, toolClass)); sel.Tag != "" {
				return sel.Tag
			}
		}
	}
	if in.SemanticDecision != nil && in.SemanticDecision.Complexity == "cheap" {
		return pickToolCapableModel(in, DefaultLocalToolModel)
	}
	if text := strings.TrimSpace(in.TaskText); text != "" && in.SemanticDecision == nil {
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
			mainClass, _ := implementationClasses(in)
			if sel := capabilities.SelectOllamaTagWithFilter(p, mainClass, in.InstalledOllamaTags, fallback, implTagFilter(in, mainClass)); sel.Tag != "" {
				return sel.Tag, sel.Reason
			}
		}
	}
	model := pickToolCapableModel(in, fallback)
	return model, "default_agent_provider"
}

func implementationClasses(in Input) (capabilities.TaskClass, capabilities.TaskClass) {
	if in.RepairAttempts >= 1 || in.VerifyFailed {
		return capabilities.TaskImplementHeavy, capabilities.TaskImplementHeavy
	}
	if in.SemanticDecision != nil {
		class := capabilities.ClassifySemantic(*in.SemanticDecision, false, true)
		return class, class
	}
	return capabilities.ClassifyImpl(capabilities.ImplInput{
		TaskText: in.TaskText, AgentType: in.AgentType,
		RepairAttempts: in.RepairAttempts, VerifyFailed: in.VerifyFailed,
		BootFixIntent: in.BootFixIntent,
	})
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

func providerIsLocal(providers []config.ProviderConfig, id string) bool {
	for _, p := range providers {
		if p.ID == id {
			return strings.EqualFold(strings.TrimSpace(p.Type), "ollama")
		}
	}
	return false
}

func installedTag(installed map[string]struct{}, tag string) bool {
	if _, ok := installed[tag]; ok {
		return true
	}
	base := strings.SplitN(tag, ":", 2)[0]
	for candidate := range installed {
		if candidate == tag || strings.HasPrefix(candidate, base+":") {
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
