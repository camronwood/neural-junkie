// Package routing selects a configured AI provider id for collaboration execution tasks.
package routing

import (
	"strings"

	"github.com/camronwood/neural-junkie/internal/config"
)

// Input is the routing decision context for a single collaboration task message.
type Input struct {
	TaskText            string
	HasUserImages       bool
	Providers           []config.ProviderConfig
	DefaultProviderID   string
}

// SelectProviderID returns a provider config id and a short reason code.
func SelectProviderID(in Input) (id string, reason string) {
	if in.DefaultProviderID == "" || len(in.Providers) == 0 {
		return in.DefaultProviderID, "no_providers_or_default"
	}

	text := strings.ToLower(in.TaskText)

	if in.HasUserImages {
		id := pickByTier(in.Providers, tierCost, true)
		if id != "" {
			return id, "vision_cheapest"
		}
		return in.DefaultProviderID, "vision_fallback_default"
	}

	if looksSecurity(text) {
		id := pickByTier(in.Providers, tierPremium, false)
		if id != "" {
			return id, "security_premium"
		}
		return in.DefaultProviderID, "security_fallback_default"
	}

	if looksCheap(text) && len(in.TaskText) < 1200 {
		id := pickByTier(in.Providers, tierCost, true)
		if id != "" && id != in.DefaultProviderID {
			return id, "cheap_local"
		}
		if id != "" {
			return id, "cheap_already_default"
		}
	}

	return in.DefaultProviderID, "default_agent_provider"
}

func looksSecurity(text string) bool {
	keywords := []string{
		"security", "auth", "oauth", "jwt", "encrypt", "crypt", "owasp",
		"penetration", "vulnerability", "cve", "compliance", "gdpr", "hipaa",
	}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func looksCheap(text string) bool {
	keywords := []string{
		"typo", "wording", "rephrase", "shorten", "grammar", "polish",
		"tweak", "rename", "comment", "whitespace", "formatting",
	}
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

// tierCost ranks provider types for low-cost routing (lower = cheaper).
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

// tierPremium ranks provider types for high-stakes routing (higher = more capable / premium).
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

// pickByTier selects a provider id by minimizing or maximizing tierFn over configured providers.
// minimize true -> pick lowest tier; false -> pick highest tier.
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
