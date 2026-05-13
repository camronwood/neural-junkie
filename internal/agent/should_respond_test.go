package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// minimal stubs avoid importing hub (hub imports agent → cycle).

type shouldRespondTestHub struct{ dmChannel string }

func (shouldRespondTestHub) SendMessage(*protocol.Message) error { return nil }
func (shouldRespondTestHub) BroadcastDirect(string, *protocol.Message) {}
func (shouldRespondTestHub) Subscribe(string) (chan *protocol.Message, error) {
	ch := make(chan *protocol.Message, 1)
	return ch, nil
}
func (shouldRespondTestHub) GetMessages(string, int) ([]*protocol.Message, error) { return nil, nil }
func (shouldRespondTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (shouldRespondTestHub) GetThreadParentAuthor(string) string { return "" }
func (shouldRespondTestHub) GetCommandHandler() CommandHandlerInterface { return nil }
func (shouldRespondTestHub) GetAgentChannels(string) []string { return nil }
func (h shouldRespondTestHub) GetChannelType(channel string) protocol.ChannelType {
	if channel == h.dmChannel {
		return protocol.ChannelTypeDM
	}
	return protocol.ChannelTypePublic
}

type shouldRespondTestCollab struct{}

func (shouldRespondTestCollab) IsParticipant(string, string) bool { return false }
func (shouldRespondTestCollab) IsAgentTurn(string, string) bool { return false }
func (shouldRespondTestCollab) IsActive(string) bool { return false }
func (shouldRespondTestCollab) GetCurrentTurnAgent(string) (string, error) { return "", nil }
func (shouldRespondTestCollab) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (shouldRespondTestCollab) RecordMessage(string, *protocol.Message) error { return nil }
func (shouldRespondTestCollab) AnalyzeConsensus(string, *protocol.Message) string { return "" }

type dmSlugHubStub struct{ shouldRespondTestHub }

func (dmSlugHubStub) GetChannelType(string) protocol.ChannelType {
	return protocol.ChannelTypePublic
}

func TestShouldRespond_DMBySlugWhenHubReturnsPublic(t *testing.T) {
	const dm = "dm-alice-test-bot"
	hubStub := dmSlugHubStub{shouldRespondTestHub: shouldRespondTestHub{dmChannel: dm}}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeRust, "test-dm-bot", []string{"rust"}, mockAI, hubStub)
	ag.SetCollabClient(shouldRespondTestCollab{})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		dm,
		protocol.AgentInfo{ID: "human-user", Name: "alice", Type: "human"},
		"Hello",
	)
	msg.SetCollaborationID("orphan-collab-id")

	if !ag.shouldRespond(msg) {
		t.Fatal("expected dm- slug to classify as DM even when hub GetChannelType is wrong")
	}
}

type collabSystemTurnStub struct{ agentID string }

func (s collabSystemTurnStub) IsParticipant(_collabID, agentID string) bool { return agentID == s.agentID }
func (collabSystemTurnStub) IsAgentTurn(_collabID, _agentID string) bool    { return true }
func (collabSystemTurnStub) IsActive(_collabID string) bool                 { return true }
func (collabSystemTurnStub) GetCurrentTurnAgent(string) (string, error)     { return "", nil }
func (collabSystemTurnStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabSystemTurnStub) RecordMessage(string, *protocol.Message) error { return nil }
func (collabSystemTurnStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}

func TestShouldRespond_SystemCollabTurnPrompt(t *testing.T) {
	const agentID = "cursor-cli-id"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeCLI, "Cursor", []string{"code"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabSystemTurnStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"general",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@Cursor -- You're up first.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")

	if !ag.shouldRespond(msg) {
		t.Fatal("expected agent to respond to System-authored collaboration turn prompt")
	}
}

func TestShouldRespond_PlainSystemChatStillIgnored(t *testing.T) {
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeCLI, "Cursor", []string{"code"}, mockAI, hubStub)
	ag.Info.ID = "cursor-cli-id"
	ag.SetCollabClient(collabSystemTurnStub{agentID: "cursor-cli-id"})

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"general",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Server restarted",
	)

	if ag.shouldRespond(msg) {
		t.Fatal("expected plain System chat to be ignored")
	}
}

func TestShouldRespond_DMWithUnknownCollaborationID(t *testing.T) {
	const dm = "dm-alice-test-bot"
	hubStub := shouldRespondTestHub{dmChannel: dm}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeRust, "test-dm-bot", []string{"rust"}, mockAI, hubStub)
	ag.SetCollabClient(shouldRespondTestCollab{})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		dm,
		protocol.AgentInfo{ID: "human-user", Name: "alice", Type: "human"},
		"Hello — can you hear me?",
	)
	msg.SetCollaborationID("definitely-not-a-registered-collaboration-id")

	if !ag.shouldRespond(msg) {
		t.Fatal("expected DM to respond to human despite unknown collaboration_id in metadata")
	}
}

type collabTaskAssigneeStub struct{ agentID string }

func (s collabTaskAssigneeStub) IsParticipant(_collabID, agentID string) bool { return agentID == s.agentID }
func (collabTaskAssigneeStub) IsAgentTurn(_collabID, _agentID string) bool    { return false }
func (collabTaskAssigneeStub) IsActive(_collabID string) bool                 { return true }
func (collabTaskAssigneeStub) GetCurrentTurnAgent(string) (string, error)     { return "", nil }
func (collabTaskAssigneeStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabTaskAssigneeStub) RecordMessage(string, *protocol.Message) error { return nil }
func (collabTaskAssigneeStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}

func TestShouldRespond_CollabTaskViaAssigneeMetadata(t *testing.T) {
	const agentID = "agent-xyz"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendExpert", []string{"api"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabTaskAssigneeStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabTask,
		"general",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@BackendExpert -- Your assigned task:\n\ndo thing",
	)
	msg.Metadata = map[string]interface{}{
		"collaboration_id": "550e8400-e29b-41d4-a716-446655440000",
		"task_id":          "task-1",
		"task_status":      "pending",
		"task_assigned_to": agentID,
	}

	if !ag.shouldRespond(msg) {
		t.Fatal("expected assignee to respond to collaboration_task via task_assigned_to metadata")
	}
}
