package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// ProviderPickInput is context for mapping a decision to a provider id.
type ProviderPickInput struct {
	Decision          RoutingDecision
	HasUserImages     bool
	Providers         []config.ProviderConfig
	DefaultProviderID string
	InstalledTags     map[string]struct{}
}

// PickProviderID maps a routing decision to a configured provider id and reason.
func PickProviderID(in ProviderPickInput) (id string, reason string) {
	dec := in.Decision.Normalized()
	if in.DefaultProviderID == "" || len(in.Providers) == 0 {
		return in.DefaultProviderID, "no_providers_or_default"
	}

	if in.HasUserImages {
		id := pickByTier(in.Providers, tierCost, true)
		if id != "" {
			return id, "vision_cheapest"
		}
		return in.DefaultProviderID, "vision_fallback_default"
	}

	if dec.Domain == DomainSecurity {
		if tagInstalled(in.InstalledTags, "nj-security:14b") {
			if id := pickProviderByType(in.Providers, "ollama"); id != "" {
				return id, "security_lora_local"
			}
		}
		if dec.CostTier == CostPremium {
			id := pickByTier(in.Providers, tierPremium, false)
			if id != "" {
				return id, "security_premium"
			}
		}
		return in.DefaultProviderID, "security_fallback_default"
	}

	if dec.CostTier == CostCheap && len(strings.TrimSpace(in.Decision.Reason)) > 0 {
		id := pickByTier(in.Providers, tierCost, true)
		if id != "" && id != in.DefaultProviderID {
			return id, "cheap_local"
		}
		if id != "" {
			return id, "cheap_already_default"
		}
	}

	if dec.Reason != "" {
		return in.DefaultProviderID, dec.Reason
	}
	return in.DefaultProviderID, "default_agent_provider"
}

func tierCost(pType string) int {
	switch strings.ToLower(strings.TrimSpace(pType)) {
	case "ollama":
		return 1
	case "openai-compatible":
		return 2
	case "anthropic":
		return 4
	case "cursor-cli", "gemini-cli":
		return 5
	default:
		return 99
	}
}

func tierPremium(pType string) int {
	switch strings.ToLower(strings.TrimSpace(pType)) {
	case "ollama":
		return 1
	case "openai-compatible":
		return 2
	case "cursor-cli", "gemini-cli":
		return 4
	case "anthropic":
		return 5
	default:
		return 0
	}
}

func pickByTier(providers []config.ProviderConfig, tierFn func(string) int, minimize bool) string {
	var bestID string
	var bestTier int
	first := true
	for _, p := range providers {
		if strings.TrimSpace(p.ID) == "" {
			continue
		}
		t := tierFn(p.Type)
		if t >= 99 {
			continue
		}
		if first {
			bestID, bestTier, first = p.ID, t, false
			continue
		}
		if minimize {
			if t < bestTier {
				bestID, bestTier = p.ID, t
			}
		} else {
			if t > bestTier {
				bestID, bestTier = p.ID, t
			}
		}
	}
	return bestID
}

func pickProviderByType(providers []config.ProviderConfig, pType string) string {
	pType = strings.ToLower(strings.TrimSpace(pType))
	for _, p := range providers {
		if strings.EqualFold(strings.TrimSpace(p.Type), pType) && strings.TrimSpace(p.ID) != "" {
			return p.ID
		}
	}
	return ""
}
