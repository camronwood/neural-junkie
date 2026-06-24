package collaboration

import (
	"testing"

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
