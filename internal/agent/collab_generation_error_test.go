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

type collabErrorRecordStub struct {
	recorded int
}

func (c *collabErrorRecordStub) IsParticipant(string, string) bool          { return true }
func (c *collabErrorRecordStub) IsAgentTurn(string, string) bool            { return true }
func (c *collabErrorRecordStub) IsActive(string) bool                       { return true }
func (c *collabErrorRecordStub) GetCurrentTurnAgent(string) (string, error) { return "sa-1", nil }
func (c *collabErrorRecordStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (c *collabErrorRecordStub) GetCollaboration(string, string) CollaborationInfo {
	return CollaborationInfo{Phase: "planning"}
}
func (c *collabErrorRecordStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (c *collabErrorRecordStub) RecordMessage(string, *protocol.Message) error {
	c.recorded++
	return nil
}
func (c *collabErrorRecordStub) AnalyzeConsensus(string, *protocol.Message) string { return "" }
func (c *collabErrorRecordStub) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (c *collabErrorRecordStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }

func TestSendCollabVisibleGenerationError_PromptsNextTurn(t *testing.T) {
	hub := &collabErrorCaptureHub{}
	collab := &collabErrorRecordStub{}
	a := &Agent{
		Info:   protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend},
		Hub:    hub,
		Collab: collab,
	}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-abc",
		protocol.AgentInfo{Name: "System", Type: protocol.AgentTypeGeneral},
		"handoff",
	)
	msg.SetCollaborationID("collab-uuid-1")
	msg.SetCollaborationPhase("planning")

	a.sendCollabVisibleGenerationError(msg, "timed out", "timeout", true)

	if collab.recorded != 1 {
		t.Fatalf("expected RecordMessage once, got %d", collab.recorded)
	}
	handoffs := 0
	for _, m := range hub.sent {
		if m.Type == protocol.MessageTypeCollabDiscussion && strings.Contains(m.Content, "Collaboration turn handoff") {
			handoffs++
		}
	}
	if handoffs != 1 {
		t.Fatalf("expected one turn handoff after generation error, got %d", handoffs)
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
