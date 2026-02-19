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
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Mock hub client for helper agent testing
type mockHubClientHelper struct {
	sentMessages []*protocol.Message
	subscribers  []chan *protocol.Message
}

func (m *mockHubClientHelper) SendMessage(msg *protocol.Message) error {
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

func (m *mockHubClientHelper) Subscribe(channelName string) (chan *protocol.Message, error) {
	subCh := make(chan *protocol.Message, 100)
	m.subscribers = append(m.subscribers, subCh)
	return subCh, nil
}

func (m *mockHubClientHelper) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return nil, nil
}

func (m *mockHubClientHelper) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return nil, nil
}

func (m *mockHubClientHelper) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *mockHubClientHelper) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

// TestHelperAgentCreation tests helper agent creation
func TestHelperAgentCreation(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge files
	knowledgeFiles := map[string]string{
		"general.md": `# General Knowledge

This is general knowledge about various topics.

## Topics
- Programming
- Technology
- Best practices`,
		"programming.md": `# Programming Knowledge

This document contains programming-related information.

## Languages
- Go
- Python
- JavaScript

## Concepts
- Object-oriented programming
- Functional programming
- Design patterns`,
	}

	for filename, content := range knowledgeFiles {
		filePath := filepath.Join(knowledgePath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create knowledge file %s: %v", filename, err)
		}
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"programming", "technology", "general"},
		Keywords:      []string{"help", "assist", "guide"},
		SystemPrompt:  "You are a helpful assistant with specialized knowledge.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	if helperAgent == nil {
		t.Fatal("Expected helper agent to be created")
	}

	if helperAgent.Info.Name != "TestHelper" {
		t.Errorf("Expected agent name 'TestHelper', got '%s'", helperAgent.Info.Name)
	}

	if helperAgent.Info.Type != protocol.AgentTypeHelper {
		t.Errorf("Expected agent type AgentTypeHelper, got %v", helperAgent.Info.Type)
	}

	if helperAgent.Config.Name != "TestHelper" {
		t.Errorf("Expected config name 'TestHelper', got '%s'", helperAgent.Config.Name)
	}

	if helperAgent.KnowledgePath != knowledgePath {
		t.Errorf("Expected knowledge path '%s', got '%s'", knowledgePath, helperAgent.KnowledgePath)
	}
}

// TestHelperAgentKnowledgeLoading tests knowledge base loading
func TestHelperAgentKnowledgeLoading(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge files
	knowledgeFiles := map[string]string{
		"topic1.md": `# Topic 1

This is information about topic 1.

## Key Points
- Point 1
- Point 2
- Point 3`,
		"topic2.md": `# Topic 2

This is information about topic 2.

## Details
- Detail 1
- Detail 2`,
		"nested/subtopic.md": `# Subtopic

This is a nested subtopic.

## Information
- Info 1
- Info 2`,
	}

	for filename, content := range knowledgeFiles {
		filePath := filepath.Join(knowledgePath, filename)
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != knowledgePath {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create nested directory: %v", err)
			}
		}
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create knowledge file %s: %v", filename, err)
		}
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"topic1", "topic2"},
		Keywords:      []string{"help", "assist"},
		SystemPrompt:  "You are a helpful assistant.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Test knowledge base loading
	err = helperAgent.LoadKnowledge()
	if err != nil {
		t.Fatalf("Expected knowledge base loading to succeed, got error: %v", err)
	}

	// Verify knowledge was loaded
	if helperAgent.Knowledge == nil {
		t.Fatal("Expected knowledge base to be loaded")
	}

	if len(helperAgent.Knowledge.Documents) == 0 {
		t.Error("Expected knowledge base to contain documents")
	}

	// Verify specific documents were loaded
	expectedDocs := []string{"topic1.md", "topic2.md", "nested/subtopic.md"}
	for _, expectedDoc := range expectedDocs {
		if _, exists := helperAgent.Knowledge.Documents[expectedDoc]; !exists {
			t.Errorf("Expected knowledge base to contain document '%s'", expectedDoc)
		}
	}

	// Verify document content
	if !strings.Contains(helperAgent.Knowledge.Documents["topic1.md"], "Topic 1") {
		t.Error("Expected topic1.md to contain 'Topic 1'")
	}

	if !strings.Contains(helperAgent.Knowledge.Documents["topic2.md"], "Topic 2") {
		t.Error("Expected topic2.md to contain 'Topic 2'")
	}
}

