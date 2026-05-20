package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Mock hub client for integration testing
type mockHubClientIntegration struct {
	sentMessages []*protocol.Message
	subscribers  []chan *protocol.Message
	channels     map[string][]*protocol.Message
}

func (m *mockHubClientIntegration) SendMessage(msg *protocol.Message) error {
	m.sentMessages = append(m.sentMessages, msg)
	m.channels[msg.Channel] = append(m.channels[msg.Channel], msg)

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

func (m *mockHubClientIntegration) Subscribe(channelName string) (chan *protocol.Message, error) {
	subCh := make(chan *protocol.Message, 100)
	m.subscribers = append(m.subscribers, subCh)
	return subCh, nil
}

func (m *mockHubClientIntegration) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	messages := m.channels[channelName]
	if len(messages) > limit {
		return messages[len(messages)-limit:], nil
	}
	return messages, nil
}

func (m *mockHubClientIntegration) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return nil, nil
}

func (m *mockHubClientIntegration) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *mockHubClientIntegration) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (m *mockHubClientIntegration) BroadcastDirect(channelName string, msg *protocol.Message) {}
func (m *mockHubClientIntegration) GetAgentChannels(agentID string) []string { return []string{"general"} }
func (m *mockHubClientIntegration) GetChannelType(channelName string) protocol.ChannelType { return protocol.ChannelTypePublic }
func (m *mockHubClientIntegration) GetChannelSessionSummary(channel string) string         { return "" }
func (m *mockHubClientIntegration) ImageGenerationEnabled() bool                            { return false }
func (m *mockHubClientIntegration) GenerateAndPostImage(context.Context, string, protocol.AgentInfo, string, string) error {
	return nil
}

// TestEndToEndMessageFlow tests complete message flow from user to agent response
func TestEndToEndMessageFlow(t *testing.T) {
	// Create hub
	h := hub.NewHub()
	_, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create specialized agents
	backendAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"BackendExpert",
		[]string{"go", "apis", "microservices"},
		mockAI,
		h,
	)

	frontendAgent := agent.NewAgent(
		protocol.AgentTypeFrontend,
		"FrontendExpert",
		[]string{"react", "ui", "ux"},
		mockAI,
		h,
	)

	// Create test channel
	h.CreateChannel("integration-test", "Integration test channel", "test-project")

	// Register agents with hub
	err = h.RegisterAgent(&backendAgent.Info)
	if err != nil {
		t.Fatalf("Expected backend agent registration to succeed, got error: %v", err)
	}

	err = h.RegisterAgent(&frontendAgent.Info)
	if err != nil {
		t.Fatalf("Expected frontend agent registration to succeed, got error: %v", err)
	}

	// Start agents
	ctx := context.Background()
	channel := "integration-test"

	err = backendAgent.Start(ctx, channel)
	if err != nil {
		t.Fatalf("Expected backend agent to start, got error: %v", err)
	}

	err = frontendAgent.Start(ctx, channel)
	if err != nil {
		t.Fatalf("Expected frontend agent to start, got error: %v", err)
	}

	// Give agents time to start
	time.Sleep(100 * time.Millisecond)

	// Create user message
	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	// Test 1: Backend question
	backendQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@BackendExpert How should I structure my REST API?",
	)
	backendQuestion.Mentions = []string{backendAgent.Info.ID}

	// Send message
	err = h.SendMessage(backendQuestion)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Wait for response
	time.Sleep(500 * time.Millisecond)

	// Check if backend agent responded
	messages, err := h.GetMessages(channel, 10)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	backendResponded := false
	for _, msg := range messages {
		if msg.From.Name == "BackendExpert" && msg.Type == protocol.MessageTypeChat {
			backendResponded = true
			break
		}
	}

	if !backendResponded {
		t.Error("Expected backend agent to respond to question")
	}

	// Test 2: Frontend question
	frontendQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@FrontendExpert What's the best way to handle state in React?",
	)
	frontendQuestion.Mentions = []string{frontendAgent.Info.ID}

	// Send message
	err = h.SendMessage(frontendQuestion)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Wait for response
	time.Sleep(500 * time.Millisecond)

	// Check if frontend agent responded
	messages, err = h.GetMessages(channel, 10)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	frontendResponded := false
	for _, msg := range messages {
		if msg.From.Name == "FrontendExpert" && msg.Type == protocol.MessageTypeChat {
			frontendResponded = true
			break
		}
	}

	if !frontendResponded {
		t.Error("Expected frontend agent to respond to question")
	}

	// Cleanup
	backendAgent.Stop()
	frontendAgent.Stop()
}

