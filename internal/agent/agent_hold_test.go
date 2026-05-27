package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type holdTestHub struct {
	shouldRespondTestHub
	held    bool
	sent    []*protocol.Message
	heldCh  string
}

func (h *holdTestHub) IsChannelHeld(ch string) bool {
	if h.heldCh != "" && h.heldCh != ch {
		return false
	}
	return h.held
}

func (h *holdTestHub) SendMessage(msg *protocol.Message) error {
	h.sent = append(h.sent, msg)
	return nil
}

func TestShouldRespondFalseWhenChannelHeld(t *testing.T) {
	hubStub := &holdTestHub{held: true}
	a := NewAgent(protocol.AgentTypeBackend, "HoldTest", nil, ai.NewMockProvider(), hubStub)
	msg := protocol.NewMessage(protocol.MessageTypeChat, "general", protocol.AgentInfo{
		ID: "other", Name: "Other", Type: protocol.AgentTypeBackend,
	}, "hello")
	if a.shouldRespond(msg) {
		t.Fatal("should not respond when channel is held")
	}
}

func TestPromptNextCollaborationTurnSkippedWhenHeld(t *testing.T) {
	hubStub := &holdTestHub{held: true, heldCh: "collab-ch"}
	a := NewAgent(protocol.AgentTypeBackend, "HoldTest2", nil, ai.NewMockProvider(), hubStub)
	a.Info.ID = "agent-a"
	a.SetCollabClient(collabHandoffStub{nextAgentID: "agent-b"})

	src := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{
		ID: "agent-a", Name: "A1", Type: protocol.AgentTypeBackend,
	}, "plan")
	src.SetCollaborationID("collab-1")
	a.promptNextCollaborationTurn(src, "collab-1")

	if len(hubStub.sent) != 0 {
		t.Fatalf("expected no handoff messages when channel held, got %d", len(hubStub.sent))
	}
}

func TestPromptNextCollaborationTurnSendsHandoffWhenNotHeld(t *testing.T) {
	hubStub := &holdTestHub{held: false, heldCh: "collab-ch"}
	a := NewAgent(protocol.AgentTypeBackend, "HoldTest3", nil, ai.NewMockProvider(), hubStub)
	a.Info.ID = "agent-a"
	a.SetCollabClient(collabHandoffStub{nextAgentID: "agent-b"})

	src := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{
		ID: "agent-a", Name: "A1", Type: protocol.AgentTypeBackend,
	}, "plan")
	src.SetCollaborationID("collab-1")
	a.promptNextCollaborationTurn(src, "collab-1")

	if len(hubStub.sent) != 1 {
		t.Fatalf("expected one handoff message, got %d", len(hubStub.sent))
	}
	if hubStub.sent[0].Mentions == nil || len(hubStub.sent[0].Mentions) != 1 || hubStub.sent[0].Mentions[0] != "agent-b" {
		t.Fatalf("expected mention of next agent, got %#v", hubStub.sent[0].Mentions)
	}
}

type collabHandoffStub struct {
	nextAgentID string
}

func (c collabHandoffStub) IsParticipant(string, string) bool          { return true }
func (c collabHandoffStub) IsAgentTurn(string, string) bool            { return true }
func (c collabHandoffStub) IsActive(string) bool                       { return true }
func (c collabHandoffStub) GetCurrentTurnAgent(string) (string, error) { return c.nextAgentID, nil }
func (c collabHandoffStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{ID: "collab-1", Channel: "collab-ch", Phase: "planning"}
}
func (c collabHandoffStub) GetCollaboration(collabID, _ string) CollaborationInfo {
	if collabID == "collab-1" {
		return CollaborationInfo{ID: "collab-1", Channel: "collab-ch", Phase: "planning"}
	}
	return CollaborationInfo{}
}
func (c collabHandoffStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (c collabHandoffStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (c collabHandoffStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (c collabHandoffStub) AgentOutOfTurnMentionAllowed(string) bool { return true }
