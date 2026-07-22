package config

import "strings"

// RoutingConfig controls the unified task routing classifier.
type RoutingConfig struct {
	Classifier      string  `json:"classifier"` // llm | rules
	ClassifierModel string  `json:"classifier_model,omitempty"`
	RulesFallback   bool    `json:"rules_fallback"`
	MinConfidence   float64 `json:"min_confidence"`
	// ModelCapabilityRoutingEnabled selects Ollama models from benchmark-derived profiles.
	ModelCapabilityRoutingEnabled bool `json:"model_capability_routing_enabled"`
	// CapabilityProfilesPath overrides the default model-capability-profiles.json path.
	CapabilityProfilesPath string `json:"capability_profiles_path,omitempty"`
	// LocalEscalationEnabled permits automatic retries on progressively more
	// capable installed local models. It defaults to enabled.
	LocalEscalationEnabled bool `json:"local_escalation_enabled"`
	// FrontierEscalationEnabled is explicit consent to send a failed local turn
	// to a configured non-local provider. Merely configuring that provider is
	// intentionally insufficient.
	FrontierEscalationEnabled bool `json:"frontier_escalation_enabled"`
	// SemanticRoutingLegacyRollback disables the canonical server semantic
	// router and restores legacy distributed inference for emergency rollback.
	SemanticRoutingLegacyRollback bool `json:"semantic_routing_legacy_rollback"`
}

// DefaultRoutingConfig returns LLM-first routing defaults.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		Classifier:                    "llm",
		ClassifierModel:               UtilityOllamaModel,
		RulesFallback:                 true,
		MinConfidence:                 0.6,
		ModelCapabilityRoutingEnabled: true,
		LocalEscalationEnabled:        true,
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
