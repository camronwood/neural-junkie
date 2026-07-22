package gitchange

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestProposalCannotBeActedOnAfterTerminalState(t *testing.T) {
	manager := NewManager()
	proposal, err := manager.Propose(Proposal{
		Operation: OpCommit,
		Agent:     protocol.AgentInfo{ID: "agent-1"},
		Channel:   "general",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.MarkApplying(proposal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.MarkApproved(proposal.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.MarkApplying(proposal.ID); err == nil {
		t.Fatal("expected approved proposal to reject another action")
	}
	if _, err := manager.Reject(proposal.ID, "late rejection"); err == nil {
		t.Fatal("expected approved proposal to reject a late rejection")
	}
}

func TestDefaultPendingListIncludesAllAgentsInRequestOrder(t *testing.T) {
	manager := NewManager()
	first, _ := manager.Propose(Proposal{
		Operation: OpStage,
		Agent:     protocol.AgentInfo{ID: "agent-1"},
	})
	second, _ := manager.Propose(Proposal{
		Operation: OpPush,
		Agent:     protocol.AgentInfo{ID: "agent-2"},
	})
	got := manager.ListPending("default")
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("unexpected pending proposals: %#v", got)
	}
}
