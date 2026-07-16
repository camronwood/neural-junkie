package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func TestUsageFromMap(t *testing.T) {
	u := usageFromMap(map[string]interface{}{
		"prompt_tokens":     float64(1200),
		"completion_tokens": float64(88),
		"calls":             float64(2),
	})
	if u.PromptTokens != 1200 || u.CompletionTokens != 88 || u.Calls != 2 {
		t.Fatalf("usageFromMap = %+v", u)
	}
}

func TestCollectTurnUsageFromImplOutcome(t *testing.T) {
	a := &Agent{}
	st := &turnState{
		implSessionOutcome: map[string]interface{}{
			"inference_usage": map[string]interface{}{
				"prompt_tokens":     500,
				"completion_tokens": 100,
			},
		},
	}
	u := a.collectTurnUsage(st)
	if u.PromptTokens != 500 || u.CompletionTokens != 100 {
		t.Fatalf("collectTurnUsage = %+v", u)
	}
}

func TestCollectTurnUsageEmpty(t *testing.T) {
	a := &Agent{}
	u := a.collectTurnUsage(&turnState{})
	if u != (ai.InferenceUsage{}) {
		t.Fatalf("expected empty usage, got %+v", u)
	}
}
