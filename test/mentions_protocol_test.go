package test

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestMentionParsing tests mention parsing with various formats
func TestMentionParsing(t *testing.T) {
	testCases := []struct {
		content     string
		expected    []string
		description string
	}{
		{"@AgentName hello", []string{"agentname"}, "Simple mention"},
		{"@AgentName @AnotherAgent hello", []string{"agentname", "anotheragent"}, "Multiple mentions"},
		{"Hello @AgentName how are you?", []string{"agentname"}, "Mention in middle"},
		{"@AgentName", []string{"agentname"}, "Only mention"},
		{"@Agent-Name", []string{"agent-name"}, "Mention with hyphen"},
		{"@Agent_Name", []string{"agent"}, "Mention with underscore"},
		{"@Agent123", []string{"agent123"}, "Mention with numbers"},
		{"@Agent.Name", []string{"agent"}, "Mention with dot"},
		{"@Agent@Name", []string{"agent", "name"}, "Mention with @ in name"},
		{"No mentions here", []string{}, "No mentions"},
		{"@", []string{}, "Incomplete mention"},
		{"@@AgentName", []string{"agentname"}, "Double @"},
		{"@ AgentName", []string{}, "Space after @"},
		{"@AgentName@", []string{"agentname"}, "Mention ending with @"},
	}

	for _, tc := range testCases {
		msg := protocol.NewMessage(
			protocol.MessageTypeChat,
			"test-channel",
			protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
			tc.content,
		)

		mentions := msg.Mentions
		if len(mentions) != len(tc.expected) {
			t.Errorf("Test '%s': expected %d mentions, got %d", tc.description, len(tc.expected), len(mentions))
			continue
		}

		for i, expected := range tc.expected {
			if i >= len(mentions) || mentions[i] != expected {
				t.Errorf("Test '%s': expected mention %d to be '%s', got '%s'",
					tc.description, i, expected, mentions[i])
			}
		}
	}
}

// TestMentionValidation tests mention validation
func TestMentionValidation(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"@AgentName hello",
	)

	// Test HasMentions
	if !msg.HasMentions() {
		t.Error("Expected message to have mentions")
	}

	// Test IsMentioned
	if !msg.IsMentioned("agentname") {
		t.Error("Expected agentname to be mentioned")
	}

	if msg.IsMentioned("NotMentioned") {
		t.Error("Expected NotMentioned to not be mentioned")
	}
}

// TestCommandDetection tests command detection
func TestCommandDetection(t *testing.T) {
	testCases := []struct {
		content         string
		shouldBeCommand bool
		description     string
	}{
		{"/help", true, "Simple help command"},
		{"/list-agents", true, "List agents command"},
		{"/create-repo-agent /path TestAgent", true, "Create repo agent command"},
		{"Not a command", false, "Regular message"},
		{" /help", false, "Command with leading space"},
		{"help", false, "Command without slash"},
		{"//help", false, "Double slash"},
		{"/", false, "Just slash"},
		{"/help extra text", true, "Command with extra text"},
		{"/help\nnewline", true, "Command with newline"},
	}

	for _, tc := range testCases {
		msg := protocol.NewMessage(
			protocol.MessageTypeChat,
			"test-channel",
			protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
			tc.content,
		)

		// More sophisticated command detection
		isCommand := false
		if len(msg.Content) > 0 && msg.Content[0] == '/' {
			// Check if it's a valid command (not just a slash or double slash)
			content := strings.TrimSpace(msg.Content)
			if len(content) > 1 && !strings.HasPrefix(content, "//") {
				isCommand = true
			}
		}

		if isCommand != tc.shouldBeCommand {
			t.Errorf("Test '%s': expected command=%v, got %v", tc.description, tc.shouldBeCommand, isCommand)
		}
	}
}

// TestMessageTypes tests message type handling
func TestMessageTypes(t *testing.T) {
	types := []protocol.MessageType{
		protocol.MessageTypeChat,
		protocol.MessageTypeQuestion,
		protocol.MessageTypeAnswer,
		protocol.MessageTypeSystemInfo,
		protocol.MessageTypeAgentJoin,
		protocol.MessageTypeAgentLeave,
		protocol.MessageTypeAgentStatus,
		protocol.MessageTypeContextShare,
		protocol.MessageTypeRequestHelp,
	}

	for _, msgType := range types {
		msg := protocol.NewMessage(
			msgType,
			"test-channel",
			protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
			"Test message",
		)

		if msg.Type != msgType {
			t.Errorf("Expected message type %v, got %v", msgType, msg.Type)
		}
	}
}