// TestMultiAgentConversation tests conversation between multiple agents
func TestMultiAgentConversation(t *testing.T) {
	// Create hub
	h := hub.NewHub()

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create multiple agents
	agents := []*agent.Agent{
		agent.NewAgent(protocol.AgentTypeBackend, "BackendExpert", []string{"go", "apis"}, mockAI, h),
		agent.NewAgent(protocol.AgentTypeFrontend, "FrontendExpert", []string{"react", "ui"}, mockAI, h),
		agent.NewAgent(protocol.AgentTypeDatabase, "DatabaseExpert", []string{"sql", "postgres"}, mockAI, h),
	}

	// Create test channel
	h.CreateChannel("multi-agent-test", "Multi-agent test channel", "test-project")

	// Register all agents with hub
	for _, agent := range agents {
		err := h.RegisterAgent(&agent.Info)
		if err != nil {
			t.Fatalf("Expected agent %s registration to succeed, got error: %v", agent.Info.Name, err)
		}
	}

	// Start all agents
	ctx := context.Background()
	channel := "multi-agent-test"

	for _, agent := range agents {
		err := agent.Start(ctx, channel)
		if err != nil {
			t.Fatalf("Expected agent %s to start, got error: %v", agent.Info.Name, err)
		}
	}

	// Give agents time to start
	time.Sleep(100 * time.Millisecond)

	// Create user message that should trigger multiple agents
	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	// Send a general question that might interest multiple agents
	generalQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"How should I design a full-stack application with Go backend, React frontend, and PostgreSQL database?",
	)

	// Send message
	err := h.SendMessage(generalQuestion)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Wait for responses
	time.Sleep(1 * time.Second)

	// Check if multiple agents responded
	messages, err := h.GetMessages(channel, 20)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	respondedAgents := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type == protocol.MessageTypeChat && msg.From.Type != protocol.AgentTypeGeneral {
			respondedAgents[msg.From.Name] = true
		}
	}

	if len(respondedAgents) == 0 {
		t.Error("Expected at least one agent to respond to general question")
	}

	// Cleanup
	for _, agent := range agents {
		agent.Stop()
	}
}

