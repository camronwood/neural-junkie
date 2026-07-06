package packs

// LoRAPolicy configures specialist-tuning thresholds and gates (pack manifest lora_policy).
type LoRAPolicy struct {
	SuggestAfterTurns       int     `yaml:"suggest_after_turns,omitempty"`
	RefreshAfterDelta       int     `yaml:"refresh_after_delta,omitempty"`
	EvalMinScore            float64 `yaml:"eval_min_score,omitempty"`
	RequireEvalToAssign     bool    `yaml:"require_eval_to_assign,omitempty"`
	IncludeLearningsDefault bool    `yaml:"include_learnings_default,omitempty"`
	DefaultBase             string  `yaml:"default_base,omitempty"`
}

const (
	defaultSuggestAfterTurns = 10
	defaultRefreshAfterDelta = 20
	defaultEvalMinScore      = 0.35
)

// Resolved returns policy with pack defaults applied.
func (p LoRAPolicy) Resolved() LoRAPolicy {
	out := p
	if out.SuggestAfterTurns <= 0 {
		out.SuggestAfterTurns = defaultSuggestAfterTurns
	}
	if out.RefreshAfterDelta <= 0 {
		out.RefreshAfterDelta = defaultRefreshAfterDelta
	}
	if out.EvalMinScore <= 0 {
		out.EvalMinScore = defaultEvalMinScore
	}
	return out
}
