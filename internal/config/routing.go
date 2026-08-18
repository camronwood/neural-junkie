package config

import "strings"

// RoutingConfig controls the unified task routing classifier.
type RoutingConfig struct {
	Classifier      string  `json:"classifier"` // llm | rules
	ClassifierModel string  `json:"classifier_model,omitempty"`
	RulesFallback   bool    `json:"rules_fallback"`
	MinConfidence   float64 `json:"min_confidence"`
	// SemanticClassifierModel is the Ollama tag used only for per-turn semantic
	// intent (SendMessage path). Defaults to a small/fast model so classify stays
	// inside the timeout budget; task routing keeps ClassifierModel.
	SemanticClassifierModel string `json:"semantic_classifier_model,omitempty"`
	// SemanticClassifierTimeoutMS bounds semantic classify wait before policy
	// fallback. UI echo happens first, but HTTP send still waits here.
	SemanticClassifierTimeoutMS int `json:"semantic_classifier_timeout_ms,omitempty"`
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
	// SemanticPrepareDispatchEnabled enables /api/turn/prepare + dispatch.
	// Defaults to true; set false to force single-shot /api/send only.
	SemanticPrepareDispatchEnabled *bool `json:"semantic_prepare_dispatch_enabled,omitempty"`
	// SemanticTextGatesDisabled disables LooksLike* ResolvePolicy overrides for
	// dual-gate shadow evaluation (prefer NJ_SEMANTIC_TEXT_GATES=0).
	SemanticTextGatesDisabled bool `json:"semantic_text_gates_disabled,omitempty"`
}

// Cutover rollout (semantic intent):
// 1) Shadow: prepare/dispatch on by default; dual-gate via NJ_SEMANTIC_TEXT_GATES=0.
// 2) Measure: make semantic-eval + corpus_policy dual-gate disagreements.
// 3) Promote: keep SemanticPrepareDispatchEnabled unset/true.
// 4) Remove: SemanticRoutingLegacyRollback + gated LooksLike* after gates stay green.

// DefaultRoutingConfig returns LLM-first routing defaults.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		Classifier:                    "llm",
		ClassifierModel:               UtilityOllamaModel,
		RulesFallback:                 true,
		MinConfidence:                 0.6,
		SemanticClassifierModel:       SemanticClassifierOllamaModel,
		SemanticClassifierTimeoutMS:   12000,
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
	if strings.TrimSpace(out.SemanticClassifierModel) == "" {
		out.SemanticClassifierModel = SemanticClassifierOllamaModel
	}
	if out.SemanticClassifierTimeoutMS <= 0 {
		out.SemanticClassifierTimeoutMS = 12000
	}
	return out
}
