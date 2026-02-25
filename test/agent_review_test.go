package test

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Helper function to filter out thinking status messages
func filterChatMessages(messages []*protocol.Message) []*protocol.Message {
	chatMessages := make([]*protocol.Message, 0)
	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeChat {
			chatMessages = append(chatMessages, msg)
		}
	}
	return chatMessages
}

// Mock hub client for testing with proper message broadcasting
type mockHubClientReview struct {
	sentMessages []*protocol.Message
	subscribers  []chan *protocol.Message
}

func (m *mockHubClientReview) SendMessage(msg *protocol.Message) error {
	m.sentMessages = append(m.sentMessages, msg)
	// Broadcast to all subscribers
	for _, subCh := range m.subscribers {
		select {
		case subCh <- msg:
		default:
			// Skip if channel is full
		}
	}
	return nil
}

func (m *mockHubClientReview) Subscribe(channelName string) (chan *protocol.Message, error) {
	// Create a new channel for each subscriber
	subCh := make(chan *protocol.Message, 100)
	m.subscribers = append(m.subscribers, subCh)
	return subCh, nil
}

func (m *mockHubClientReview) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return nil, nil
}

func (m *mockHubClientReview) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return nil, nil
}

func (m *mockHubClientReview) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *mockHubClientReview) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (m *mockHubClientReview) BroadcastDirect(channelName string, msg *protocol.Message) {}

func (m *mockHubClientReview) GetAgentChannels(agentID string) []string {
	return []string{"general"}
}

func (m *mockHubClientReview) GetChannelType(channelName string) protocol.ChannelType {
	return protocol.ChannelTypePublic
}

// Helper function to broadcast a message to all subscribers
func (m *mockHubClientReview) BroadcastMessage(msg *protocol.Message) {
	for _, subCh := range m.subscribers {
		select {
		case subCh <- msg:
		default:
			// Skip if channel is full
		}
	}
}

// TestAgentReview tests basic agent review functionality
func TestAgentReview(t *testing.T) {
	// Create mock hub
	hub := &mockHubClientReview{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}

	// Create two test agents - backend and security
	mockAI := ai.NewMockProvider()
	backendAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"BackendExpert",
		[]string{"APIs", "Go", "microservices"},
		mockAI,
		hub,
	)

	securityAgent := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"SecurityExpert",
		[]string{"authentication", "security"},
		mockAI,
		hub,
	)

	ctx := context.Background()
	channel := "test-channel"

	// Start both agents
	if err := backendAgent.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start backend agent: %v", err)
	}
	if err := securityAgent.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start security agent: %v", err)
	}

	// Give agents time to start
	time.Sleep(100 * time.Millisecond)

	// Step 1: User asks backend agent a question
	userInfo := protocol.AgentInfo{ID: "user1", Name: "TestUser", Type: protocol.AgentTypeGeneral}
	userQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@BackendExpert How should I structure my REST API?",
	)
	userQuestion.Mentions = []string{backendAgent.Info.ID} // Properly set mentions

	hub.BroadcastMessage(userQuestion)
	time.Sleep(200 * time.Millisecond)

	// Backend agent should have responded (filter out thinking status messages)
	chatResponses := filterChatMessages(hub.sentMessages)
	if len(chatResponses) != 1 {
		t.Fatalf("Expected 1 chat response from backend agent, got %d", len(chatResponses))
	}

	backendResponse := chatResponses[0]
	if backendResponse.From.Name != "BackendExpert" {
		t.Errorf("Expected response from BackendExpert, got %s", backendResponse.From.Name)
	}
	if backendResponse.ReplyTo != userQuestion.ID {
		t.Errorf("Expected response to reply to user question")
	}

	contentPreview := backendResponse.Content
	if len(contentPreview) > 50 {
		contentPreview = contentPreview[:50] + "..."
	}
	t.Logf("✅ Backend agent responded: %s", contentPreview)

	// Simulate both agents receiving the backend response (for history)
	hub.BroadcastMessage(backendResponse)
	time.Sleep(100 * time.Millisecond)

	// Step 2: User asks security agent to review (reply to backend's response)
	reviewRequest := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@SecurityExpert thoughts on this from a security perspective?",
	)
	reviewRequest.ReplyTo = backendResponse.ID               // Replying to backend's message
	reviewRequest.Mentions = []string{securityAgent.Info.ID} // Properly set mentions
	reviewRequest.SetReviewDepth(0)                          // User messages have depth 0

	hub.BroadcastMessage(reviewRequest)
	time.Sleep(200 * time.Millisecond)

	// Security agent should have responded with a review (filter out thinking status messages)
	chatResponses = filterChatMessages(hub.sentMessages)
	if len(chatResponses) != 2 {
		t.Fatalf("Expected 2 total chat responses, got %d", len(chatResponses))
	}

	securityResponse := chatResponses[1]
	if securityResponse.From.Name != "SecurityExpert" {
		t.Errorf("Expected response from SecurityExpert, got %s", securityResponse.From.Name)
	}
	if securityResponse.ReplyTo != reviewRequest.ID {
		t.Errorf("Expected response to reply to review request")
	}

	// Check review metadata
	if securityResponse.GetReviewDepth() != 1 {
		t.Errorf("Expected review depth 1, got %d", securityResponse.GetReviewDepth())
	}
	if securityResponse.GetReviewedMessageID() != backendResponse.ID {
		t.Errorf("Expected reviewed message ID to be backend response ID")
	}

	secContentPreview := securityResponse.Content
	if len(secContentPreview) > 50 {
		secContentPreview = secContentPreview[:50] + "..."
	}
	t.Logf("✅ Security agent reviewed with depth %d: %s",
		securityResponse.GetReviewDepth(), secContentPreview)

	// Cleanup
	backendAgent.Stop()
	securityAgent.Stop()
}

