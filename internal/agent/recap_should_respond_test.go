package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestShouldRespond_CollabRecapAssignee(t *testing.T) {
	const agentID = "rust-expert-id"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeRust, "RustExpert", []string{"rust"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabRecapStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabRecap,
		"collab-ch",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@RustExpert please recap for the user.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["recap_assignee"] = agentID
	msg.Metadata["recap_kind"] = "pre_approval"

	if !ag.shouldRespond(msg) {
		t.Fatal("expected recap assignee to respond to collaboration_recap")
	}
}

type collabRecapStub struct{ agentID string }

func (s collabRecapStub) IsParticipant(_collabID, agentID string) bool { return agentID == s.agentID }
func (collabRecapStub) IsAgentTurn(_collabID, _agentID string) bool    { return false }
func (collabRecapStub) IsActive(_collabID string) bool                 { return true }
func (collabRecapStub) GetCurrentTurnAgent(string) (string, error)     { return "", nil }
func (collabRecapStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{ID: "550e8400-e29b-41d4-a716-446655440000", Phase: "reviewing"}
}
func (collabRecapStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabRecapStub) RecordMessage(string, *protocol.Message) error { return nil }
func (collabRecapStub) AnalyzeConsensus(string, *protocol.Message) string { return "" }
func (collabRecapStub) AgentOutOfTurnMentionAllowed(string) bool       { return false }
