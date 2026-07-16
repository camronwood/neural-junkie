package ai

import "testing"

func TestEstimateCostUSD_LocalZero(t *testing.T) {
	if got := EstimateCostUSD("ollama-default", "llama3.1", "standard", 10000, 2000); got != 0 {
		t.Fatalf("local provider cost = %v, want 0", got)
	}
}

func TestEstimateCostUSD_ClaudeSonnet(t *testing.T) {
	got := EstimateCostUSD("anthropic", "claude-3-5-sonnet-20241022", "", 1_000_000, 1_000_000)
	want := 3.0 + 15.0
	if got < want-0.001 || got > want+0.001 {
		t.Fatalf("cost = %v, want ~%v", got, want)
	}
}

func TestEstimateCostUSD_TierFallback(t *testing.T) {
	got := EstimateCostUSD("openai-compat", "unknown-model", "cheap", 1_000_000, 0)
	if got < 0.14 || got > 0.16 {
		t.Fatalf("cheap tier fallback = %v", got)
	}
}