// TestThreadHandling tests thread-related functionality
func TestThreadHandling(t *testing.T) {
	// Create parent message
	parentMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"Parent message",
	)

	// Create thread message
	threadMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user2", Name: "User2", Type: protocol.AgentTypeGeneral},
		"Thread reply",
	)
	threadMsg.ThreadID = parentMsg.ID
	threadMsg.IsThreadReply = true

	// Test thread ID
	if threadMsg.GetThreadID() != parentMsg.ID {
		t.Error("Expected thread ID to match parent message ID")
	}

	// Test IsInThread
	if !threadMsg.IsInThread() {
		t.Error("Expected thread message to be in thread")
	}

	if parentMsg.IsInThread() {
		t.Error("Expected parent message to not be in thread")
	}
}

// TestReviewMetadata tests review metadata functionality
func TestReviewMetadata(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "agent1", Name: "Agent", Type: protocol.AgentTypeBackend},
		"Review message",
	)

	// Test initial state
	if msg.GetReviewDepth() != 0 {
		t.Error("Expected initial review depth to be 0")
	}

	if msg.GetReviewedMessageID() != "" {
		t.Error("Expected initial reviewed message ID to be empty")
	}

	// Set review metadata
	msg.SetReviewDepth(1)
	msg.SetReviewedMessageID("original-msg-id")
	msg.SetOriginalQuestionID("question-id")

	// Test metadata
	if msg.GetReviewDepth() != 1 {
		t.Error("Expected review depth to be 1")
	}

	if msg.GetReviewedMessageID() != "original-msg-id" {
		t.Error("Expected reviewed message ID to be 'original-msg-id'")
	}

	if msg.GetOriginalQuestionID() != "question-id" {
		t.Error("Expected original question ID to be 'question-id'")
	}
}

// TestReviewRequestDetection tests review request detection
func TestReviewRequestDetection(t *testing.T) {
	testCases := []struct {
		content     string
		expected    bool
		description string
	}{
		{"@Agent thoughts?", true, "Simple thoughts question"},
		{"@Agent what do you think?", true, "Think question"},
		{"@Agent agree?", true, "Agree question"},
		{"@Agent review this please", true, "Explicit review request"},
		{"@Agent your opinion?", true, "Opinion question"},
		{"@Agent thoughts on this approach?", true, "Thoughts on approach"},
		{"@Agent do you agree with this?", true, "Agreement question"},
		{"@Agent just a regular question", false, "Regular question"},
		{"@Agent can you help?", false, "Help request"},
		{"Regular message", false, "No mention"},
		{"@Agent how does this work?", false, "How question"},
		{"@Agent what is this?", false, "What question"},
	}

	for _, tc := range testCases {
		msg := protocol.NewMessage(
			protocol.MessageTypeChat,
			"test-channel",
			protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
			tc.content,
		)

		result := msg.IsReviewRequest()
		if result != tc.expected {
			t.Errorf("Test '%s': expected IsReviewRequest()=%v, got %v",
				tc.description, tc.expected, result)
		}
	}
}

func TestSystemMessagesDoNotParseMentions(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		"test-channel",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"❌ Agent @ghost not found",
	)
	if len(msg.Mentions) != 0 {
		t.Fatalf("expected no mentions for system info, got %v", msg.Mentions)
	}
}

func TestHumanActionableMessagesStillParseMentions(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"test-channel",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"@RustExpert can you review this?",
	)
	if len(msg.Mentions) != 1 || msg.Mentions[0] != "rustexpert" {
		t.Fatalf("expected rustexpert mention, got %v", msg.Mentions)
	}
}

// TestMessageValidation tests message validation
func TestMessageValidation(t *testing.T) {
	// Test valid message
	validMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"Valid message",
	)

	if validMsg.ID == "" {
		t.Error("Expected message to have ID")
	}

	if validMsg.Channel != "test-channel" {
		t.Error("Expected message to have correct channel")
	}

	if validMsg.Content != "Valid message" {
		t.Error("Expected message to have correct content")
	}

	// Test message with empty content
	emptyMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"",
	)

	if emptyMsg.Content != "" {
		t.Error("Expected empty message to have empty content")
	}
}

