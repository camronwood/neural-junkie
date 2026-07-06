package packs

import "testing"

func TestLoRAPolicyResolvedDefaults(t *testing.T) {
	p := LoRAPolicy{}.Resolved()
	if p.SuggestAfterTurns != 10 {
		t.Fatalf("suggest: %d", p.SuggestAfterTurns)
	}
	if p.RefreshAfterDelta != 20 {
		t.Fatalf("refresh: %d", p.RefreshAfterDelta)
	}
	if p.EvalMinScore != 0.35 {
		t.Fatalf("eval: %v", p.EvalMinScore)
	}
}

func TestLoRAPolicyResolvedCustom(t *testing.T) {
	p := LoRAPolicy{SuggestAfterTurns: 5, EvalMinScore: 0.5}.Resolved()
	if p.SuggestAfterTurns != 5 || p.EvalMinScore != 0.5 {
		t.Fatalf("custom policy not preserved: %+v", p)
	}
}
