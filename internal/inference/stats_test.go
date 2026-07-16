package inference

import (
	"path/filepath"
	"testing"
)

func TestStatsStoreRecordAndSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	s, err := NewStatsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Record(TurnRecord{
		ProviderID:       "ollama",
		Model:            "qwen3.5:9b",
		PromptTokens:     1000,
		CompletionTokens: 200,
		Calls:            1,
	})
	s.Record(TurnRecord{
		ProviderID:       "anthropic",
		Model:            "claude-sonnet",
		PromptTokens:     500,
		CompletionTokens: 100,
		EstimatedCostUSD: 0.003,
		Calls:            1,
	})

	sum := s.Summary()
	if sum.Totals.PromptTokens != 1500 {
		t.Fatalf("prompt tokens = %d, want 1500", sum.Totals.PromptTokens)
	}
	if sum.Totals.CompletionTokens != 300 {
		t.Fatalf("completion tokens = %d, want 300", sum.Totals.CompletionTokens)
	}
	if len(sum.ByProvider) != 2 {
		t.Fatalf("by_provider len = %d, want 2", len(sum.ByProvider))
	}

	s2, err := NewStatsStore(path)
	if err != nil {
		t.Fatal(err)
	}
	sum2 := s2.Summary()
	if sum2.Totals.PromptTokens != 1500 {
		t.Fatalf("reload prompt tokens = %d", sum2.Totals.PromptTokens)
	}
}
