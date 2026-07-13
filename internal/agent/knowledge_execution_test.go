package agent

import (
	"reflect"
	"testing"

	"github.com/camronwood/neural-junkie/internal/memory"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func TestShouldInjectMemory(t *testing.T) {
	closure := routing.KnowledgePlan{Reason: "closure_phrase"}
	if ShouldInjectMemory(closure) {
		t.Fatal("closure should skip memory")
	}
	mixed := routing.PlanKnowledgeRoute("what did we decide about auth, and where is it in the repo?")
	if !ShouldInjectMemory(mixed) {
		t.Fatal("mixed plan should inject memory")
	}
}

func TestMemorySourceFilter(t *testing.T) {
	collab := routing.PlanKnowledgeRoute("what did we decide in the collab?")
	got := MemorySourceFilter(collab)
	if len(got) != 0 {
		t.Fatalf("mixed memory+collab should not filter sources, got %v", got)
	}
	onlyCollab := routing.KnowledgePlan{
		Targets: []routing.RouteTarget{routing.RouteCollabArtifact},
	}
	got = MemorySourceFilter(onlyCollab)
	want := []memory.SourceType{memory.SourceCollabArtifact}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestShouldRunCodebaseSearch(t *testing.T) {
	plan := routing.PlanKnowledgeRoute("where is auth in the repo?")
	if !ShouldRunCodebaseSearch(plan) {
		t.Fatal("expected codebase search")
	}
}

func TestSkipKnowledgeRetrievalForMessage_collabOrchestration(t *testing.T) {
	handoff := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-abc",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments under collabs/<id>/.",
	)
	handoff.SetCollaborationID("collab-id")
	handoff.Metadata = map[string]interface{}{"collab_internal_event": true}
	if !skipKnowledgeRetrievalForMessage(handoff) {
		t.Fatal("expected collab orchestration message to skip knowledge retrieval")
	}
	human := protocol.NewMessage(
		protocol.MessageTypeChat,
		"collab-abc",
		protocol.AgentInfo{ID: "human-user", Name: "Camron", Type: protocol.AgentTypeGeneral},
		"Please focus only on README.md — keep the plan brief.",
	)
	human.SetCollaborationID("collab-id")
	if skipKnowledgeRetrievalForMessage(human) {
		t.Fatal("human steering during collab should not skip knowledge retrieval")
	}
	plan := (&Agent{}).effectiveKnowledgePlan(handoff, IntentSubstantive)
	if len(plan.Targets) != 0 || plan.Reason != "collab_turn" {
		t.Fatalf("expected empty collab plan, got %+v", plan)
	}
}