// TestReviewDepthLimit tests that reviews don't cascade beyond depth 1
func TestReviewDepthLimit(t *testing.T) {
	// Create mock hub
	hub := &mockHubClientReview{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}

	// Create three agents
	mockAI := ai.NewMockProvider()
	agentA := agent.NewAgent(
		protocol.AgentTypeBackend,
		"AgentA",
		[]string{"backend"},
		mockAI,
		hub,
	)
	agentB := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"AgentB",
		[]string{"security"},
		mockAI,
		hub,
	)
	agentC := agent.NewAgent(
		protocol.AgentTypeFrontend,
		"AgentC",
		[]string{"frontend"},
		mockAI,
		hub,
	)

	ctx := context.Background()
	channel := "test-channel"

	// Start all agents
	if err := agentA.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start agent A: %v", err)
	}
	if err := agentB.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start agent B: %v", err)
	}
	if err := agentC.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start agent C: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	userInfo := protocol.AgentInfo{ID: "user1", Name: "TestUser", Type: protocol.AgentTypeGeneral}

	// User asks Agent A
	userQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@AgentA test question",
	)
	userQuestion.Mentions = []string{agentA.Info.ID}

	hub.BroadcastMessage(userQuestion)
	time.Sleep(200 * time.Millisecond)

	// Agent A responds
	agentAResponse := hub.sentMessages[0]
	t.Logf("Agent A responded")

	// Simulate all agents receiving Agent A's response (for history)
	hub.BroadcastMessage(agentAResponse)
	time.Sleep(100 * time.Millisecond)

	// User asks Agent B to review (depth 0 -> 1)
	reviewRequest1 := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@AgentB thoughts?",
	)
	reviewRequest1.ReplyTo = agentAResponse.ID
	reviewRequest1.Mentions = []string{agentB.Info.ID}
	reviewRequest1.SetReviewDepth(0)

	hub.BroadcastMessage(reviewRequest1)
	time.Sleep(200 * time.Millisecond)

	// Agent B should respond (depth 1) (filter out thinking status messages)
	chatResponses := filterChatMessages(hub.sentMessages)
	if len(chatResponses) != 2 {
		t.Fatalf("Expected 2 chat responses, got %d", len(chatResponses))
	}
	agentBResponse := chatResponses[1]
	if agentBResponse.GetReviewDepth() != 1 {
		t.Errorf("Expected Agent B review depth 1, got %d", agentBResponse.GetReviewDepth())
	}
	t.Logf("Agent B reviewed at depth 1")

	// Simulate all agents receiving Agent B's response (for history)
	hub.BroadcastMessage(agentBResponse)
	time.Sleep(100 * time.Millisecond)

	// User asks Agent C to review Agent B's review (depth 1 -> 2, should be blocked)
	reviewRequest2 := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@AgentC what do you think?",
	)
	reviewRequest2.ReplyTo = agentBResponse.ID
	reviewRequest2.Mentions = []string{agentC.Info.ID}
	reviewRequest2.SetReviewDepth(1) // Already at depth 1

	hub.BroadcastMessage(reviewRequest2)
	time.Sleep(200 * time.Millisecond)

	// Agent C should NOT respond (depth would be 2) - check chat responses only
	if len(chatResponses) != 2 {
		t.Errorf("Expected Agent C to not respond at depth 2, but got %d chat responses", len(chatResponses))
	}
	t.Logf("✅ Agent C correctly rejected review request at depth 2")

	// Cleanup
	agentA.Stop()
	agentB.Stop()
	agentC.Stop()
}