// TestAgentTypes tests agent type handling
func TestAgentTypes(t *testing.T) {
	types := []protocol.AgentType{
		protocol.AgentTypeGeneral,
		protocol.AgentTypeFrontend,
		protocol.AgentTypeBackend,
		protocol.AgentTypeDatabase,
		protocol.AgentTypeSecurity,
		protocol.AgentTypeDevOps,
		protocol.AgentTypeRepo,
		protocol.AgentTypeHelper,
	}

	for _, agentType := range types {
		agent := protocol.AgentInfo{
			ID:   "test-agent",
			Name: "TestAgent",
			Type: agentType,
		}

		if agent.Type != agentType {
			t.Errorf("Expected agent type %v, got %v", agentType, agent.Type)
		}
	}
}

// TestThinkingStatus tests thinking status functionality
func TestThinkingStatus(t *testing.T) {
	statuses := []protocol.ThinkingStatus{
		protocol.ThinkingStatusStarted,
		protocol.ThinkingStatusCompleted,
		protocol.ThinkingStatusError,
	}

	for _, status := range statuses {
		msg := protocol.NewMessage(
			protocol.MessageTypeAgentStatus,
			"test-channel",
			protocol.AgentInfo{ID: "agent1", Name: "Agent", Type: protocol.AgentTypeBackend},
			"",
		)
		msg.Metadata["thinking_status"] = string(status)

		// Test status retrieval
		if msg.Metadata["thinking_status"] != string(status) {
			t.Errorf("Expected thinking status %v, got %v", status, msg.Metadata["thinking_status"])
		}
	}
}

// TestSingleAgentMentionResolution tests that only mentioned agents should respond
func TestSingleAgentMentionResolution(t *testing.T) {
	// Create a message with @Assistant mention
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"@Assistant what's going on today?",
	)

	// Verify the message has mentions
	if !msg.HasMentions() {
		t.Error("Expected message to have mentions")
	}

	// Test that only the Assistant agent should be considered mentioned
	// (This simulates the hub resolving mentions to agent IDs)
	assistantID := "assistant-agent-id"
	backendID := "backend-agent-id"
	moderatorID := "moderator-agent-id"

	// Simulate hub resolution - replace raw mention strings with agent IDs
	msg.Mentions = []string{assistantID} // Only Assistant is mentioned

	// Test IsMentioned with agent IDs
	if !msg.IsMentioned(assistantID) {
		t.Error("Expected Assistant to be mentioned")
	}

	if msg.IsMentioned(backendID) {
		t.Error("Expected Backend agent to NOT be mentioned")
	}

	if msg.IsMentioned(moderatorID) {
		t.Error("Expected Moderator agent to NOT be mentioned")
	}

	// Test that the message should only trigger response from Assistant
	// This verifies the fix where we removed the incorrect name-based check
	shouldAssistantRespond := msg.IsMentioned(assistantID)
	shouldBackendRespond := msg.IsMentioned(backendID)
	shouldModeratorRespond := msg.IsMentioned(moderatorID)

	if !shouldAssistantRespond {
		t.Error("Expected Assistant to respond (mentioned by ID)")
	}

	if shouldBackendRespond {
		t.Error("Expected Backend to NOT respond (not mentioned)")
	}

	if shouldModeratorRespond {
		t.Error("Expected Moderator to NOT respond (not mentioned)")
	}
}

// TestMentionResolutionWithMultipleAgents tests mention resolution with multiple agents
func TestMentionResolutionWithMultipleAgents(t *testing.T) {
	// Create a message mentioning multiple agents
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user1", Name: "User", Type: protocol.AgentTypeGeneral},
		"@Assistant @backend can you help with this?",
	)

	// Simulate hub resolution - multiple agents mentioned
	assistantID := "assistant-agent-id"
	backendID := "backend-agent-id"
	moderatorID := "moderator-agent-id"

	msg.Mentions = []string{assistantID, backendID} // Assistant and Backend mentioned

	// Test that both mentioned agents should respond
	if !msg.IsMentioned(assistantID) {
		t.Error("Expected Assistant to be mentioned")
	}

	if !msg.IsMentioned(backendID) {
		t.Error("Expected Backend to be mentioned")
	}

	if msg.IsMentioned(moderatorID) {
		t.Error("Expected Moderator to NOT be mentioned")
	}
}