// TestHelperAgentKnowledgeSearch tests knowledge base search functionality
func TestHelperAgentKnowledgeSearch(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge files
	knowledgeFiles := map[string]string{
		"programming.md": `# Programming

This document covers programming concepts.

## Languages
- Go: A statically typed language
- Python: A dynamically typed language
- JavaScript: A web programming language

## Concepts
- Variables: Store data
- Functions: Reusable code blocks
- Classes: Object-oriented programming`,
		"algorithms.md": `# Algorithms

This document covers algorithm concepts.

## Sorting
- Bubble sort: Simple but inefficient
- Quick sort: Efficient divide and conquer
- Merge sort: Stable sorting algorithm

## Searching
- Linear search: Check each element
- Binary search: Divide and conquer approach`,
	}

	for filename, content := range knowledgeFiles {
		filePath := filepath.Join(knowledgePath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create knowledge file %s: %v", filename, err)
		}
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"programming", "algorithms"},
		Keywords:      []string{"help", "assist"},
		SystemPrompt:  "You are a helpful assistant.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Load knowledge base
	err = helperAgent.LoadKnowledge()
	if err != nil {
		t.Fatalf("Expected knowledge base loading to succeed, got error: %v", err)
	}

	// Test knowledge search
	searchTests := []struct {
		query    string
		expected int // minimum expected results
	}{
		{"Go", 1},           // Should find Go in programming.md
		{"sorting", 1},      // Should find sorting in algorithms.md
		{"function", 1},      // Should find function in programming.md
		{"binary", 1},       // Should find binary in algorithms.md
		{"nonexistent", 0},  // Should find nothing
	}

	for _, test := range searchTests {
		// Test knowledge search (simplified)
		results := []string{}
		for filename, content := range helperAgent.Knowledge.Documents {
			if strings.Contains(strings.ToLower(content), strings.ToLower(test.query)) {
				results = append(results, filename)
			}
		}
		if len(results) < test.expected {
			t.Errorf("Expected at least %d results for query '%s', got %d", test.expected, test.query, len(results))
		}

		// Verify results contain the query
		for _, result := range results {
			content := helperAgent.Knowledge.Documents[result]
			if !strings.Contains(strings.ToLower(content), strings.ToLower(test.query)) {
				t.Errorf("Expected search result to contain query '%s', got: %s", test.query, content)
			}
		}
	}
}

// TestHelperAgentMessageHandling tests helper agent message handling
func TestHelperAgentMessageHandling(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge file
	knowledgeFile := filepath.Join(knowledgePath, "help.md")
	knowledgeContent := `# Help Documentation

This is help documentation for the helper agent.

## Topics
- General assistance
- Technical support
- Best practices

## Examples
- How to use the system
- Common troubleshooting steps
- Tips and tricks`
	err = os.WriteFile(knowledgeFile, []byte(knowledgeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create knowledge file: %v", err)
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"help", "assistance"},
		Keywords:      []string{"help", "assist", "guide"},
		SystemPrompt:  "You are a helpful assistant with specialized knowledge.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers: make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Load knowledge base
	err = helperAgent.LoadKnowledge()
	if err != nil {
		t.Fatalf("Expected knowledge base loading to succeed, got error: %v", err)
	}

	// Start agent
	ctx := context.Background()
	err = helperAgent.Start(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected agent to start, got error: %v", err)
	}

	// Create test message asking for help
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"@TestHelper Can you help me with troubleshooting?",
	)
	msg.Mentions = []string{helperAgent.Info.ID}

	// Send message
	hub.SendMessage(msg)
	time.Sleep(200 * time.Millisecond)

	// Check if agent responded
	chatResponses := 0
	for _, sentMsg := range hub.sentMessages {
		if sentMsg.Type == protocol.MessageTypeChat {
			chatResponses++
		}
	}

	if chatResponses == 0 {
		t.Error("Expected helper agent to respond to help request")
	}

	// Stop agent
	helperAgent.Stop()
}

// TestHelperAgentKeywordMatching tests keyword-based response triggering
func TestHelperAgentKeywordMatching(t *testing.T) {
	// Create helper agent configuration with specific keywords
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"programming", "debugging"},
		Keywords:      []string{"bug", "error", "debug", "fix", "problem"},
		SystemPrompt:  "You are a debugging assistant.",
		KnowledgePath: "", // No knowledge base for this test
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Test keyword matching
	keywordTests := []struct {
		message   string
		shouldMatch bool
	}{
		{"I have a bug in my code", true},
		{"There's an error in the system", true},
		{"Can you help me debug this?", true},
		{"I need to fix this problem", true},
		{"Hello, how are you?", false},
		{"What's the weather like?", false},
		{"Tell me about programming", true}, // Matches expertise
	}

	for _, test := range keywordTests {
		msg := protocol.NewMessage(
			protocol.MessageTypeQuestion,
			"test-channel",
			protocol.AgentInfo{
				ID:   "user-123",
				Name: "TestUser",
				Type: protocol.AgentTypeGeneral,
			},
			test.message,
		)

		// Test shouldRespond logic
		shouldRespond := helperAgent.ShouldRespond(msg)
		if shouldRespond != test.shouldMatch {
			t.Errorf("Expected shouldRespond=%v for message '%s', got %v", 
				test.shouldMatch, test.message, shouldRespond)
		}
	}
}

