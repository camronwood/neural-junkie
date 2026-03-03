package test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestCommandHandlerCreation tests command handler creation
func TestCommandHandlerCreation(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}
	if handler == nil {
		t.Fatal("Expected command handler to be created")
	}
}

// TestHelpCommand tests the help command
func TestHelpCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test message with help command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/help",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected help command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected help command to return a response")
	}

	if !strings.Contains(response.Content, "Available Commands") {
		t.Error("Expected help response to contain 'Available Commands'")
	}
}

// TestListAgentsCommand tests the list agents command
func TestListAgentsCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create the test channel
	h.CreateChannel("test-channel", "Test channel", "test-project")

	// Create test message with list agents command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/list-agents",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected list agents command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected list agents command to return a response")
	}

	if !strings.Contains(response.Content, "No agents") && !strings.Contains(response.Content, "Agents") {
		t.Error("Expected list agents response to contain agent information")
	}
}

// TestCreateRepoAgentCommand tests the create repo agent command
func TestCreateRepoAgentCommand(t *testing.T) {
	useIsolatedRepoStorage(t)

	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create the test channel
	h.CreateChannel("test-channel", "Test channel", "test-project")

	// Create isolated test directory
	testRepoPath := t.TempDir()

	// Create test message with create repo agent command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		fmt.Sprintf("/create-repo-agent %s TestRepoAgent", testRepoPath),
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected create repo agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected create repo agent command to return a response")
	}

	// The command should either succeed or fail with a specific error
	if !strings.Contains(response.Content, "created") &&
		!strings.Contains(response.Content, "Creating") &&
		!strings.Contains(response.Content, "error") &&
		!strings.Contains(response.Content, "not found") {
		t.Error("Expected create repo agent response to contain status information")
	}

	// Wait for indexing to complete to avoid background writes during test cleanup.
	deadline := time.Now().Add(2 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		agents := h.ListAgents()
		for _, a := range agents {
			if strings.EqualFold(a.Name, "TestRepoAgent") &&
				a.Type == protocol.AgentTypeRepo &&
				a.IndexingStatus == string(protocol.IndexingStatusReady) {
				ready = true
				break
			}
		}
		if ready {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		t.Fatal("Expected repo agent indexing to complete")
	}
}

func TestSwitchProviderUpdatesRuntimeAgent(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}
	h.CreateChannel("test-channel", "Test channel", "test-project")

	runtimeAgent := agent.NewAgentWithProvider(
		protocol.AgentTypeBackend,
		"SwitchTarget",
		[]string{"backend"},
		ai.NewMockProvider(),
		h,
		"mock",
		"mock-model",
	)
	handler.RegisterRuntimeAgent(runtimeAgent)
	if err := h.RegisterAgent(&runtimeAgent.Info); err != nil {
		t.Fatalf("Expected runtime agent registration to succeed, got error: %v", err)
	}

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/switch-provider SwitchTarget ollama llama3.2",
	)

	response, err := handler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Expected switch provider command to succeed, got error: %v", err)
	}
	if response == nil {
		t.Fatal("Expected switch provider command to return a response")
	}
	if !strings.Contains(response.Content, "switched") {
		t.Fatalf("Expected success response, got: %s", response.Content)
	}

	if runtimeAgent.GetAIProvider().GetModel() != "llama3.2" {
		t.Fatalf("Expected runtime provider model to be updated to llama3.2, got %s", runtimeAgent.GetAIProvider().GetModel())
	}
	if runtimeAgent.Info.AIProvider != "ollama" {
		t.Fatalf("Expected runtime provider to be ollama, got %s", runtimeAgent.Info.AIProvider)
	}
	if runtimeAgent.Info.AIModel != "llama3.2" {
		t.Fatalf("Expected runtime AI model to be llama3.2, got %s", runtimeAgent.Info.AIModel)
	}
}

// TestDeleteAgentCommand tests the delete agent command
func TestDeleteAgentCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test message with delete agent command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/delete-agent NonExistentAgent",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected delete agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected delete agent command to return a response")
	}

	// Should indicate agent not found
	if !strings.Contains(response.Content, "not found") &&
		!strings.Contains(response.Content, "No agent") {
		t.Error("Expected delete agent response to indicate agent not found")
	}
}

