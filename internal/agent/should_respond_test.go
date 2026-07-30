package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// minimal stubs avoid importing hub (hub imports agent → cycle).

type shouldRespondTestHub struct {
	hubArenaNoop
	dmChannel string
}

func (shouldRespondTestHub) SendMessage(*protocol.Message) error       { return nil }
func (shouldRespondTestHub) BroadcastDirect(string, *protocol.Message) {}
func (shouldRespondTestHub) Subscribe(string) (chan *protocol.Message, error) {
	ch := make(chan *protocol.Message, 1)
	return ch, nil
}
func (shouldRespondTestHub) GetMessages(string, int) ([]*protocol.Message, error)  { return nil, nil }
func (shouldRespondTestHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) { return nil, nil }
func (shouldRespondTestHub) GetThreadParentAuthor(string) string                   { return "" }
func (shouldRespondTestHub) GetCommandHandler() CommandHandlerInterface            { return nil }
func (shouldRespondTestHub) ImageGenerationEnabled() bool                          { return false }
func (shouldRespondTestHub) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}
func (shouldRespondTestHub) MusicGenerationEnabled() bool { return false }
func (shouldRespondTestHub) GenerateAndPostMusic(context.Context, string, protocol.AgentInfo, MusicGenerateRequest) error {
	return nil
}
func (shouldRespondTestHub) ExtractAndPostMusicStems(context.Context, string, protocol.AgentInfo, MusicExtractRequest) error {
	return nil
}
func (shouldRespondTestHub) AskUserQuestion(string, string, string, string, []string) (string, error) {
	return "", nil
}
func (shouldRespondTestHub) GetAgentChannels(string) []string { return nil }
func (h shouldRespondTestHub) GetChannelType(channel string) protocol.ChannelType {
	if channel == h.dmChannel {
		return protocol.ChannelTypeDM
	}
	return protocol.ChannelTypePublic
}
func (shouldRespondTestHub) GetChannelSessionSummary(string) string { return "" }
func (shouldRespondTestHub) GetThreadMessages(string, int) ([]*protocol.Message, error) {
	return nil, nil
}
func (shouldRespondTestHub) IsChannelHeld(string) bool { return false }
func (shouldRespondTestHub) RequestToolApproval(string, string, string, string, map[string]interface{}) (bool, error) {
	return true, nil
}

type shouldRespondTestCollab struct{}

func (shouldRespondTestCollab) IsParticipant(string, string) bool          { return false }
func (shouldRespondTestCollab) IsAgentTurn(string, string) bool            { return false }
func (shouldRespondTestCollab) IsActive(string) bool                       { return false }
func (shouldRespondTestCollab) GetCurrentTurnAgent(string) (string, error) { return "", nil }
func (shouldRespondTestCollab) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (shouldRespondTestCollab) GetCollaboration(string, string) CollaborationInfo {
	return CollaborationInfo{}
}
func (shouldRespondTestCollab) GetCollaborationWorkingDirectory(string) string     { return "" }
func (shouldRespondTestCollab) RecordMessage(string, *protocol.Message) error      { return nil }
func (shouldRespondTestCollab) AnalyzeConsensus(string, *protocol.Message) string  { return "" }
func (shouldRespondTestCollab) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (shouldRespondTestCollab) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (shouldRespondTestCollab) ParticipantTurnCount(string, string) int            { return 0 }

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

type collabSystemTurnStub struct {
	agentID  string
	inactive bool
}

