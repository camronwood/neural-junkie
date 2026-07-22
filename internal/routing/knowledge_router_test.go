package routing

import (
	"reflect"
	"testing"

	"github.com/camronwood/neural-junkie/internal/intent"
)

func TestClassifyKnowledgeRoute(t *testing.T) {
	cases := []struct {
		in     string
		target RouteTarget
	}{
		{"thanks!", RouteGeneral},
		{"search @codebase for main", RouteCodebase},
		{"what did we decide in the collab?", RouteConversationMemory},
		{"remember when we talked about auth?", RouteConversationMemory},
		{"what did you say earlier about that message?", RoutePriorReference},
		{"hello team", RouteConversationMemory},
	}
	for _, tc := range cases {
		got := ClassifyKnowledgeRoute(tc.in)
		if got.Target != tc.target {
			t.Fatalf("Classify(%q) = %s, want %s", tc.in, got.Target, tc.target)
		}
	}
}

func TestPlanKnowledgeRoute_mixed(t *testing.T) {
	plan := PlanKnowledgeRoute("what did we decide about auth, and where is it in the repo?")
	want := []RouteTarget{RouteConversationMemory, RouteCodebase}
	if !reflect.DeepEqual(plan.Targets, want) {
		t.Fatalf("targets = %v, want %v", plan.Targets, want)
	}
	if plan.Reason != "mixed" {
		t.Fatalf("reason = %q, want mixed", plan.Reason)
	}
}

func TestPlanKnowledgeRoute_closure(t *testing.T) {
	plan := PlanKnowledgeRoute("thanks!")
	if len(plan.Targets) != 0 {
		t.Fatalf("closure should have no targets, got %v", plan.Targets)
	}
	if plan.Reason != "closure_phrase" {
		t.Fatalf("reason = %q, want closure_phrase", plan.Reason)
	}
}

func TestPlanKnowledgeRoute_collabOnly(t *testing.T) {
	plan := PlanKnowledgeRoute("show me collabs/plan.md summary")
	if !plan.Has(RouteCollabArtifact) {
		t.Fatalf("expected collab_artifact target, got %v", plan.Targets)
	}
}

func TestPlanKnowledgeRoute_priorAndMemory(t *testing.T) {
	plan := PlanKnowledgeRoute("what you said earlier about that message — was it right?")
	if !plan.Has(RoutePriorReference) {
		t.Fatalf("expected prior_reference, got %v", plan.Targets)
	}
}

func TestPlanKnowledgeRoute_empty(t *testing.T) {
	plan := PlanKnowledgeRoute("   ")
	if len(plan.Targets) != 0 || plan.Reason != "empty" {
		t.Fatalf("empty plan = %+v", plan)
	}
}

func TestPlanKnowledgeRoute_bootFailureUsesCodebase(t *testing.T) {
	plan := PlanKnowledgeRoute("Something is wrong with this code I am working on and the app will not boot up")
	if !plan.Has(RouteCodebase) {
		t.Fatalf("boot failure should retrieve codebase context, got %+v", plan)
	}
	if plan.Has(RouteConversationMemory) {
		t.Fatalf("first-turn boot failure should not default to conversation memory, got %+v", plan)
	}
}

func TestKnowledgePlan_Has(t *testing.T) {
	plan := PlanKnowledgeRoute("@codebase auth")
	if !plan.Has(RouteCodebase) || plan.Has(RouteCollabArtifact) {
		t.Fatal("Has mismatch")
	}
}

func TestPlanKnowledgeRouteForDecisionIgnoresTextPhrases(t *testing.T) {
	decision := intent.TurnDecision{
		SchemaVersion: intent.SchemaVersion, Interaction: intent.InteractionQuestion,
		RequestedAction: intent.ActionAnswer, Action: intent.ActionAnswer,
		Retrieval: []intent.RetrievalTarget{intent.RetrievalMemory},
		Mutation:  intent.MutationNone, Confidence: 1, Source: intent.SourceLocalModel,
	}
	plan := PlanKnowledgeRouteForDecision(decision)
	if !plan.Has(RouteConversationMemory) || plan.Has(RouteCodebase) || plan.Reason != "semantic_decision" {
		t.Fatalf("plan=%+v", plan)
	}
	// Phrase path would pull codebase for "@codebase"; decision path must not.
	phrase := PlanKnowledgeRoute("please create a canvas @codebase auth")
	if !phrase.Has(RouteCodebase) {
		t.Fatal("phrase fixture must match codebase cue")
	}
	if PlanKnowledgeRouteForDecision(decision).Has(RouteCodebase) {
		t.Fatal("decision plan must ignore @codebase phrases in user text")
	}
}