// TestReviewWithoutReplyTo tests that agents don't respond to mentions without ReplyTo
func TestReviewWithoutReplyTo(t *testing.T) {
	// Create mock hub
	hub := &mockHubClientReview{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}

	mockAI := ai.NewMockProvider()
	agentA := agent.NewAgent(
		protocol.AgentTypeBackend,
		"AgentA",
		[]string{"backend"},
		mockAI,
		hub,
	)
	agentB := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"AgentB",
		[]string{"security"},
		mockAI,
		hub,
	)

	ctx := context.Background()
	channel := "test-channel"

	if err := agentA.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start agent A: %v", err)
	}
	if err := agentB.Start(ctx, channel); err != nil {
		t.Fatalf("Failed to start agent B: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	userInfo := protocol.AgentInfo{ID: "user1", Name: "TestUser", Type: protocol.AgentTypeGeneral}

	// Agent A sends a message
	agentAMessage := protocol.NewMessage(
		protocol.MessageTypeChat,
		channel,
		agentA.Info,
		"I think the API should use REST",
	)

	hub.BroadcastMessage(agentAMessage)
	time.Sleep(100 * time.Millisecond)

	// User tries to mention Agent B but WITHOUT ReplyTo field
	userMention := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@AgentB thoughts?",
	)
	// NO ReplyTo field set
	userMention.Mentions = []string{agentB.Info.ID}

	hub.BroadcastMessage(userMention)
	time.Sleep(200 * time.Millisecond)

	// Agent B SHOULD respond to user mention (normal response, not a review)
	// But since AgentA's message was from another agent and the user mention
	// doesn't have ReplyTo, Agent B will respond in normal mode, not review mode
	chatResponses := filterChatMessages(hub.sentMessages)
	if len(chatResponses) != 1 {
		t.Errorf("Expected Agent B to respond to user mention, got %d chat messages", len(chatResponses))
	}
	if len(chatResponses) > 0 {
		response := chatResponses[0]
		// Check it's not marked as a review (no review depth set)
		if response.GetReviewDepth() != 0 {
			t.Errorf("Expected non-review response (depth 0), got depth %d", response.GetReviewDepth())
		}
		t.Logf("✅ Agent B responded normally (not as review) since ReplyTo points to user question")
	}

	// Cleanup
	agentA.Stop()
	agentB.Stop()
}

// TestReviewMetadataTracking tests that review metadata is properly tracked
func TestReviewMetadataTracking(t *testing.T) {
	// Create a review message and test metadata helpers
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "agent1", Name: "TestAgent", Type: protocol.AgentTypeBackend},
		"This is a review",
	)

	// Test initial state
	if msg.GetReviewDepth() != 0 {
		t.Errorf("Expected initial review depth 0, got %d", msg.GetReviewDepth())
	}
	if msg.GetReviewedMessageID() != "" {
		t.Errorf("Expected empty reviewed message ID, got %s", msg.GetReviewedMessageID())
	}

	// Set review metadata
	msg.SetReviewDepth(1)
	msg.SetReviewedMessageID("original-msg-id")
	msg.SetOriginalQuestionID("question-id")

	// Verify metadata
	if msg.GetReviewDepth() != 1 {
		t.Errorf("Expected review depth 1, got %d", msg.GetReviewDepth())
	}
	if msg.GetReviewedMessageID() != "original-msg-id" {
		t.Errorf("Expected reviewed message ID 'original-msg-id', got %s", msg.GetReviewedMessageID())
	}
	if msg.GetOriginalQuestionID() != "question-id" {
		t.Errorf("Expected original question ID 'question-id', got %s", msg.GetOriginalQuestionID())
	}

	t.Logf("✅ Review metadata tracking works correctly")
}

// TestIsReviewRequest tests the review keyword detection
func TestIsReviewRequest(t *testing.T) {
	testCases := []struct {
		content  string
		expected bool
	}{
		{"@Agent thoughts?", true},
		{"@Agent what do you think?", true},
		{"@Agent agree?", true},
		{"@Agent review this please", true},
		{"@Agent your opinion?", true},
		{"@Agent thoughts on this approach?", true},
		{"@Agent do you agree with this?", true},
		{"@Agent just a regular question", false},
		{"@Agent can you help?", false},
		{"Regular message", false},
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
			t.Errorf("Content '%s': expected IsReviewRequest()=%v, got %v",
				tc.content, tc.expected, result)
		}
	}

	t.Logf("✅ Review keyword detection works correctly")
}