func TestCustomChannelAutoRespondWithoutMention(t *testing.T) {
	h := hub.NewHub()
	mockAI := ai.NewMockProvider()

	backendAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"GoExpert",
		[]string{"go", "api", "backend", "service"},
		mockAI,
		h,
	)

	h.CreateChannelWithType("custom-runtime", "Custom channel", "", protocol.ChannelTypeCustom, "tester")
	if err := h.RegisterAgent(&backendAgent.Info); err != nil {
		t.Fatalf("register agent failed: %v", err)
	}
	if err := backendAgent.Start(context.Background(), "custom-runtime"); err != nil {
		t.Fatalf("start agent failed: %v", err)
	}
	defer backendAgent.Stop()

	time.Sleep(100 * time.Millisecond)
	userMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"custom-runtime",
		protocol.AgentInfo{ID: "user-1", Name: "Camron", Type: "human"},
		"please help debug this API latency in our backend service",
	)
	if err := h.SendMessage(userMsg); err != nil {
		t.Fatalf("send message failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	msgs, err := h.GetMessages("custom-runtime", 20)
	if err != nil {
		t.Fatalf("get messages failed: %v", err)
	}

	agentResponded := false
	for _, m := range msgs {
		if m.From.ID == backendAgent.Info.ID && m.Type == protocol.MessageTypeChat {
			agentResponded = true
			break
		}
	}
	if !agentResponded {
		t.Fatalf("expected auto-response from backend agent in custom channel without mention")
	}
}

// TestRepositoryAgentWorkflow tests complete repository agent workflow
func TestRepositoryAgentWorkflow(t *testing.T) {
	useIsolatedRepoStorage(t)
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test files
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`,
		"utils.go": `package main

import "strings"

func toUpperCase(s string) string {
    return strings.ToUpper(s)
}`,
		"README.md": `# Test Repository

This is a test repository for integration testing.`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create hub
	h := hub.NewHub()
	_, err = hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, h)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Register repository agent with hub
	err = h.RegisterAgent(&repoAgent.Info)
	if err != nil {
		t.Fatalf("Expected repository agent registration to succeed, got error: %v", err)
	}

	// Create test channel
	h.CreateChannel("repo-test", "Repository test channel", "test-project")

	// Start with indexing
	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "repo-test")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Start agent
	channel := "repo-test"
	err = repoAgent.Start(ctx, channel)
	if err != nil {
		t.Fatalf("Expected repository agent to start, got error: %v", err)
	}

	// Give agent time to start
	time.Sleep(100 * time.Millisecond)

	// Test repository questions
	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	// Test 1: Ask about repository structure
	structureQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@TestRepoAgent What files are in this repository?",
	)
	structureQuestion.Mentions = []string{repoAgent.Info.ID}

	err = h.SendMessage(structureQuestion)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Test 2: Ask about specific code
	codeQuestion := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@TestRepoAgent What does the toUpperCase function do?",
	)
	codeQuestion.Mentions = []string{repoAgent.Info.ID}

	err = h.SendMessage(codeQuestion)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Check if repository agent responded
	messages, err := h.GetMessages(channel, 10)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	repoResponded := false
	for _, msg := range messages {
		if msg.From.Name == "TestRepoAgent" && msg.Type == protocol.MessageTypeChat {
			repoResponded = true
			break
		}
	}

	if !repoResponded {
		t.Error("Expected repository agent to respond to questions")
	}

	// Cleanup
	repoAgent.Stop()

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestCommandIntegration tests command processing integration
func TestCommandIntegration(t *testing.T) {
	workspaceDir := t.TempDir()

	// Create hub
	h := hub.NewHub()
	_, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test messages with various commands
	_ = protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	commands := []string{
		"/help",
		"/list-agents",
		"/create-expert rust",
		fmt.Sprintf("/add-workspace %s", workspaceDir),
		"/list-workspaces",
	}

	// Create test channel
	h.CreateChannel("command-test", "Command test channel", "test-project")

	channel := "command-test"

	// Create command handler
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	for _, command := range commands {
		// Create test message with command
		msg := protocol.NewMessage(
			protocol.MessageTypeChat,
			channel,
			protocol.AgentInfo{ID: "user-123", Name: "TestUser", Type: protocol.AgentTypeGeneral},
			command,
		)

		// Process command
		ctx := context.Background()
		response, err := handler.ProcessCommand(ctx, msg)
		if err != nil {
			t.Errorf("Expected command '%s' to be processed successfully, got error: %v", command, err)
			continue
		}

		if response == nil {
			t.Errorf("Expected command '%s' to return a response", command)
			continue
		}

		// Verify response contains expected content
		if !strings.Contains(response.Content, "error") &&
			!strings.Contains(response.Content, "success") &&
			!strings.Contains(response.Content, "Available") &&
			!strings.Contains(response.Content, "No") &&
			!strings.Contains(response.Content, "Workspaces") &&
			!strings.Contains(response.Content, "Adding") &&
			!strings.Contains(response.Content, "Added") &&
			!strings.Contains(response.Content, "Created") &&
			!strings.Contains(response.Content, "Unknown") {
			t.Errorf("Expected command '%s' response to contain meaningful content. Got: %s", command, response.Content)
		}
	}
}

// TestChannelSwitching tests switching between different channels
func TestChannelSwitching(t *testing.T) {
	// Create hub
	h := hub.NewHub()

	// Create channels
	channel1 := h.CreateChannel("channel1", "Test Channel 1", "test-project")
	channel2 := h.CreateChannel("channel2", "Test Channel 2", "test-project")

	if channel1 == nil || channel2 == nil {
		t.Fatal("Expected channels to be created")
	}

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create agent
	testAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"TestAgent",
		[]string{"go", "apis"},
		mockAI,
		h,
	)

	// Register agent with hub
	err := h.RegisterAgent(&testAgent.Info)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	// Start agent in first channel
	ctx := context.Background()
	err = testAgent.Start(ctx, "channel1")
	if err != nil {
		t.Fatalf("Expected agent to start in channel1, got error: %v", err)
	}

	// Give agent time to start
	time.Sleep(100 * time.Millisecond)

	// Send message to first channel
	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	msg1 := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"channel1",
		userInfo,
		"@TestAgent Hello from channel 1",
	)
	// The mentions should be parsed automatically by NewMessage

	err = h.SendMessage(msg1)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Check if agent responded in first channel
	messages1, err := h.GetMessages("channel1", 10)
	if err != nil {
		t.Fatalf("Expected to get messages from channel1, got error: %v", err)
	}

	respondedInChannel1 := false
	for _, msg := range messages1 {
		if msg.From.Name == "TestAgent" && msg.Type == protocol.MessageTypeChat {
			respondedInChannel1 = true
			break
		}
	}

	if !respondedInChannel1 {
		t.Error("Expected agent to respond in channel1")
	}

	// Send message to second channel (agent should not respond)
	msg2 := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"channel2",
		userInfo,
		"@TestAgent Hello from channel 2",
	)
	msg2.Mentions = []string{testAgent.Info.ID}

	err = h.SendMessage(msg2)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Check if agent did NOT respond in second channel
	messages2, err := h.GetMessages("channel2", 10)
	if err != nil {
		t.Fatalf("Expected to get messages from channel2, got error: %v", err)
	}

	respondedInChannel2 := false
	for _, msg := range messages2 {
		if msg.From.Name == "TestAgent" && msg.Type == protocol.MessageTypeChat {
			respondedInChannel2 = true
			break
		}
	}

	if respondedInChannel2 {
		t.Error("Expected agent NOT to respond in channel2 (different channel)")
	}

	// Cleanup
	testAgent.Stop()
}

// TestIntegrationConcurrentOperations tests concurrent operations across the system
func TestIntegrationConcurrentOperations(t *testing.T) {
	// Create hub
	h := hub.NewHub()

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create multiple agents
	agents := []*agent.Agent{
		agent.NewAgent(protocol.AgentTypeBackend, "BackendExpert", []string{"go", "apis"}, mockAI, h),
		agent.NewAgent(protocol.AgentTypeFrontend, "FrontendExpert", []string{"react", "ui"}, mockAI, h),
		agent.NewAgent(protocol.AgentTypeDatabase, "DatabaseExpert", []string{"sql", "postgres"}, mockAI, h),
	}

	// Create channel first
	ctx := context.Background()
	channel := "concurrent-test"
	h.CreateChannel(channel, "Test channel for concurrent operations", "test-project")

	// Start all agents
	for _, agent := range agents {
		err := agent.Start(ctx, channel)
		if err != nil {
			t.Fatalf("Expected agent %s to start, got error: %v", agent.Info.Name, err)
		}
	}

	// Give agents time to start
	time.Sleep(100 * time.Millisecond)

	// Create concurrent messages
	numMessages := 10
	done := make(chan bool, numMessages)

	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	for i := 0; i < numMessages; i++ {
		go func(i int) {
			msg := protocol.NewMessage(
				protocol.MessageTypeQuestion,
				channel,
				userInfo,
				fmt.Sprintf("Concurrent question %d", i),
			)

			err := h.SendMessage(msg)
			if err != nil {
				t.Errorf("Expected concurrent message %d to be sent, got error: %v", i, err)
			}
			done <- true
		}(i)
	}

	// Wait for all messages to be sent
	for i := 0; i < numMessages; i++ {
		select {
		case <-done:
			// Message sent successfully
		case <-time.After(5 * time.Second):
			t.Error("Test timed out waiting for concurrent messages")
			// Cleanup
			for _, agent := range agents {
				agent.Stop()
			}
			return
		}
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Verify messages were processed
	messages, err := h.GetMessages(channel, 50)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	if len(messages) < numMessages {
		t.Errorf("Expected at least %d messages, got %d", numMessages, len(messages))
	}

	// Cleanup
	for _, agent := range agents {
		agent.Stop()
	}
}

// TestErrorRecovery tests system recovery from errors
func TestErrorRecovery(t *testing.T) {
	// Create hub
	h := hub.NewHub()

	// Create mock AI provider
	mockAI := ai.NewMockProvider()

	// Create agent
	testAgent := agent.NewAgent(
		protocol.AgentTypeBackend,
		"TestAgent",
		[]string{"go", "apis"},
		mockAI,
		h,
	)

	// Create channel first
	ctx := context.Background()
	channel := "error-test"
	h.CreateChannel(channel, "Test channel for error recovery", "test-project")

	// Start agent
	err := testAgent.Start(ctx, channel)
	if err != nil {
		t.Fatalf("Expected agent to start, got error: %v", err)
	}

	// Give agent time to start
	time.Sleep(100 * time.Millisecond)

	// Test with invalid message (should not crash system)
	userInfo := protocol.AgentInfo{
		ID:   "user-123",
		Name: "TestUser",
		Type: protocol.AgentTypeGeneral,
	}

	// Send normal message first
	normalMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"@TestAgent Normal question",
	)
	normalMsg.Mentions = []string{testAgent.Info.ID}

	err = h.SendMessage(normalMsg)
	if err != nil {
		t.Fatalf("Expected normal message to be sent, got error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send message with empty content
	emptyMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		"", // Empty content
	)
	emptyMsg.Mentions = []string{testAgent.Info.ID}

	err = h.SendMessage(emptyMsg)
	if err != nil {
		t.Fatalf("Expected empty message to be sent, got error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send message with very long content
	longContent := strings.Repeat("a", 10000) // 10KB content
	longMsg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		channel,
		userInfo,
		longContent,
	)
	longMsg.Mentions = []string{testAgent.Info.ID}

	err = h.SendMessage(longMsg)
	if err != nil {
		t.Fatalf("Expected long message to be sent, got error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// System should still be responsive
	messages, err := h.GetMessages(channel, 10)
	if err != nil {
		t.Fatalf("Expected to get messages after error conditions, got error: %v", err)
	}

	// Should have received at least the normal message
	if len(messages) == 0 {
		t.Error("Expected system to remain responsive after error conditions")
	}

	// Cleanup
	testAgent.Stop()
}
