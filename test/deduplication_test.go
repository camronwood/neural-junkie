package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Mock hub client for testing
type mockHubClient struct {
	sentMessages []*protocol.Message
	subChannel   chan *protocol.Message
	mutex        sync.RWMutex
}

func (m *mockHubClient) SendMessage(msg *protocol.Message) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockHubClient) Subscribe(channelName string) (chan *protocol.Message, error) {
	return m.subChannel, nil
}

func (m *mockHubClient) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return nil, nil
}

func (m *mockHubClient) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return nil, nil
}

func (m *mockHubClient) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *mockHubClient) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (m *mockHubClient) BroadcastDirect(channelName string, msg *protocol.Message) {}
func (m *mockHubClient) GetAgentChannels(agentID string) []string { return []string{"general"} }
func (m *mockHubClient) GetChannelType(channelName string) protocol.ChannelType { return protocol.ChannelTypePublic }

func (m *mockHubClient) GetSentMessages() []*protocol.Message {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	// Return a copy to avoid race conditions
	result := make([]*protocol.Message, len(m.sentMessages))
	copy(result, m.sentMessages)
	return result
}

func TestAgentDeduplication(t *testing.T) {
	// Create mock hub
	hub := &mockHubClient{
		sentMessages: make([]*protocol.Message, 0),
		subChannel:   make(chan *protocol.Message, 100),
	}

	// Create test agent
	mockAI := ai.NewMockProvider()
	testAgent := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"Test Agent",
		[]string{"security", "testing"},
		mockAI,
		hub,
	)

	// Start agent
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := testAgent.Start(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	// Create a test message that should trigger a response
	testMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "Test User",
			Type: protocol.AgentTypeGeneral,
		},
		"How do I prevent security vulnerabilities?",
	)

	// Send the SAME message 5 times (simulating duplicate delivery)
	for i := 0; i < 5; i++ {
		hub.subChannel <- testMsg
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Stop the agent to ensure all processing is complete
	testAgent.Stop()

	// Check how many actual chat responses were sent (exclude thinking status messages)
	sentMessages := hub.GetSentMessages()
	chatResponses := 0
	for _, msg := range sentMessages {
		if msg.Type == protocol.MessageTypeChat {
			chatResponses++
		}
	}

	t.Logf("Sent same message 5 times")
	t.Logf("Agent sent %d total messages, %d chat responses", len(sentMessages), chatResponses)

	// Should only respond ONCE to the same message ID
	if chatResponses != 1 {
		t.Errorf("Expected 1 response, got %d chat responses", chatResponses)
		for i, msg := range sentMessages {
			content := msg.Content
			replyTo := msg.ReplyTo
			if len(content) > 50 {
				content = content[:50]
			}
			if len(replyTo) > 8 {
				replyTo = replyTo[:8]
			}
			t.Logf("  Response %d: %s (ReplyTo: %s)", i+1, content, replyTo)
		}
		t.Fail()
	} else {
		t.Logf("✅ PASS: Agent correctly deduplicated and sent only 1 response")
	}
}

func TestMultipleMessagesNoDeduplication(t *testing.T) {
	// Create mock hub
	hub := &mockHubClient{
		sentMessages: make([]*protocol.Message, 0),
		subChannel:   make(chan *protocol.Message, 100),
	}

	// Create test agent
	mockAI := ai.NewMockProvider()
	testAgent := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"Test Agent",
		[]string{"security", "testing"},
		mockAI,
		hub,
	)

	// Start agent
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := testAgent.Start(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	// Send 3 DIFFERENT messages (different IDs)
	for i := 0; i < 3; i++ {
		testMsg := protocol.NewMessage(
			protocol.MessageTypeQuestion,
			"test-channel",
			protocol.AgentInfo{
				ID:   "user-123",
				Name: "Test User",
				Type: protocol.AgentTypeGeneral,
			},
			fmt.Sprintf("Security question #%d?", i+1),
		)
		hub.subChannel <- testMsg
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Stop the agent to ensure all processing is complete
	testAgent.Stop()

	// Check how many actual chat responses were sent (exclude thinking status messages)
	sentMessages := hub.GetSentMessages()
	chatResponses := 0
	for _, msg := range sentMessages {
		if msg.Type == protocol.MessageTypeChat {
			chatResponses++
		}
	}

	t.Logf("Sent 3 different messages")
	t.Logf("Agent sent %d total messages, %d chat responses", len(sentMessages), chatResponses)

	// Should respond to all 3 different messages
	if chatResponses != 3 {
		t.Errorf("Expected 3 responses, got %d chat responses", chatResponses)
		t.Fail()
	} else {
		t.Logf("✅ PASS: Agent correctly responded to all 3 different messages")
	}
}

func TestConcurrentDuplicateMessages(t *testing.T) {
	// Create mock hub
	hub := &mockHubClient{
		sentMessages: make([]*protocol.Message, 0),
		subChannel:   make(chan *protocol.Message, 100),
	}

	// Create test agent
	mockAI := ai.NewMockProvider()
	testAgent := agent.NewAgent(
		protocol.AgentTypeSecurity,
		"Test Agent",
		[]string{"security", "testing"},
		mockAI,
		hub,
	)

	// Start agent
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := testAgent.Start(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	// Create a test message
	testMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "Test User",
			Type: protocol.AgentTypeGeneral,
		},
		"Concurrent security test?",
	)

	// Send the SAME message 10 times CONCURRENTLY
	for i := 0; i < 10; i++ {
		go func() {
			hub.subChannel <- testMsg
		}()
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Stop the agent to ensure all processing is complete
	testAgent.Stop()

	// Check how many actual chat responses were sent (exclude thinking status messages)
	sentMessages := hub.GetSentMessages()
	chatResponses := 0
	for _, msg := range sentMessages {
		if msg.Type == protocol.MessageTypeChat {
			chatResponses++
		}
	}

	t.Logf("Sent same message 10 times concurrently")
	t.Logf("Agent sent %d total messages, %d chat responses", len(sentMessages), chatResponses)

	// Should only respond ONCE even with concurrent duplicates
	if chatResponses != 1 {
		t.Errorf("Expected 1 response, got %d chat responses (thread safety issue!)", chatResponses)
		t.Fail()
	} else {
		t.Logf("✅ PASS: Agent correctly handled concurrent duplicates (thread-safe)")
	}
}
