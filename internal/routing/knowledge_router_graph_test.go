package routing

import "testing"

func TestMatchesCodeGraphCue(t *testing.T) {
	plan := PlanKnowledgeRoute("How does Hub relate to CommandHandler?")
	if !plan.Has(RouteCodeGraph) {
		t.Fatalf("expected code_graph in plan, got %#v", plan)
	}
	if !plan.Has(RouteCodebase) {
		t.Fatalf("expected codebase implied by graph cue, got %#v", plan)
	}
}

func TestPathBetweenCue(t *testing.T) {
	plan := PlanKnowledgeRoute("show the path between Analyzer and Storage")
	if !plan.Has(RouteCodeGraph) {
		t.Fatalf("expected code_graph, got %#v", plan)
	}
}
