package eval

import "testing"

func TestDefaultSpecialistQuestions(t *testing.T) {
	q := DefaultSpecialistQuestions("cad")
	if len(q) < 2 {
		t.Fatalf("expected cad probes, got %d", len(q))
	}
}

func TestQuestionsForAgentTypeFallback(t *testing.T) {
	q := QuestionsForAgentType("unknown", "agent-1", "", "")
	if len(q) == 0 {
		t.Fatal("expected fallback questions")
	}
}
