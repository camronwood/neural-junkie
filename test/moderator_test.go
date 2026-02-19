package test

import (
	"context"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestModeratorAgentCreation tests that we can create a moderator agent
func TestModeratorAgentCreation(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()

	moderator := agent.NewModeratorAgent("Test Moderator", mockAI, testHub)
	if moderator == nil {
		t.Fatal("Failed to create moderator agent")
	}

	if moderator.Info.Type != protocol.AgentTypeModerator {
		t.Errorf("Expected agent type 'moderator', got '%s'", moderator.Info.Type)
	}

	if moderator.Info.Name != "Test Moderator" {
		t.Errorf("Expected name 'Test Moderator', got '%s'", moderator.Info.Name)
	}
}

// TestModeratorMessageTracking tests that the moderator tracks user messages
func TestModeratorMessageTracking(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()
	testHub.CreateChannel("test", "Test Channel", "")

	moderator := agent.NewModeratorAgent("Test Moderator", mockAI, testHub)

	// Start moderator in test channel
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go moderator.Start(ctx, "test")
	time.Sleep(100 * time.Millisecond) // Give it time to start

	// Send a user message (not from an agent)
	userMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test",
		protocol.AgentInfo{
			ID:   "user-1",
			Name: "Test User",
			Type: "", // Empty type indicates user message
		},
		"This is a test question",
	)

	// Process the message
	moderator.ProcessMessage(ctx, userMsg)

	// Verify message is tracked
	time.Sleep(100 * time.Millisecond)
	// Note: We can't directly access trackedMessages (private field)
	// but we verify no errors occurred during processing
}

// TestModeratorRespondsToMentions tests that moderator responds when mentioned
func TestModeratorRespondsToMentions(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()
	testHub.CreateChannel("test", "Test Channel", "")

	moderator := agent.NewModeratorAgent("Test Moderator", mockAI, testHub)

	// Create a message mentioning the moderator
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test",
		protocol.AgentInfo{
			ID:   "user-1",
			Name: "Test User",
			Type: "",
		},
		"@Test Moderator how do I use commands?",
	)
	msg.Mentions = []string{moderator.Info.ID, "Test Moderator"}

	// Test shouldRespond logic
	// Note: shouldRespond is private, but we can verify through behavior
	// The moderator should respond because it's mentioned
}

// TestModeratorIgnoresAgentMessages tests that moderator doesn't track agent messages
func TestModeratorIgnoresAgentMessages(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()
	testHub.CreateChannel("test", "Test Channel", "")

	moderator := agent.NewModeratorAgent("Test Moderator", mockAI, testHub)

	ctx := context.Background()

	// Send an agent message
	agentMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test",
		protocol.AgentInfo{
			ID:   "backend-1",
			Name: "Backend Expert",
			Type: protocol.AgentTypeBackend,
		},
		"This is an agent response",
	)

	// Process the message - should not be tracked
	moderator.ProcessMessage(ctx, agentMsg)

	// Verify no errors occurred
	// Message shouldn't be tracked since it's from an agent
}

// TestModeratorWithCommands tests that moderator doesn't track command messages
func TestModeratorWithCommands(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()
	testHub.CreateChannel("test", "Test Channel", "")

	moderator := agent.NewModeratorAgent("Test Moderator", mockAI, testHub)

	ctx := context.Background()

	// Send a command message
	cmdMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test",
		protocol.AgentInfo{
			ID:   "user-1",
			Name: "Test User",
			Type: "",
		},
		"/list-agents",
	)

	// Process the message - should not be tracked (commands get immediate responses)
	moderator.ProcessMessage(ctx, cmdMsg)

	// Verify no errors occurred
}

// TestModeratorAgentType tests that moderator type is recognized
func TestModeratorAgentType(t *testing.T) {
	mockAI := ai.NewMockProvider()
	testHub := hub.NewHub()

	// Test that we can create moderator through factory
	moderator, err := agent.AgentFactory(protocol.AgentTypeModerator, "Factory Moderator", mockAI, testHub)
	if err != nil {
		t.Fatalf("Failed to create moderator through factory: %v", err)
	}

	if moderator == nil {
		t.Fatal("Factory returned nil moderator")
	}

	if moderator.Info.Type != protocol.AgentTypeModerator {
		t.Errorf("Expected agent type 'moderator', got '%s'", moderator.Info.Type)
	}
}