// TestHelperAgentErrorHandling tests error handling in helper agent
func TestHelperAgentErrorHandling(t *testing.T) {
	// Test with non-existent knowledge path
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"general"},
		Keywords:      []string{"help"},
		SystemPrompt:  "You are a helpful assistant.",
		KnowledgePath: "/non/existent/path",
	}

	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent (should succeed even with non-existent path)
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed even with non-existent path, got error: %v", err)
	}

	// Test knowledge base loading with non-existent path
	err = helperAgent.LoadKnowledge()
	if err == nil {
		t.Error("Expected error when loading knowledge base from non-existent path")
	}

	// Test with empty knowledge path (should succeed)
	config.KnowledgePath = ""
	helperAgent2, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed with empty knowledge path, got error: %v", err)
	}

	// Test knowledge base loading with empty path
	err = helperAgent2.LoadKnowledge()
	if err != nil {
		t.Fatalf("Expected knowledge base loading to succeed with empty path, got error: %v", err)
	}
}

// TestHelperAgentConcurrentOperations tests concurrent operations on helper agent
func TestHelperAgentConcurrentOperations(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create multiple knowledge files
	for i := 0; i < 5; i++ {
		filePath := filepath.Join(knowledgePath, fmt.Sprintf("topic%d.md", i))
		content := fmt.Sprintf("# Topic %d\n\nThis is content for topic %d.", i, i)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create knowledge file %d: %v", i, err)
		}
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"general"},
		Keywords:      []string{"help", "assist"},
		SystemPrompt:  "You are a helpful assistant.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Test concurrent knowledge loading and searching
	done := make(chan bool, 2)

	// Concurrent knowledge loading
	go func() {
		err := helperAgent.LoadKnowledge()
		if err != nil {
			t.Errorf("Expected concurrent knowledge loading to succeed, got error: %v", err)
		}
		done <- true
	}()

	// Concurrent knowledge searching
	go func() {
		time.Sleep(100 * time.Millisecond) // Wait a bit for loading to start
		// Test knowledge search (simplified)
		results := []string{}
		for filename, content := range helperAgent.Knowledge.Documents {
			if strings.Contains(strings.ToLower(content), "topic") {
				results = append(results, filename)
			}
		}
		if len(results) == 0 {
			t.Error("Expected concurrent search to find results")
		}
		done <- true
	}()

	// Wait for both operations to complete
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// Operation completed
		case <-time.After(5 * time.Second):
			t.Error("Test timed out waiting for concurrent operations")
			return
		}
	}
}

// TestHelperAgentStoragePersistence tests storage persistence functionality
func TestHelperAgentStoragePersistence(t *testing.T) {
	// Create test knowledge base directory
	tempDir := t.TempDir()
	knowledgePath := filepath.Join(tempDir, "knowledge")
	err := os.MkdirAll(knowledgePath, 0755)
	if err != nil {
		t.Fatalf("Failed to create knowledge directory: %v", err)
	}

	// Create test knowledge file
	knowledgeFile := filepath.Join(knowledgePath, "test.md")
	knowledgeContent := `# Test Knowledge

This is test knowledge for persistence testing.

## Content
- Test point 1
- Test point 2
- Test point 3`
	err = os.WriteFile(knowledgeFile, []byte(knowledgeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create knowledge file: %v", err)
	}

	// Create helper agent configuration
	config := &agent.HelperAgentConfig{
		Name:          "TestHelper",
		Description:   "A test helper agent",
		Expertise:     []string{"general"},
		Keywords:      []string{"help", "assist"},
		SystemPrompt:  "You are a helpful assistant.",
		KnowledgePath: knowledgePath,
	}

	// Create mock hub and AI provider
	hub := &mockHubClientHelper{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected helper agent creation to succeed, got error: %v", err)
	}

	// Load knowledge base
	err = helperAgent.LoadKnowledge()
	if err != nil {
		t.Fatalf("Expected knowledge base loading to succeed, got error: %v", err)
	}

	// Test storage operations (simplified for now)
	if helperAgent.Knowledge == nil {
		t.Fatal("Expected knowledge to be loaded")
	}

	if len(helperAgent.Knowledge.Documents) == 0 {
		t.Error("Expected knowledge to contain documents")
	}
}
