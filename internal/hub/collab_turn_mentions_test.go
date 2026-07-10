package hub

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestResolveCollabParticipantLiveID_MapsStaleToLive(t *testing.T) {
	h := newTestHub(t)
	liveID := "live-sa-id"
	_ = h.RegisterAgent(&protocol.AgentInfo{
		ID:     liveID,
		Name:   "SoftwareArchitect",
		Type:   protocol.AgentTypeArchitecture,
		Status: "active",
	})

	c := &collaboration.Collaboration{
		Agents: []collaboration.CollaborationAgent{{
			AgentID:   "stale-sa-id",
			AgentName: "SoftwareArchitect",
			AgentType: protocol.AgentTypeArchitecture,
		}},
	}
	if got := h.resolveCollabParticipantLiveID(c, "stale-sa-id"); got != liveID {
		t.Fatalf("resolveCollabParticipantLiveID = %q, want %q", got, liveID)
	}
}

func TestNormalizeCollabTurnHandoffMentions_ResolvesStaleParticipantID(t *testing.T) {
	h := newTestHub(t)
	liveID := "live-sa-id"
	_ = h.RegisterAgent(&protocol.AgentInfo{
		ID:     liveID,
		Name:   "SoftwareArchitect",
		Type:   protocol.AgentTypeArchitecture,
		Status: "active",
	})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"turn-mentions",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@SoftwareArchitect -- You're up first for: design tool\n\nPropose a minimal task list.",
	)
	msg.Mentions = []string{"stale-sa-id"}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true

	h.normalizeCollabTurnHandoffMentions(msg)
	if !msg.IsMentioned(liveID) {
		t.Fatalf("expected live agent ID %q in mentions, got %v", liveID, msg.Mentions)
	}
	if msg.IsMentioned("stale-sa-id") {
		t.Fatalf("stale participant ID must not remain in mentions: %v", msg.Mentions)
	}
}
