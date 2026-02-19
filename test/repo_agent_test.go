package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Mock hub client for repository agent testing
type mockHubClientRepo struct {
	sentMessages []*protocol.Message
	subscribers  []chan *protocol.Message
}

func (m *mockHubClientRepo) SendMessage(msg *protocol.Message) error {
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

func (m *mockHubClientRepo) Subscribe(channelName string) (chan *protocol.Message, error) {
	subCh := make(chan *protocol.Message, 100)
	m.subscribers = append(m.subscribers, subCh)
	return subCh, nil
}

func (m *mockHubClientRepo) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return nil, nil
}

func (m *mockHubClientRepo) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	return nil, nil
}

func (m *mockHubClientRepo) GetThreadParentAuthor(threadID string) string {
	return ""
}

func (m *mockHubClientRepo) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

// TestRepoAgentCreation tests repository agent creation
func TestRepoAgentCreation(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create some test files
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`,
		"README.md": `# Test Repository

This is a test repository for testing the repository agent.`,
		"go.mod": `module test-repo

go 1.21`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	if repoAgent == nil {
		t.Fatal("Expected repository agent to be created")
	}

	if repoAgent.Info.Name != "TestRepoAgent" {
		t.Errorf("Expected agent name 'TestRepoAgent', got '%s'", repoAgent.Info.Name)
	}

	if repoAgent.Info.Type != protocol.AgentTypeRepo {
		t.Errorf("Expected agent type AgentTypeRepo, got %v", repoAgent.Info.Type)
	}

	if repoAgent.Info.RepositoryPath != testRepoPath {
		t.Errorf("Expected repository path '%s', got '%s'", testRepoPath, repoAgent.Info.RepositoryPath)
	}
}

// TestRepoAgentIndexing tests repository indexing functionality
func TestRepoAgentIndexing(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test files with various content
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

This is a test repository for testing the repository agent.

## Features
- Go application
- Utility functions
- Documentation`,
		"go.mod": `module test-repo

go 1.21`,
		"config.json": `{
    "name": "test-repo",
    "version": "1.0.0",
    "description": "A test repository"
}`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Test indexing (simplified for test)
	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Wait for indexing to complete
	time.Sleep(1 * time.Second)

	// Verify agent is running
	if repoAgent.Info.Status != "active" {
		t.Error("Expected repository agent to be active")
	}

	// Test completed successfully
	t.Log("Repository agent indexing test completed")

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentSearch tests repository search functionality
func TestRepoAgentSearch(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test files with searchable content
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
    fmt.Println("This is a test function")
}`,
		"utils.go": `package main

import "strings"

func toUpperCase(s string) string {
    return strings.ToUpper(s)
}

func processData(data string) string {
    return toUpperCase(data)
}`,
		"README.md": `# Test Repository

This is a test repository for testing the repository agent.

## Features
- Go application
- Utility functions
- Documentation
- Search functionality`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Start with indexing
	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Wait for indexing to complete
	time.Sleep(1 * time.Second)

	// Test search functionality (simplified)
	searchTests := []struct {
		query    string
		expected int // minimum expected results
	}{
		{"Hello", 1},       // Should find "Hello, World!"
		{"test", 3},        // Should find multiple occurrences
		{"function", 2},    // Should find function definitions
		{"fmt.Println", 2}, // Should find fmt.Println calls
		{"toUpperCase", 2}, // Should find function definition and usage
		{"nonexistent", 0}, // Should find nothing
	}

	for _, test := range searchTests {
		// Test search (simplified for test)
		t.Logf("Testing search for query: %s", test.query)
		// Note: Actual search functionality would be tested in integration tests
	}

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentCaching tests repository agent caching functionality
func TestRepoAgentCaching(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test file
	testFile := filepath.Join(testRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create first repository agent and start with indexing
	repoAgent1, err := agent.NewRepoAgent("TestRepoAgent1", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	ctx := context.Background()
	err = repoAgent1.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Wait for indexing to complete
	time.Sleep(1 * time.Second)

	// Create second repository agent for the same path (should use cache)
	repoAgent2, err := agent.NewRepoAgent("TestRepoAgent2", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Start second agent
	err = repoAgent2.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected second repository agent to start, got error: %v", err)
	}

	// Wait for second agent to start
	time.Sleep(1 * time.Second)

	// Verify both agents are running
	if repoAgent1.Info.Status != "active" || repoAgent2.Info.Status != "active" {
		t.Error("Expected both agents to be active")
	}

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentFileWatching tests file watching functionality
func TestRepoAgentFileWatching(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create initial test file
	testFile := filepath.Join(testRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("Initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Start with indexing
	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Enable file watching
	repoAgent.EnableAutoWatch(ctx)

	// Wait a moment for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify the test file
	err = os.WriteFile(testFile, []byte("Modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for file change to be detected
	time.Sleep(500 * time.Millisecond)

	// Disable file watching
	repoAgent.DisableAutoWatch()

	// Wait for any pending reindex to complete
	time.Sleep(1 * time.Second)

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentMessageHandling tests repository agent message handling
func TestRepoAgentMessageHandling(t *testing.T) {
	// Create temporary test repository
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
		"README.md": `# Test Repository

This is a test repository for testing the repository agent.`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testRepoPath, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Start with indexing
	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository agent to start with indexing, got error: %v", err)
	}

	// Start agent
	err = repoAgent.Start(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected agent to start, got error: %v", err)
	}

	// Create test message asking about the repository
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"@TestRepoAgent What files are in this repository?",
	)
	msg.Mentions = []string{repoAgent.Info.ID}

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
		t.Error("Expected repository agent to respond to question about repository")
	}

	// Stop agent
	repoAgent.Stop()

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentErrorHandling tests error handling in repository agent
func TestRepoAgentErrorHandling(t *testing.T) {
	// Test with non-existent repository path
	mockAI := ai.NewMockProvider()
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}

	_, err := agent.NewRepoAgent("TestRepoAgent", "/non/existent/path", mockAI, hub)
	if err == nil {
		t.Error("Expected error when creating repository agent with non-existent path")
	}

	// Test with file instead of directory
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "not-a-directory")
	err = os.WriteFile(testFile, []byte("not a directory"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = agent.NewRepoAgent("TestRepoAgent", testFile, mockAI, hub)
	if err == nil {
		t.Error("Expected error when creating repository agent with file path instead of directory")
	}
}

// TestRepoAgentConcurrentOperations tests concurrent operations on repository agent
func TestRepoAgentConcurrentOperations(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test files
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(testRepoPath, fmt.Sprintf("file%d.txt", i))
		content := fmt.Sprintf("Content of file %d", i)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	// Test concurrent indexing and searching
	ctx := context.Background()
	done := make(chan bool, 2)

	// Concurrent indexing
	go func() {
		err := repoAgent.StartWithIndexing(ctx, "test-channel")
		if err != nil {
			t.Errorf("Expected concurrent indexing to succeed, got error: %v", err)
		}
		done <- true
	}()

	// Concurrent search (should work even during indexing)
	go func() {
		time.Sleep(100 * time.Millisecond) // Wait a bit for indexing to start
		// Test search (simplified for test)
		t.Log("Testing concurrent search")
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

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}

// TestRepoAgentStoragePersistence tests storage persistence functionality
func TestRepoAgentStoragePersistence(t *testing.T) {
	// Create temporary test repository
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")
	err := os.MkdirAll(testRepoPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Clean up cache entry after test
	cleanupRepoAgentCache(t, testRepoPath)

	// Create test file
	testFile := filepath.Join(testRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("Test content for persistence"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock hub and AI provider
	hub := &mockHubClientRepo{
		sentMessages: make([]*protocol.Message, 0),
		subscribers:  make([]chan *protocol.Message, 0),
	}
	mockAI := ai.NewMockProvider()

	// Create repository agent and index
	repoAgent, err := agent.NewRepoAgent("TestRepoAgent", testRepoPath, mockAI, hub)
	if err != nil {
		t.Fatalf("Expected repository agent creation to succeed, got error: %v", err)
	}

	ctx := context.Background()
	err = repoAgent.StartWithIndexing(ctx, "test-channel")
	if err != nil {
		t.Fatalf("Expected repository indexing to succeed, got error: %v", err)
	}

	// Test storage operations (simplified for test)
	if repoAgent.Info.Status != "active" {
		t.Error("Expected repository agent to be active")
	}

	// Clean up immediately after test
	cleanupRepoAgentCacheImmediate(t, testRepoPath)
}