// TestPauseUnpauseAgentCommand tests the pause and unpause agent commands
func TestPauseUnpauseAgentCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Test pause command
	pauseMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/pause-agent NonExistentAgent",
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, pauseMsg)
	if err != nil {
		t.Fatalf("Expected pause agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected pause agent command to return a response")
	}

	// Test unpause command
	unpauseMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/unpause-agent NonExistentAgent",
	)

	response, err = handler.ProcessCommand(ctx, unpauseMsg)
	if err != nil {
		t.Fatalf("Expected unpause agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected unpause agent command to return a response")
	}
}

// TestReindexAgentCommand tests the reindex agent command
func TestReindexAgentCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test message with reindex agent command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/reindex-agent NonExistentAgent",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected reindex agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected reindex agent command to return a response")
	}

	// Should indicate agent not found
	if !strings.Contains(response.Content, "not found") &&
		!strings.Contains(response.Content, "No agent") {
		t.Error("Expected reindex agent response to indicate agent not found")
	}
}

// TestWatchCommands tests the enable/disable watch commands
func TestWatchCommands(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Test enable watch command
	enableMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/enable-watch NonExistentAgent",
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, enableMsg)
	if err != nil {
		t.Fatalf("Expected enable watch command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected enable watch command to return a response")
	}

	// Test disable watch command
	disableMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/disable-watch NonExistentAgent",
	)

	response, err = handler.ProcessCommand(ctx, disableMsg)
	if err != nil {
		t.Fatalf("Expected disable watch command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected disable watch command to return a response")
	}
}

// TestCreateHelperCommand tests the create helper command
func TestCreateHelperCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test message with create helper command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/create-helper TestHelper Test helper agent",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected create helper command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected create helper command to return a response")
	}

	// The command should either succeed or provide information
	if !strings.Contains(response.Content, "created") &&
		!strings.Contains(response.Content, "helper") {
		t.Error("Expected create helper response to contain helper information")
	}
}

// TestListHelperTemplatesCommand tests the list helper templates command
func TestListHelperTemplatesCommand(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Create test message with list helper templates command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/list-helper-templates",
	)

	// Process command
	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected list helper templates command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected list helper templates command to return a response")
	}

	// Should contain template information
	if !strings.Contains(response.Content, "templates") &&
		!strings.Contains(response.Content, "Available") {
		t.Error("Expected list helper templates response to contain template information")
	}
}

// TestConfluenceCommands tests confluence-related commands
func TestConfluenceCommands(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Test create confluence agent command
	createMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/create-confluence-agent TestConfluenceAgent",
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, createMsg)
	if err != nil {
		t.Fatalf("Expected create confluence agent command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected create confluence agent command to return a response")
	}

	// Test list confluence agents command
	listMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/list-confluence-agents",
	)

	response, err = handler.ProcessCommand(ctx, listMsg)
	if err != nil {
		t.Fatalf("Expected list confluence agents command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected list confluence agents command to return a response")
	}
}

// TestWorkspaceCommands tests workspace-related commands
func TestWorkspaceCommands(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}
	workspaceDir := t.TempDir()

	// Test add workspace command
	addMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		fmt.Sprintf("/add-workspace %s", workspaceDir),
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, addMsg)
	if err != nil {
		t.Fatalf("Expected add workspace command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected add workspace command to return a response")
	}

	// Test list workspaces command
	listMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/list-workspaces",
	)

	response, err = handler.ProcessCommand(ctx, listMsg)
	if err != nil {
		t.Fatalf("Expected list workspaces command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected list workspaces command to return a response")
	}
}

// TestAssistantCommandsIntegration tests assistant-related commands
func TestAssistantCommandsIntegration(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Test reminder command
	reminderMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/remind in 5 minutes Test reminder",
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, reminderMsg)
	if err != nil {
		t.Fatalf("Expected reminder command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected reminder command to return a response")
	}

	// Test task command
	taskMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/task-add Test task",
	)

	response, err = handler.ProcessCommand(ctx, taskMsg)
	if err != nil {
		t.Fatalf("Expected task command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected task command to return a response")
	}

	// Test note command
	noteMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/note-save Test note content",
	)

	response, err = handler.ProcessCommand(ctx, noteMsg)
	if err != nil {
		t.Fatalf("Expected note command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected note command to return a response")
	}

	// Test meeting command
	meetingMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/meeting-add tomorrow 2pm Test meeting",
	)

	response, err = handler.ProcessCommand(ctx, meetingMsg)
	if err != nil {
		t.Fatalf("Expected meeting command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected meeting command to return a response")
	}

	// Test assistant help command
	helpMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/help-assistant",
	)

	response, err = handler.ProcessCommand(ctx, helpMsg)
	if err != nil {
		t.Fatalf("Expected assistant help command to succeed, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected assistant help command to return a response")
	}
}

