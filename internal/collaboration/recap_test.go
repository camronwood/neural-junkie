package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSelectRecapFacilitator_LastSpeaker(t *testing.T) {
	c := &Collaboration{
		Agents: []CollaborationAgent{
			{AgentID: "a1", AgentName: "Alpha"},
			{AgentID: "a2", AgentName: "Beta"},
		},
		Discussion: &DiscussionSession{
			Messages: []*protocol.Message{
				protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
					protocol.AgentInfo{ID: "a1", Name: "Alpha"}, "first"),
				protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
					protocol.AgentInfo{ID: "a2", Name: "Beta"}, "last word"),
			},
		},
	}
	got := SelectRecapFacilitator(c, RecapKindPreApproval)
	if got != "a2" {
		t.Fatalf("SelectRecapFacilitator = %q, want a2", got)
	}
}

func TestSelectRecapFacilitator_FallbackFirstAgent(t *testing.T) {
	c := &Collaboration{
		Agents: []CollaborationAgent{{AgentID: "a1", AgentName: "Alpha"}},
	}
	if got := SelectRecapFacilitator(c, RecapKindPreApproval); got != "a1" {
		t.Fatalf("fallback = %q, want a1", got)
	}
}

type recapTestHub struct{}

func (recapTestHub) SendMessage(*protocol.Message) error { return nil }
func (recapTestHub) GetAgent(id string) (*protocol.AgentInfo, error) {
	return &protocol.AgentInfo{ID: id, Name: id, Type: protocol.AgentTypeRust}, nil
}
func (recapTestHub) FindLiveAgentByDisplayName(string, protocol.AgentType) *protocol.AgentInfo {
	return nil
}
func (recapTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (recapTestHub) CreateChannelWithType(string, string, string, protocol.ChannelType, string) *protocol.Channel {
	return nil
}

func TestApprovePlan_BlockedWhileRecapPending(t *testing.T) {
	cm := NewCollaborationManager(recapTestHub{})
	collab, err := cm.CreateCollaboration("goal", []string{"a1", "a2"}, "general", "user", DiscussionConfig{}, CreateOptions{})
	if err != nil {
		t.Fatalf("CreateCollaboration: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("TransitionToReviewing: %v", err)
	}
	if err := cm.MarkPlanningRecapDispatched(collab.ID, "a1"); err != nil {
		t.Fatalf("MarkPlanningRecapDispatched: %v", err)
	}

	_, err = cm.ApprovePlan(collab.ID)
	if err == nil {
		t.Fatal("expected ApprovePlan error while recap pending")
	}
}

func TestSelectRecapFacilitator_FinalUsesExecutionDiscussion(t *testing.T) {
	c := &Collaboration{
		PlanningRecapAgentID: "a1",
		Agents: []CollaborationAgent{
			{AgentID: "a1", AgentName: "Alpha"},
			{AgentID: "a2", AgentName: "Beta"},
		},
		Discussion: &DiscussionSession{
			Messages: []*protocol.Message{
				protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
					protocol.AgentInfo{ID: "a2", Name: "Beta"}, "execution wrap-up"),
			},
		},
	}
	if got := SelectRecapFacilitator(c, RecapKindFinal); got != "a2" {
		t.Fatalf("final facilitator = %q, want a2", got)
	}
}

func TestBuildRecapContext_IncludesPlanningRecapForFinal(t *testing.T) {
	c := &Collaboration{
		Description:   "goal",
		PlanningRecap: "Earlier we agreed on X.",
		Tasks:         []CollaborationTask{{Title: "T1", Status: TaskCompleted, Output: "done"}},
	}
	ctx := BuildRecapContext(c, RecapKindFinal)
	if !strings.Contains(ctx, "Earlier we agreed on X.") {
		t.Fatalf("missing planning recap in final context: %s", ctx)
	}
	if !strings.Contains(ctx, "done") {
		t.Fatal("missing task output")
	}
}

func TestCopyDiscussionSession_PreservesMessages(t *testing.T) {
	d := &DiscussionSession{
		ID: "d1",
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeChat, "ch", protocol.AgentInfo{ID: "a1", Name: "A"}, "hi"),
		},
	}
	cp := CopyDiscussionSession(d)
	if cp == nil || len(cp.Messages) != 1 || cp.ID != "d1" {
		t.Fatalf("copy messages = %v", cp)
	}
}
