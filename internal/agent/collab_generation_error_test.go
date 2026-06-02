package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

type collabErrorCaptureHub struct {
	shouldRespondTestHub
	sent []*protocol.Message
}

func (h *collabErrorCaptureHub) SendMessage(msg *protocol.Message) error {
	cp := *msg
	h.sent = append(h.sent, &cp)
	return nil
}

func (h *collabErrorCaptureHub) GetChannelType(channel string) protocol.ChannelType {
	if strings.HasPrefix(channel, "collab") {
		return protocol.ChannelTypeCollaboration
	}
	return protocol.ChannelTypePublic
}

func TestSendGenerationFailureMessages_CollabDiscussion(t *testing.T) {
	hub := &collabErrorCaptureHub{}
	a := &Agent{
		Info: protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		Hub:  hub,
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-abc",
		protocol.AgentInfo{Name: "User", Type: protocol.AgentTypeGeneral},
		"turn",
	)
	msg.SetCollaborationID("collab-uuid-1")
	msg.SetCollaborationPhase("planning")

	a.sendGenerationFailureMessages(msg, errors.New("connection refused"))

	if len(hub.sent) < 2 {
		t.Fatalf("expected system + collab discussion, got %d messages", len(hub.sent))
	}
	hasSystem := false
	hasCollab := false
	for _, m := range hub.sent {
		if m.Type == protocol.MessageTypeSystemInfo {
			hasSystem = true
		}
		if m.Type == protocol.MessageTypeCollabDiscussion {
			hasCollab = true
			if m.GetCollaborationID() != "collab-uuid-1" {
				t.Fatalf("collab id: got %q", m.GetCollaborationID())
			}
			if m.Metadata["generation_error"] != true {
				t.Fatalf("expected generation_error metadata")
			}
		}
	}
	if !hasSystem || !hasCollab {
		t.Fatalf("system=%v collab=%v", hasSystem, hasCollab)
	}
}

func TestSendCollabVisibleGenerationError_SkipsNonCollabChannel(t *testing.T) {
	hub := &collabErrorCaptureHub{}
	a := &Agent{
		Info: protocol.AgentInfo{ID: "a1", Name: "Assistant", Type: protocol.AgentTypeGeneral},
		Hub:  hub,
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{Name: "User", Type: protocol.AgentTypeGeneral},
		"hi",
	)
	msg.SetCollaborationID("collab-uuid-1")

	a.sendCollabVisibleGenerationError(msg, "model down", "provider_error", true)
	if len(hub.sent) != 0 {
		t.Fatalf("expected no collab message on non-collab channel, got %d", len(hub.sent))
	}
}
