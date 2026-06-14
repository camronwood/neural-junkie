package config

import "testing"

func TestPerformanceDefaults(t *testing.T) {
	p := PerformanceConfig{}
	if p.ContextBudgetBytes() != defaultContextBudgetKB*1024 {
		t.Fatalf("context budget = %d", p.ContextBudgetBytes())
	}
	if p.IdeContextBudgetBytes() != defaultIdeContextBudgetKB*1024 {
		t.Fatalf("ide budget = %d", p.IdeContextBudgetBytes())
	}
	if p.ImplSessionBudgetBytes() != defaultImplSessionBudgetKB*1024 {
		t.Fatalf("impl budget = %d", p.ImplSessionBudgetBytes())
	}
	if p.MaxHistoryMessagesOrDefault() != defaultMaxHistoryMessages {
		t.Fatalf("history = %d", p.MaxHistoryMessagesOrDefault())
	}
}

func TestPerformanceCustomValues(t *testing.T) {
	p := PerformanceConfig{
		ContextBudgetKB:     24,
		IdeContextBudgetKB:  40,
		ImplSessionBudgetKB: 56,
		MaxHistoryMessages:  6,
	}
	if p.ContextBudgetBytes() != 24*1024 {
		t.Fatalf("context budget = %d", p.ContextBudgetBytes())
	}
	if p.MaxHistoryMessagesOrDefault() != 6 {
		t.Fatalf("history = %d", p.MaxHistoryMessagesOrDefault())
	}
}

func TestPerformanceMaxHistoryClampsHigh(t *testing.T) {
	p := PerformanceConfig{MaxHistoryMessages: 999}
	if p.MaxHistoryMessagesOrDefault() != defaultMaxHistoryMessages {
		t.Fatalf("expected default cap, got %d", p.MaxHistoryMessagesOrDefault())
	}
}
