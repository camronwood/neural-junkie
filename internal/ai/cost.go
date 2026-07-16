package ai

import "strings"

// ModelRates holds per-million-token pricing in USD.
type ModelRates struct {
	InputPerM  float64
	OutputPerM float64
}

// defaultRates maps model name substrings to rough public API pricing (USD per 1M tokens).
// Local models (Ollama/LM Studio) return zero cost.
var defaultRates = []struct {
	substr string
	rates  ModelRates
}{
	{"claude-opus", ModelRates{15, 75}},
	{"claude-3-5-sonnet", ModelRates{3, 15}},
	{"claude-sonnet", ModelRates{3, 15}},
	{"claude-haiku", ModelRates{0.25, 1.25}},
	{"gpt-4o-mini", ModelRates{0.15, 0.6}},
	{"gpt-4o", ModelRates{2.5, 10}},
	{"gpt-4", ModelRates{30, 60}},
	{"o1-mini", ModelRates{3, 12}},
	{"o1", ModelRates{15, 60}},
	{"o3-mini", ModelRates{1.1, 4.4}},
}

// tierFallbackRates applies when model name is unknown but cost_tier is set (cloud heuristics).
var tierFallbackRates = map[string]ModelRates{
	"cheap":    {0.15, 0.6},
	"standard": {3, 15},
	"premium":  {15, 75},
}

// IsLocalProvider reports providers that typically have no per-token bill.
func IsLocalProvider(providerID string) bool {
	p := strings.ToLower(strings.TrimSpace(providerID))
	if p == "" {
		return false
	}
	return strings.Contains(p, "ollama") ||
		strings.Contains(p, "lmstudio") ||
		strings.Contains(p, "lm-studio") ||
		strings.Contains(p, "local")
}

// LookupModelRates returns pricing for a model name, optionally falling back to cost tier.
func LookupModelRates(model, costTier string) (ModelRates, bool) {
	m := strings.ToLower(strings.TrimSpace(model))
	for _, row := range defaultRates {
		if strings.Contains(m, row.substr) {
			return row.rates, true
		}
	}
	tier := strings.ToLower(strings.TrimSpace(costTier))
	if r, ok := tierFallbackRates[tier]; ok {
		return r, true
	}
	return ModelRates{}, false
}

// EstimateCostUSD returns an approximate API spend for a turn (0 for local providers).
func EstimateCostUSD(providerID, model, costTier string, promptTokens, completionTokens int) float64 {
	if promptTokens == 0 && completionTokens == 0 {
		return 0
	}
	if IsLocalProvider(providerID) {
		return 0
	}
	rates, ok := LookupModelRates(model, costTier)
	if !ok {
		return 0
	}
	inCost := float64(promptTokens) / 1_000_000 * rates.InputPerM
	outCost := float64(completionTokens) / 1_000_000 * rates.OutputPerM
	return inCost + outCost
}
