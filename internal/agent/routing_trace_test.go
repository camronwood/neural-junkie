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
	snap := a.LastRoutingSnapshotFor(msg.ID)
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
	snap := a.LastRoutingSnapshotFor(msg.ID)
	if snap.CostTier != "" || snap.Domain != "" {
		t.Fatalf("collab task should not classify chat routing: %+v", snap)
	}
}

func TestRecordClassifierRoutingSetsTier(t *testing.T) {
	a := &Agent{Info: protocol.AgentInfo{Type: protocol.AgentTypeBackend}}
	msg := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "fix typo in README")
	a.recordClassifierRouting(msg)
	snap := a.LastRoutingSnapshotFor(msg.ID)
	if snap.CostTier != "cheap" {
		t.Fatalf("cost tier = %q, want cheap", snap.CostTier)
	}
}

func TestRoutingSnapshotsAreIsolatedPerTurn(t *testing.T) {
	a := &Agent{}
	msgA := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "a")
	msgB := protocol.NewMessage(protocol.MessageTypeChat, "#general", protocol.AgentInfo{ID: "u1", Name: "User", Type: "human"}, "b")
	a.beginTurnRouting(msgA.ID)
	a.beginTurnRouting(msgB.ID)
	a.RecordRoutingSnapshotFor(msgA.ID, RoutingSnapshot{Reason: "turn_a", Domain: "frontend"})
	a.RecordRoutingSnapshotFor(msgB.ID, RoutingSnapshot{Reason: "turn_b", Domain: "backend"})
	if got := a.LastRoutingSnapshotFor(msgA.ID); got.Reason != "turn_a" || got.Domain != "frontend" {
		t.Fatalf("turn A snap = %+v", got)
	}
	if got := a.LastRoutingSnapshotFor(msgB.ID); got.Reason != "turn_b" || got.Domain != "backend" {
		t.Fatalf("turn B snap = %+v", got)
	}
}
