package config

import "strings"

// RoutingConfig controls the unified task routing classifier.
type RoutingConfig struct {
	Classifier      string  `json:"classifier"` // llm | rules
	ClassifierModel string  `json:"classifier_model,omitempty"`
	RulesFallback   bool    `json:"rules_fallback"`
	MinConfidence   float64 `json:"min_confidence"`
}

// DefaultRoutingConfig returns LLM-first routing defaults.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		Classifier:      "llm",
		ClassifierModel: UtilityOllamaModel,
		RulesFallback:   true,
		MinConfidence:   0.6,
	}
}

// Normalized returns routing settings with defaults filled in.
func (r RoutingConfig) Normalized() RoutingConfig {
	out := r
	if strings.TrimSpace(out.Classifier) == "" {
		out.Classifier = "llm"
	}
	if strings.TrimSpace(out.ClassifierModel) == "" {
		out.ClassifierModel = UtilityOllamaModel
	}
	if out.MinConfidence <= 0 {
		out.MinConfidence = 0.6
	}
	return out
}