func (s collabSystemTurnStub) IsParticipant(_collabID, agentID string) bool {
	return agentID == s.agentID
}
func (collabSystemTurnStub) IsAgentTurn(_collabID, _agentID string) bool { return true }
func (s collabSystemTurnStub) IsActive(_collabID string) bool            { return !s.inactive }
func (collabSystemTurnStub) GetCurrentTurnAgent(string) (string, error)  { return "", nil }
func (collabSystemTurnStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabSystemTurnStub) GetCollaboration(string, string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabSystemTurnStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabSystemTurnStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (collabSystemTurnStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (collabSystemTurnStub) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (collabSystemTurnStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (collabSystemTurnStub) ParticipantTurnCount(string, string) int            { return 0 }

func TestShouldRespond_CollabInternalHandoffWakesMentionedAgent(t *testing.T) {
	const agentID = "gemini-cli-id"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeCLI, "Gemini", []string{"code"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabSystemTurnStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.Mentions = []string{agentID}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true

	if !ag.shouldRespond(msg) {
		t.Fatal("expected mentioned agent to respond to collaboration turn handoff (collab_internal_event)")
	}
}

func TestShouldRespond_CollabInternalHandoffIgnoresCancelledCollab(t *testing.T) {
	const agentID = "backend-id"
	ag := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"code"}, ai.NewMockProvider(), shouldRespondTestHub{})
	ag.Info.ID = agentID
	ag.SetCollabClient(collabSystemTurnStub{agentID: agentID, inactive: true})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue the plan discussion.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.Mentions = []string{agentID}
	msg.Metadata["collab_internal_event"] = true

	if ag.shouldRespond(msg) {
		t.Fatal("cancelled collaboration must ignore queued internal handoff")
	}
}

func TestShouldRespond_CollabInternalHandoffIgnoresNonMentionedOnTurn(t *testing.T) {
	const mentionedID = "assistant-id"
	const otherID = "backend-id"

	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()

	mentioned := NewAgent(protocol.AgentTypeAssistant, "Assistant", []string{}, mockAI, hubStub)
	mentioned.Info.ID = mentionedID
	mentioned.SetCollabClient(collabSystemTurnStub{agentID: mentionedID})

	other := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{}, mockAI, hubStub)
	other.Info.ID = otherID
	other.SetCollabClient(collabSystemTurnStub{agentID: otherID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.Mentions = []string{mentionedID}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true

	if !mentioned.shouldRespond(msg) {
		t.Fatal("expected mentioned agent to respond to collaboration turn handoff")
	}
	if other.shouldRespond(msg) {
		t.Fatal("non-mentioned agent must not respond to collaboration turn handoff even when IsAgentTurn is true")
	}
}

func TestShouldRespond_CollabSeedBannerIgnored(t *testing.T) {
	const agentID = "gemini-cli-id"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeCLI, "Gemini", []string{"code"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabSystemTurnStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"🤝 **Collaboration Started** (ID: `abcd1234`)",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true

	if ag.shouldRespond(msg) {
		t.Fatal("expected agent to ignore collaboration seed banner (collab_internal_event)")
	}
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

func (s collabTaskAssigneeStub) IsParticipant(_collabID, agentID string) bool {
	return agentID == s.agentID
}
func (collabTaskAssigneeStub) IsAgentTurn(_collabID, _agentID string) bool { return false }
func (collabTaskAssigneeStub) IsActive(_collabID string) bool              { return true }
func (collabTaskAssigneeStub) GetCurrentTurnAgent(string) (string, error)  { return "", nil }
func (collabTaskAssigneeStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (s collabTaskAssigneeStub) GetCollaboration(_collabID, agentID string) CollaborationInfo {
	if agentID == s.agentID {
		return CollaborationInfo{Phase: "executing"}
	}
	return CollaborationInfo{}
}
func (collabTaskAssigneeStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabTaskAssigneeStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (collabTaskAssigneeStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (collabTaskAssigneeStub) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (collabTaskAssigneeStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (collabTaskAssigneeStub) ParticipantTurnCount(string, string) int            { return 0 }

type collabMultiActiveStub struct {
	agentID string
}

func (s collabMultiActiveStub) IsParticipant(collabID, agentID string) bool {
	return agentID == s.agentID && (collabID == "exec-collab-id" || collabID == "plan-collab-id")
}
func (collabMultiActiveStub) IsAgentTurn(string, string) bool            { return true }
func (collabMultiActiveStub) IsActive(string) bool                       { return true }
func (collabMultiActiveStub) GetCurrentTurnAgent(string) (string, error) { return "", nil }
func (collabMultiActiveStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{ID: "exec-collab-id", Phase: "executing"}
}
func (s collabMultiActiveStub) GetCollaboration(collabID, agentID string) CollaborationInfo {
	if agentID != s.agentID {
		return CollaborationInfo{}
	}
	switch collabID {
	case "plan-collab-id":
		return CollaborationInfo{ID: collabID, Phase: "planning"}
	case "exec-collab-id":
		return CollaborationInfo{ID: collabID, Phase: "executing"}
	default:
		return CollaborationInfo{}
	}
}
func (collabMultiActiveStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabMultiActiveStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (collabMultiActiveStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (collabMultiActiveStub) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (collabMultiActiveStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (collabMultiActiveStub) ParticipantTurnCount(string, string) int            { return 0 }

type collabPlanningTurnStub struct{}

func (collabPlanningTurnStub) IsParticipant(string, string) bool { return true }
func (collabPlanningTurnStub) IsAgentTurn(string, string) bool   { return false }
func (collabPlanningTurnStub) IsActive(string) bool              { return true }
func (collabPlanningTurnStub) GetCurrentTurnAgent(string) (string, error) {
	return "gemini-id", nil
}
func (collabPlanningTurnStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabPlanningTurnStub) GetCollaboration(string, string) CollaborationInfo {
	return CollaborationInfo{Phase: "planning"}
}
func (collabPlanningTurnStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabPlanningTurnStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (collabPlanningTurnStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (collabPlanningTurnStub) AgentOutOfTurnMentionAllowed(string) bool           { return true }
func (collabPlanningTurnStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (collabPlanningTurnStub) ParticipantTurnCount(string, string) int            { return 0 }

func TestShouldRespond_PlanningCollabIgnoresOtherExecutingCollab(t *testing.T) {
	const agentID = "agent-multi"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeAssistant, "Assistant", []string{}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabMultiActiveStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-plan",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"@Assistant -- You're up first.",
	)
	msg.Metadata = map[string]interface{}{
		"collaboration_id": "plan-collab-id",
	}
	msg.Mentions = []string{agentID}

	if !ag.shouldRespond(msg) {
		t.Fatal("expected planning handoff to be answered even when agent is also in an executing collab")
	}
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

type collabExhaustedMentionStub struct{ agentID string }

func (s collabExhaustedMentionStub) IsParticipant(_collabID, agentID string) bool {
	return agentID == s.agentID
}
func (collabExhaustedMentionStub) IsAgentTurn(string, string) bool { return false }
func (collabExhaustedMentionStub) IsActive(string) bool            { return true }
func (collabExhaustedMentionStub) GetCurrentTurnAgent(string) (string, error) {
	return "", nil
}
func (collabExhaustedMentionStub) GetCollaborationForAgent(string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabExhaustedMentionStub) GetCollaboration(string, string) CollaborationInfo {
	return CollaborationInfo{}
}
func (collabExhaustedMentionStub) GetCollaborationWorkingDirectory(string) string { return "" }
func (collabExhaustedMentionStub) RecordMessage(string, *protocol.Message) error  { return nil }
func (collabExhaustedMentionStub) AnalyzeConsensus(string, *protocol.Message) string {
	return ""
}
func (collabExhaustedMentionStub) AgentOutOfTurnMentionAllowed(string) bool           { return false }
func (collabExhaustedMentionStub) PlanningSpeakerCooldownBlocked(string, string) bool { return false }
func (collabExhaustedMentionStub) ParticipantTurnCount(string, string) int            { return 0 }

func TestShouldRespond_CollaborationMentionIgnoredWhenDiscussionExhausted(t *testing.T) {
	const agentID = "agent-xyz"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeBackend, "BackendExpert", []string{"api"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabExhaustedMentionStub{agentID: agentID})

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"collab-test",
		protocol.AgentInfo{ID: "human-user", Name: "alice", Type: "human"},
		"@BackendExpert please say more about the plan",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.Mention(agentID)

	if ag.shouldRespond(msg) {
		t.Fatal("expected no agent reply when discussion is exhausted and only @mention would apply")
	}
}

func TestShouldRespond_ExplicitMentionOverridesIdeRoute(t *testing.T) {
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()

	backend := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"go"}, mockAI, hubStub)
	backend.SetCollabClient(shouldRespondTestCollab{})

	assistant := NewAgent(protocol.AgentTypeAssistant, "Assistant", []string{"help"}, mockAI, hubStub)
	assistant.SetCollabClient(shouldRespondTestCollab{})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human-user", Name: "camron", Type: "human"},
		"@Assistant hello",
	)
	msg.Metadata = map[string]interface{}{
		protocol.IdeMetaRouteAgentType: "backend",
	}
	msg.Mention(assistant.Info.ID)

	if backend.shouldRespond(msg) {
		t.Fatal("BackendEngineer must not respond when only Assistant is @mentioned")
	}
	if !assistant.shouldRespond(msg) {
		t.Fatal("Assistant should respond when @mentioned even with IDE backend route metadata")
	}
}

func TestShouldRespond_SlashCommandIgnoresIdeRoute(t *testing.T) {
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()

	frontend := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", []string{"ui"}, mockAI, hubStub)
	frontend.SetCollabClient(shouldRespondTestCollab{})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human-user", Name: "camron", Type: "human"},
		"/create-expert ios SwiftExpert ollama gemma3:12b",
	)
	msg.Metadata = map[string]interface{}{
		protocol.IdeMetaRouteAgentType: "frontend",
		protocol.MetadataSlashCommand:  true,
		"implementation_session":       true,
		protocol.IdeMetaEditorMode:     "agent",
	}

	if frontend.shouldRespond(msg) {
		t.Fatal("FrontendEngineer must not respond to slash commands even with IDE route metadata")
	}
}

func TestShouldRespond_PlanningHandoffSkippedWhenGenerationInFlight(t *testing.T) {
	const collabID = "550e8400-e29b-41d4-a716-446655440000"
	const agentID = "claude-id"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()
	ag := NewAgent(protocol.AgentTypeCLI, "Claude", []string{"code"}, mockAI, hubStub)
	ag.Info.ID = agentID
	ag.SetCollabClient(collabSystemTurnStub{agentID: agentID})

	RegisterGenCancelForTest(ag, "collab-test", func() {})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue planning.",
	)
	msg.SetCollaborationID(collabID)
	msg.Mention(agentID)
	msg.Metadata = map[string]interface{}{
		"collab_internal_event": true,
		"collab_turn_handoff":   true,
	}

	if ag.shouldRespond(msg) {
		t.Fatal("planning handoff must not start a second concurrent generation")
	}
}

func TestShouldRespond_CollaborationAgentMentionInPlanDoesNotStealTurn(t *testing.T) {
	const collabID = "550e8400-e29b-41d4-a716-446655440000"
	hubStub := shouldRespondTestHub{}
	mockAI := ai.NewMockProvider()

	platform := NewAgent(protocol.AgentTypeDevOps, "PlatformEngineer", []string{"infra"}, mockAI, hubStub)
	platform.Info.ID = "platform-id"
	platform.SetCollabClient(collabPlanningTurnStub{})

	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "assistant-id", Name: "Assistant", Type: protocol.AgentTypeAssistant},
		"- Task 1: @PlatformEngineer - Review CI pipeline",
	)
	msg.SetCollaborationID(collabID)
	msg.Mention(platform.Info.ID)

	if platform.shouldRespond(msg) {
		t.Fatal("PlatformEngineer must not respond to @mention in another agent's planning prose when it is not their turn")
	}
}

func TestShouldRespond_implementationStatusCheckPublicChannel(t *testing.T) {
	t.Parallel()
	ch := "implement-scenarios"
	hubStub := shouldRespondTestHub{}
	sa := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", []string{"design"}, ai.NewMockProvider(), hubStub)
	sa.Info.ID = "sa-1"
	be := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"api"}, ai.NewMockProvider(), hubStub)
	be.Info.ID = "be-1"
	sa.replaceChannelHistory(ch, []*protocol.Message{
		{
			ID:      "u1",
			Channel: ch,
			Type:    protocol.MessageTypeQuestion,
			From:    protocol.AgentInfo{ID: "u0", Name: "User"},
			Content: "the app is not booting can you fix it?",
		},
		{
			ID:      "a1",
			Channel: ch,
			Type:    protocol.MessageTypeChat,
			From:    sa.Info,
			Content: "Implementation session complete — proposals submitted for approval (changes to: src/App.js).",
		},
	})
	be.replaceChannelHistory(ch, sa.channelHistory(ch))

	msg := protocol.NewMessage(protocol.MessageTypeQuestion, ch, protocol.AgentInfo{ID: "u2", Name: "User"}, "is it fixed?")
	msg.Metadata = map[string]interface{}{
		MetadataConversationMode: ConversationModeChat,
		"editor_mode":            "agent",
	}

	if !sa.shouldRespond(msg) {
		t.Fatal("SoftwareArchitect should respond to status check after its implementation session")
	}
	if be.shouldRespond(msg) {
		t.Fatal("BackendEngineer should not respond when it did not run the implementation session")
	}
}

type customChannelHub struct {
	shouldRespondTestHub
	agents      []protocol.AgentInfo
	channelType protocol.ChannelType
}

func (h customChannelHub) GetChannelType(string) protocol.ChannelType {
	if h.channelType != "" {
		return h.channelType
	}
	return protocol.ChannelTypeCustom
}

func (h customChannelHub) GetChannelAgents(string) ([]protocol.AgentInfo, error) {
	return h.agents, nil
}

func TestShouldRespond_CustomChannelAlwaysHasFallbackResponder(t *testing.T) {
	t.Parallel()
	mockAI := ai.NewMockProvider()

	rust := NewAgent(protocol.AgentTypeRust, "RustExpert", []string{"Rust", "Cargo"}, mockAI, nil)
	rust.Info.ID = "rust-1"
	backend := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"APIs", "Services"}, mockAI, nil)
	backend.Info.ID = "be-1"
	arch := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", []string{"System Design"}, mockAI, nil)
	arch.Info.ID = "arch-1"

	hub := customChannelHub{
		agents: []protocol.AgentInfo{rust.Info, backend.Info, arch.Info},
	}
	rust.Hub = hub
	backend.Hub = hub
	arch.Hub = hub
	rust.SetCollabClient(shouldRespondTestCollab{})
	backend.SetCollabClient(shouldRespondTestCollab{})
	arch.SetCollabClient(shouldRespondTestCollab{})

	// No expertise keywords ("rust"/etc.) and no @mentions — previously silent.
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"rustgame",
		protocol.AgentInfo{ID: "human-user", Name: "Camron", Type: "human"},
		"lets build an RTS game, with 2 groups that face each other in battel",
	)

	responders := 0
	for _, ag := range []*Agent{rust, backend, arch} {
		if ag.shouldRespond(msg) {
			responders++
		}
	}
	if responders == 0 {
		t.Fatal("expected at least one custom-channel agent to reply without @mention")
	}
	if responders > customChannelBroadPromptResponderCap {
		t.Fatalf("expected at most %d fallback responders, got %d", customChannelBroadPromptResponderCap, responders)
	}
}

func TestShouldRespond_CustomChannelRelevanceStillWins(t *testing.T) {
	t.Parallel()
	mockAI := ai.NewMockProvider()
	rust := NewAgent(protocol.AgentTypeRust, "RustExpert", []string{"Rust", "Cargo"}, mockAI, nil)
	rust.Info.ID = "rust-1"
	backend := NewAgent(protocol.AgentTypeBackend, "BackendEngineer", []string{"APIs", "Services"}, mockAI, nil)
	backend.Info.ID = "be-1"
	hub := customChannelHub{agents: []protocol.AgentInfo{rust.Info, backend.Info}}
	rust.Hub = hub
	backend.Hub = hub
	rust.SetCollabClient(shouldRespondTestCollab{})
	backend.SetCollabClient(shouldRespondTestCollab{})

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"rustgame",
		protocol.AgentInfo{ID: "human-user", Name: "Camron", Type: "human"},
		"can you help with this rust ownership error?",
	)
	if !rust.shouldRespond(msg) {
		t.Fatal("RustExpert should respond via expertise match")
	}
}

func TestShouldRespond_CollaborationChannelHasFallbackResponder(t *testing.T) {
	t.Parallel()
	ag := NewAgent(protocol.AgentTypeArchitecture, "SoftwareArchitect", []string{"System Design"}, ai.NewMockProvider(), nil)
	ag.Info.ID = "arch-1"
	hub := customChannelHub{
		agents:      []protocol.AgentInfo{ag.Info},
		channelType: protocol.ChannelTypeCollaboration,
	}
	ag.Hub = hub
	ag.SetCollabClient(shouldRespondTestCollab{})
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"collab-rustgame",
		protocol.AgentInfo{ID: "human-user", Name: "Camron", Type: "human"},
		"keep going with the RTS",
	)
	if !ag.shouldRespond(msg) {
		t.Fatal("expected a collaboration-channel fallback response without @mention")
	}
}
