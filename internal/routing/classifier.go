package routing

import (
	"context"
	"strings"
)

// Classifier classifies routing input into a decision.
type Classifier interface {
	Classify(ctx context.Context, in Input) (RoutingDecision, error)
}

// Options configures the default router.
type Options struct {
	Classifier       string  // llm | rules
	ClassifierModel  string
	RulesFallback    bool
	MinConfidence    float64
	LLMClassifier    Classifier
}

// DefaultOptions returns LLM-first routing with rules fallback.
func DefaultOptions() Options {
	return Options{
		Classifier:      "llm",
		ClassifierModel: "qwen3.5:9b",
		RulesFallback:   true,
		MinConfidence:   0.6,
	}
}

// Classify runs the configured classifier with optional rules fallback.
func Classify(ctx context.Context, in Input, opts Options) RoutingDecision {
	opts = opts.withDefaults()
	rules := ClassifyRules(in)

	useLLM := strings.EqualFold(strings.TrimSpace(opts.Classifier), "llm") && opts.LLMClassifier != nil
	if !useLLM {
		return rules
	}

	dec, err := opts.LLMClassifier.Classify(ctx, in)
	if err != nil || dec.Domain == "" {
		if opts.RulesFallback {
			return rules
		}
		return rules
	}
	dec = dec.Normalized()
	if dec.Confidence < opts.MinConfidence && opts.RulesFallback {
		return rules
	}
	if dec.Source == "" {
		dec.Source = SourceLLM
	}
	if dec.LoRATag == "" {
		dec.LoRATag, _ = selectLoRATag(in)
	}
	if dec.Reason == "" {
		dec.Reason = "llm_classified"
	}
	return dec
}

func (o Options) withDefaults() Options {
	out := o
	if strings.TrimSpace(out.Classifier) == "" {
		out.Classifier = "llm"
	}
	if strings.TrimSpace(out.ClassifierModel) == "" {
		out.ClassifierModel = "qwen3.5:9b"
	}
	if out.MinConfidence <= 0 {
		out.MinConfidence = 0.6
	}
	return out
}

// RulesOnlyClassifier wraps ClassifyRules for the Classifier interface.
type RulesOnlyClassifier struct{}

func (RulesOnlyClassifier) Classify(_ context.Context, in Input) (RoutingDecision, error) {
	return ClassifyRules(in), nil
}
