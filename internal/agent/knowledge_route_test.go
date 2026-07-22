package agent

import (
	"reflect"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestRecordKnowledgeRoute_mixedTargets(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "Assistant", Type: protocol.AgentTypeAssistant}}
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"#general",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"what did we decide about auth, and where is it in the repo?",
	)
	a.recordKnowledgeRoute(msg, IntentSubstantive)
	snap := a.LastRoutingSnapshotFor(msg.ID)
	want := []string{"conversation_memory", "codebase"}
	if !reflect.DeepEqual(snap.KnowledgeTargets, want) {
		t.Fatalf("targets = %v, want %v", snap.KnowledgeTargets, want)
	}
	if snap.KnowledgeReason != "mixed" {
		t.Fatalf("reason = %q, want mixed", snap.KnowledgeReason)
	}
}

func TestApplyRoutingMetadataKnowledgePlan(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Name: "Assistant", AIModel: "test-model"}}
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"#general",
		protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"},
		"thanks!",
	)
	a.recordKnowledgeRoute(msg, IntentClosure)
	a.recordKnowledgeExecutedFor(msg.ID, "conversation_memory")
	resp := protocol.NewMessage(protocol.MessageTypeAnswer, "#general", a.Info, "You're welcome!")
	a.ApplyRoutingMetadataToResponseFor(msg.ID, resp)

	meta := protocol.ExtractRoutingMeta(resp)
	if meta.KnowledgeReason != "closure_phrase" {
		t.Fatalf("reason = %q", meta.KnowledgeReason)
	}
	if len(meta.KnowledgeTargets) != 0 {
		t.Fatalf("closure should have no targets, got %v", meta.KnowledgeTargets)
	}
	if len(meta.KnowledgeExecuted) != 1 || meta.KnowledgeExecuted[0] != "conversation_memory" {
		t.Fatalf("executed = %v", meta.KnowledgeExecuted)
	}
}