// TestInvalidCommands tests handling of invalid commands
func TestInvalidCommands(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	// Test invalid command
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"/invalid-command",
	)

	ctx := context.Background()
	response, err := handler.ProcessCommand(ctx, msg)
	if err != nil {
		t.Fatalf("Expected invalid command to be handled gracefully, got error: %v", err)
	}

	if response == nil {
		t.Fatal("Expected invalid command to return a response")
	}

	// Should indicate unknown command
	if !strings.Contains(response.Content, "Unknown command") &&
		!strings.Contains(response.Content, "not recognized") {
		t.Error("Expected invalid command response to indicate unknown command")
	}
}

// TestCommandParsing tests command parsing with various formats
func TestCommandParsing(t *testing.T) {
	useIsolatedRepoStorage(t)

	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}
	workspaceDir := t.TempDir()
	nonExistentRepoPath := filepath.Join(workspaceDir, "missing-repo")

	testCases := []struct {
		command    string
		shouldWork bool
	}{
		{"/help", true},
		{"/list-agents", true},
		{fmt.Sprintf("/create-repo-agent %s TestAgent", nonExistentRepoPath), true},
		{"/delete-agent TestAgent", true},
		{"/pause-agent TestAgent", true},
		{"/unpause-agent TestAgent", true},
		{"/reindex-agent TestAgent", true},
		{"/enable-watch TestAgent", true},
		{"/disable-watch TestAgent", true},
		{"/create-helper TestHelper Test description", true},
		{"/list-helper-templates", true},
		{"/create-confluence-agent TestAgent", true},
		{"/list-confluence-agents", true},
		{fmt.Sprintf("/add-workspace %s", workspaceDir), true},
		{"/list-workspaces", true},
		{"/remind in 5 minutes Test", true},
		{"/task-add Test task", true},
		{"/note-save Test note", true},
		{"/meeting-add tomorrow 2pm Test", true},
		{"/help-assistant", true},
		{"/invalid-command", false},
		{"not-a-command", false},
		{"", false},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		msg := protocol.NewMessage(
			protocol.MessageTypeChat,
			"test-channel",
			protocol.AgentInfo{
				ID:   "user-123",
				Name: "TestUser",
				Type: protocol.AgentTypeGeneral,
			},
			tc.command,
		)

		response, err := handler.ProcessCommand(ctx, msg)
		if tc.shouldWork {
			if err != nil {
				t.Errorf("Expected command '%s' to work, got error: %v", tc.command, err)
			}
			if response == nil {
				t.Errorf("Expected command '%s' to return a response", tc.command)
			}
		} else {
			// For invalid commands, we expect either an error or a response indicating unknown command
			if err == nil && response != nil {
				if !strings.Contains(response.Content, "Unknown command") &&
					!strings.Contains(response.Content, "not recognized") {
					t.Errorf("Expected command '%s' to indicate unknown command", tc.command)
				}
			}
		}
	}
}

func TestCommandDefinitionParityAndAssistantCommandsPresent(t *testing.T) {
	h := hub.NewHub()
	handler, err := hub.NewCommandHandler(h)
	if err != nil {
		t.Fatalf("Expected command handler creation to succeed, got error: %v", err)
	}

	defs := handler.GetCommandDefinitions()
	defSet := map[string]bool{}
	for _, d := range defs {
		defSet[strings.ToLower(d.Name)] = true
	}

	required := []string{
		"/remind", "/remind-recurring", "/task-add", "/task-list", "/task-done",
		"/note-save", "/note-search", "/meeting-add", "/summarize",
	}
	for _, cmd := range required {
		if !defSet[cmd] {
			t.Fatalf("Expected command definition for %s", cmd)
		}
	}
}
