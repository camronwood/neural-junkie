package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestRecordTurnGovernance(t *testing.T) {
	a := &Agent{}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "hello")
	msg.Metadata = map[string]interface{}{
		protocol.TurnMetaComposerMode: "ask",
		"context_scope":               "channel",
	}
	a.recordTurnGovernance(msg)
	snap := a.LastRoutingSnapshot()
	if snap.ComposerMode != "ask" || snap.ContextScope != "channel" {
		t.Fatalf("governance snap = %+v", snap)
	}
	if snap.ImplSession {
		t.Fatal("expected impl session false")
	}
}

func TestRecordClassifierRoutingSkipsCollabTask(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend}}
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "#collab", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "fix typo")
	a.recordClassifierRouting(msg)
	snap := a.LastRoutingSnapshot()
	if snap.CostTier != "" || snap.Domain != "" {
		t.Fatalf("collab task should not classify chat routing: %+v", snap)
	}
}

func TestRecordClassifierRoutingSetsTier(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "fix typo in README")
	a.recordClassifierRouting(msg)
	snap := a.LastRoutingSnapshot()
	if snap.CostTier != "cheap" {
		t.Fatalf("cost tier = %q, want cheap", snap.CostTier)
	}
}
