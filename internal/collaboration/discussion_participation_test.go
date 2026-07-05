package collaboration

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestParticipationQuorumMet(t *testing.T) {
	t.Parallel()
	arch := "arch-id"
	be := "be-id"

	d := &DiscussionSession{
		Participants:      []string{arch, be},
		TotalMessageCount: 2,
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: arch, Name: "SoftwareArchitect"}, "plan"),
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: be, Name: "BackendEngineer"}, "agree"),
		},
	}
	if !participationQuorumMet(d) {
		t.Fatal("expected quorum when both participants spoke")
	}

	d.Messages = d.Messages[:1]
	d.TotalMessageCount = 1
	if participationQuorumMet(d) {
		t.Fatal("expected no quorum when one participant is silent")
	}
}

func TestSilentPlanningParticipantIDs(t *testing.T) {
	t.Parallel()
	h := newRunbookMockHub()
	h.addAgent("arch-id", "SoftwareArchitect", protocol.AgentTypeArchitecture, nil)
	h.addAgent("be-id", "BackendEngineer", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	collab, err := cm.CreateCollaboration(
		"plan",
		[]string{"arch-id", "be-id"},
		"collab-plan",
		"tester",
		DiscussionConfig{MaxRounds: 1, MaxTotalMessages: 4},
		CreateOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	silent := cm.SilentPlanningParticipantIDs(collab.ID)
	if len(silent) != 2 {
		t.Fatalf("expected both participants silent, got %v", silent)
	}

	discMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		collab.Channel,
		protocol.AgentInfo{ID: "arch-id", Name: "SoftwareArchitect"},
		"plan",
	)
	if err := cm.RecordMessage(collab.ID, discMsg); err != nil {
		t.Fatal(err)
	}
	silent = cm.SilentPlanningParticipantIDs(collab.ID)
	if len(silent) != 1 || silent[0] != "be-id" {
		t.Fatalf("expected only BackendEngineer silent, got %v", silent)
	}
	if got := cm.ParticipantAgentName(collab.ID, "be-id"); got != "BackendEngineer" {
		t.Fatalf("ParticipantAgentName = %q", got)
	}
}

func TestDiscussionTurnBudget_untilQuorum(t *testing.T) {
	t.Parallel()
	arch := "arch-id"
	be := "be-id"
	c := &Collaboration{Phase: PhasePlanning}
	d := &DiscussionSession{
		Participants: []string{arch, be},
		TurnBudget:   2,
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: arch, Name: "SoftwareArchitect"}, "plan"),
		},
		TotalMessageCount: 1,
	}
	if got := discussionTurnBudget(c, d); got != 1 {
		t.Fatalf("expected turn budget 1 before quorum, got %d", got)
	}
	d.Messages = append(d.Messages, protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
		protocol.AgentInfo{ID: be, Name: "BackendEngineer"}, "agree"))
	d.TotalMessageCount = 2
	if got := discussionTurnBudget(c, d); got != 2 {
		t.Fatalf("expected full turn budget after quorum, got %d", got)
	}
}

func TestPlanningDiscussionTimeoutElapsed_graceForSilent(t *testing.T) {
	t.Parallel()
	c := &Collaboration{Phase: PhasePlanning}
	d := &DiscussionSession{
		Participants:      []string{"a1", "a2", "a3"},
		Timeout:           2 * time.Minute,
		MaxTotalMessages:  6,
		TotalMessageCount: 1,
		StartedAt:         time.Now().Add(-3 * time.Minute),
		Messages: []*protocol.Message{
			protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "ch",
				protocol.AgentInfo{ID: "a1", Name: "A"}, "plan"),
		},
	}
	if planningDiscussionTimeoutElapsed(c, d) {
		t.Fatal("expected grace while two participants remain silent")
	}
	d.StartedAt = time.Now().Add(-6 * time.Minute)
	if !planningDiscussionTimeoutElapsed(c, d) {
		t.Fatal("expected timeout after grace window")
	}
}
