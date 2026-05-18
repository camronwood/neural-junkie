package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSyncCollabDiscussionIntoSnapshotChannels(t *testing.T) {
	collabID := "c1111111-1111-1111-1111-111111111111"
	chName := "collab-" + collabID
	dm := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		protocol.AgentInfo{ID: "a1", Name: "RustExpert", Type: protocol.AgentTypeRust},
		"planning reply",
	)
	dm.ID = "disc-msg-1"

	snap := &SessionSnapshot{
		Channels: map[string]*ChannelSnapshot{
			chName: {Name: chName, Messages: []*protocol.Message{}},
		},
		Collaborations: map[string]*collaboration.Collaboration{
			collabID: {
				ID:      collabID,
				Channel: chName,
				Phase:   collaboration.PhaseCancelled,
				Discussion: &collaboration.DiscussionSession{
					Messages: []*protocol.Message{dm},
				},
			},
		},
	}

	syncCollabDiscussionIntoSnapshotChannels(snap)
	msgs := snap.Channels[chName].Messages
	if len(msgs) != 1 {
		t.Fatalf("expected 1 mirrored message, got %d", len(msgs))
	}
	if msgs[0].ID != dm.ID {
		t.Fatalf("expected mirrored id %q, got %q", dm.ID, msgs[0].ID)
	}
}

func TestDedupeMessagesByID(t *testing.T) {
	from := protocol.AgentInfo{ID: "u", Name: "Camron", Type: protocol.AgentTypeGeneral}
	m := protocol.NewMessage(protocol.MessageTypeQuestion, "ch", from, "dup")
	m.ID = "same-id"
	out := dedupeMessagesByID([]*protocol.Message{m, m, m})
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
}
