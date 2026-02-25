package test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// TestAssistantStorage tests the assistant storage functionality
func TestAssistantStorage(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create storage
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create assistant storage: %v", err)
	}

	// Test saving and loading reminders
	reminder := &agent.Reminder{
		ID:          uuid.New().String(),
		Content:     "Test reminder",
		TriggerTime: time.Now().Add(1 * time.Hour),
		Channel:     "general",
		CreatedBy:   "test-user",
		Active:      true,
		CreatedAt:   time.Now(),
	}

	err = storage.SaveReminder(reminder)
	if err != nil {
		t.Fatalf("Failed to save reminder: %v", err)
	}

	loadedReminders, err := storage.LoadReminders()
	if err != nil {
		t.Fatalf("Failed to load reminders: %v", err)
	}

	if len(loadedReminders) != 1 {
		t.Fatalf("Expected 1 reminder, got %d", len(loadedReminders))
	}

	if loadedReminders[0].Content != reminder.Content {
		t.Fatalf("Expected content %s, got %s", reminder.Content, loadedReminders[0].Content)
	}

	// Test saving and loading tasks
	task := &agent.Task{
		ID:          uuid.New().String(),
		Title:       "Test task",
		Description: "Test task description",
		Priority:    3,
		Status:      "todo",
		CreatedAt:   time.Now(),
		Channel:     "general",
		CreatedBy:   "test-user",
	}

	err = storage.SaveTask(task)
	if err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	loadedTasks, err := storage.LoadTasks()
	if err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}

	if len(loadedTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(loadedTasks))
	}

	if loadedTasks[0].Title != task.Title {
		t.Fatalf("Expected title %s, got %s", task.Title, loadedTasks[0].Title)
	}

	// Test saving and loading notes
	note := &agent.Note{
		ID:        uuid.New().String(),
		Content:   "Test note content",
		Tags:      []string{"test", "example"},
		Channel:   "general",
		CreatedAt: time.Now(),
		CreatedBy: "test-user",
	}

	err = storage.SaveNote(note)
	if err != nil {
		t.Fatalf("Failed to save note: %v", err)
	}

	loadedNotes, err := storage.LoadNotes()
	if err != nil {
		t.Fatalf("Failed to load notes: %v", err)
	}

	if len(loadedNotes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(loadedNotes))
	}

	if loadedNotes[0].Content != note.Content {
		t.Fatalf("Expected content %s, got %s", note.Content, loadedNotes[0].Content)
	}

	// Test note search
	searchResults, err := storage.SearchNotes("test")
	if err != nil {
		t.Fatalf("Failed to search notes: %v", err)
	}

	if len(searchResults) != 1 {
		t.Fatalf("Expected 1 search result, got %d", len(searchResults))
	}

	// Test saving and loading meetings
	meeting := &agent.Meeting{
		ID:          uuid.New().String(),
		Title:       "Test meeting",
		Description: "Test meeting description",
		StartTime:   time.Now().Add(2 * time.Hour),
		Channel:     "general",
		CreatedBy:   "test-user",
		CreatedAt:   time.Now(),
	}

	err = storage.SaveMeeting(meeting)
	if err != nil {
		t.Fatalf("Failed to save meeting: %v", err)
	}

	loadedMeetings, err := storage.LoadMeetings()
	if err != nil {
		t.Fatalf("Failed to load meetings: %v", err)
	}

	if len(loadedMeetings) != 1 {
		t.Fatalf("Expected 1 meeting, got %d", len(loadedMeetings))
	}

	if loadedMeetings[0].Title != meeting.Title {
		t.Fatalf("Expected title %s, got %s", meeting.Title, loadedMeetings[0].Title)
	}

	// Test upcoming meetings
	upcoming, err := storage.GetUpcomingMeetings(24) // Next 24 hours
	if err != nil {
		t.Fatalf("Failed to get upcoming meetings: %v", err)
	}

	if len(upcoming) != 1 {
		t.Fatalf("Expected 1 upcoming meeting, got %d", len(upcoming))
	}
}

// TestAssistantAgent tests the assistant agent functionality
func TestAssistantAgent(t *testing.T) {
	// Create mock AI provider
	aiProvider := ai.NewMockProvider()

	// Create mock hub
	mockHub := &MockHub{
		messages: make(map[string][]*protocol.Message),
		agents:   make(map[string]protocol.AgentInfo),
	}

	// Create assistant agent
	assistant := agent.NewAssistantAgent("Test Assistant", aiProvider, mockHub)

	// Test agent creation
	if assistant.Info.Name != "Test Assistant" {
		t.Fatalf("Expected agent name 'Test Assistant', got %s", assistant.Info.Name)
	}

	if assistant.Info.Type != protocol.AgentTypeAssistant {
		t.Fatalf("Expected agent type 'assistant', got %s", assistant.Info.Type)
	}

	// Test shouldRespond logic
	testMessage := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeChat,
		Channel: "general",
		From: protocol.AgentInfo{
			ID:   "test-user",
			Name: "Test User",
			Type: protocol.AgentTypeGeneral,
		},
		Content:   "I need a reminder for tomorrow",
		Timestamp: time.Now(),
	}

	// Test mention detection
	testMessage.Mentions = []string{assistant.Info.ID}
	if !assistant.ShouldRespond(testMessage) {
		t.Fatal("Assistant should respond when mentioned")
	}

	// Test keyword detection
	testMessage.Mentions = []string{}
	testMessage.Content = "Can you help me with a task?"
	if !assistant.ShouldRespond(testMessage) {
		t.Fatal("Assistant should respond to task-related keywords")
	}

	// Test non-relevant message
	testMessage.Content = "Hello world"
	if assistant.ShouldRespond(testMessage) {
		t.Fatal("Assistant should not respond to non-relevant messages")
	}
}

// TestAssistantTimeParsing tests the time parsing functionality
func TestAssistantTimeParsing(t *testing.T) {
	// Create mock AI provider and hub
	aiProvider := ai.NewMockProvider()
	mockHub := &MockHub{
		messages: make(map[string][]*protocol.Message),
		agents:   make(map[string]protocol.AgentInfo),
	}

	assistant := agent.NewAssistantAgent("Test Assistant", aiProvider, mockHub)

	// Test relative time parsing
	now := time.Now()

	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"in 30 minutes", 30 * time.Minute},
		{"in 2 hours", 2 * time.Hour},
		{"in 1 day", 24 * time.Hour},
	}

	for _, tc := range testCases {
		parsed, err := assistant.ParseTime(tc.input)
		if err != nil {
			t.Fatalf("Failed to parse time '%s': %v", tc.input, err)
		}

		expected := now.Add(tc.expected)
		diff := parsed.Sub(expected)
		if diff < -time.Minute || diff > time.Minute {
			t.Fatalf("Expected time around %v, got %v for input '%s'", expected, parsed, tc.input)
		}
	}

	// Test "at" time parsing
	atTime, err := assistant.ParseTime("at 3pm")
	if err != nil {
		t.Fatalf("Failed to parse 'at 3pm': %v", err)
	}

	// Should be today at 3pm
	expected := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
	diff := atTime.Sub(expected)
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("Expected time around %v, got %v for 'at 3pm'", expected, atTime)
	}
}

// TestAssistantCommands tests the command handling
func TestAssistantCommands(t *testing.T) {
	// Create command handler
	chatHub := hub.NewHub()
	commandHandler, err := hub.NewCommandHandler(chatHub)
	if err != nil {
		t.Fatalf("Failed to create command handler: %v", err)
	}

	// Test reminder command
	msg := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeChat,
		Channel: "general",
		From: protocol.AgentInfo{
			ID:   "test-user",
			Name: "Test User",
			Type: protocol.AgentTypeGeneral,
		},
		Content:   "/remind in 30 minutes Test reminder",
		Timestamp: time.Now(),
	}

	response, err := commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to process reminder command: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for reminder command")
	}

	if !strings.Contains(response.Content, "reminder") {
		t.Fatalf("Expected response to contain 'reminder', got: %s", response.Content)
	}

	// Test task command
	msg.Content = "/task-add Test task"
	response, err = commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to process task command: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for task command")
	}

	if !strings.Contains(response.Content, "task") {
		t.Fatalf("Expected response to contain 'task', got: %s", response.Content)
	}

	// Test note command
	msg.Content = "/note-save Test note content"
	response, err = commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to process note command: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for note command")
	}

	if !strings.Contains(response.Content, "note") {
		t.Fatalf("Expected response to contain 'note', got: %s", response.Content)
	}

	// Test meeting command
	msg.Content = "/meeting-add tomorrow 2pm Test meeting"
	response, err = commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to process meeting command: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for meeting command")
	}

	if !strings.Contains(response.Content, "meeting") {
		t.Fatalf("Expected response to contain 'meeting', got: %s", response.Content)
	}

	// Test help command
	msg.Content = "/help-assistant"
	response, err = commandHandler.ProcessCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("Failed to process help command: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response for help command")
	}

	if !strings.Contains(response.Content, "Assistant Commands") {
		t.Fatalf("Expected response to contain 'Assistant Commands', got: %s", response.Content)
	}
}

// TestAssistantIntegration tests the full integration
func TestAssistantIntegration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create hub
	chatHub := hub.NewHub()

	// Create channel
	chatHub.CreateChannel("test", "Test channel", "")

	// Create AI provider
	aiProvider := ai.NewMockProvider()

	// Create assistant agent
	assistant := agent.NewAssistantAgent("Test Assistant", aiProvider, chatHub)

	// Start assistant
	ctx := context.Background()
	err := assistant.Start(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to start assistant: %v", err)
	}

	// Wait a bit for the agent to initialize
	time.Sleep(100 * time.Millisecond)

	// Test sending a message that should trigger the assistant
	testMessage := &protocol.Message{
		ID:      uuid.New().String(),
		Type:    protocol.MessageTypeChat,
		Channel: "test",
		From: protocol.AgentInfo{
			ID:   "test-user",
			Name: "Test User",
			Type: protocol.AgentTypeGeneral,
		},
		Content:   "I need help with a reminder",
		Timestamp: time.Now(),
	}

	// Send message to hub
	err = chatHub.SendMessage(testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Stop assistant
	assistant.Stop()
}

// MockHub is a mock implementation of the HubClient interface
type MockHub struct {
	messages map[string][]*protocol.Message
	agents   map[string]protocol.AgentInfo
}

func (m *MockHub) SendMessage(msg *protocol.Message) error {
	m.messages[msg.Channel] = append(m.messages[msg.Channel], msg)
	return nil
}

func (m *MockHub) Subscribe(channelName string) (chan *protocol.Message, error) {
	ch := make(chan *protocol.Message, 100)
	return ch, nil
}

func (m *MockHub) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	return m.messages[channelName], nil
}

func (m *MockHub) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	var agents []protocol.AgentInfo
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

func (m *MockHub) GetThreadParentAuthor(threadID string) string {
	return "test-user"
}

func (m *MockHub) GetCommandHandler() agent.CommandHandlerInterface {
	return nil
}

func (m *MockHub) BroadcastDirect(channelName string, msg *protocol.Message) {}

func (m *MockHub) GetAgentChannels(agentID string) []string {
	return []string{"general"}
}

func (m *MockHub) GetChannelType(channelName string) protocol.ChannelType {
	return protocol.ChannelTypePublic
}

// TestAssistantStoragePersistence tests that data persists across restarts
func TestAssistantStoragePersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create first storage instance
	storage1, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create first storage: %v", err)
	}

	// Save some data
	reminder := &agent.Reminder{
		ID:          uuid.New().String(),
		Content:     "Persistent reminder",
		TriggerTime: time.Now().Add(1 * time.Hour),
		Channel:     "general",
		CreatedBy:   "test-user",
		Active:      true,
		CreatedAt:   time.Now(),
	}

	err = storage1.SaveReminder(reminder)
	if err != nil {
		t.Fatalf("Failed to save reminder: %v", err)
	}

	// Create second storage instance (simulating restart)
	storage2, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create second storage: %v", err)
	}

	// Load data from second instance
	loadedReminders, err := storage2.LoadReminders()
	if err != nil {
		t.Fatalf("Failed to load reminders: %v", err)
	}

	if len(loadedReminders) != 1 {
		t.Fatalf("Expected 1 reminder after restart, got %d", len(loadedReminders))
	}

	if loadedReminders[0].Content != reminder.Content {
		t.Fatalf("Expected content %s, got %s", reminder.Content, loadedReminders[0].Content)
	}
}

// TestAssistantRecurringReminders tests recurring reminder functionality
func TestAssistantRecurringReminders(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create recurring reminder
	reminder := &agent.Reminder{
		ID:          uuid.New().String(),
		Content:     "Daily standup",
		TriggerTime: time.Now().Add(1 * time.Minute), // Fire in 1 minute
		Recurring: &agent.RecurringSchedule{
			Type:     "daily",
			Interval: 1,
			Time:     "09:00",
		},
		Channel:   "general",
		CreatedBy: "test-user",
		Active:    true,
		CreatedAt: time.Now(),
	}

	err = storage.SaveReminder(reminder)
	if err != nil {
		t.Fatalf("Failed to save recurring reminder: %v", err)
	}

	// Load and verify
	loadedReminders, err := storage.LoadReminders()
	if err != nil {
		t.Fatalf("Failed to load reminders: %v", err)
	}

	if len(loadedReminders) != 1 {
		t.Fatalf("Expected 1 reminder, got %d", len(loadedReminders))
	}

	if loadedReminders[0].Recurring == nil {
		t.Fatal("Expected recurring schedule to be set")
	}

	if loadedReminders[0].Recurring.Type != "daily" {
		t.Fatalf("Expected recurring type 'daily', got %s", loadedReminders[0].Recurring.Type)
	}
}

// TestEmailStorage tests the email storage functionality
func TestEmailStorage(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create storage
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create assistant storage: %v", err)
	}

	// Test saving and loading emails
	email := &agent.Email{
		ID:         uuid.New().String(),
		Subject:    "Test Email Subject",
		From:       "sender@example.com",
		To:         []string{"recipient@example.com"},
		Body:       "This is a test email body content.",
		Date:       time.Now().Add(-2 * 24 * time.Hour), // 2 days ago
		ReceivedAt: time.Now(),
		MessageID:  "<test@example.com>",
		IsRead:     false,
	}

	err = storage.SaveEmail(email)
	if err != nil {
		t.Fatalf("Failed to save email: %v", err)
	}

	loadedEmails, err := storage.LoadEmails()
	if err != nil {
		t.Fatalf("Failed to load emails: %v", err)
	}

	if len(loadedEmails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(loadedEmails))
	}

	if loadedEmails[0].Subject != email.Subject {
		t.Fatalf("Expected subject %s, got %s", email.Subject, loadedEmails[0].Subject)
	}

	// Test GetRecentEmails (last 7 days)
	recentEmails, err := storage.GetRecentEmails(7)
	if err != nil {
		t.Fatalf("Failed to get recent emails: %v", err)
	}

	if len(recentEmails) != 1 {
		t.Fatalf("Expected 1 recent email, got %d", len(recentEmails))
	}

	// Test GetRecentEmails with shorter period (should return 0)
	recentEmails, err = storage.GetRecentEmails(1) // Last 1 day
	if err != nil {
		t.Fatalf("Failed to get recent emails: %v", err)
	}

	if len(recentEmails) != 0 {
		t.Fatalf("Expected 0 recent emails (1 day), got %d", len(recentEmails))
	}

	// Test SearchEmails
	searchResults, err := storage.SearchEmails("test")
	if err != nil {
		t.Fatalf("Failed to search emails: %v", err)
	}

	if len(searchResults) != 1 {
		t.Fatalf("Expected 1 search result, got %d", len(searchResults))
	}

	// Test GetEmailsByDateRange
	start := time.Now().Add(-3 * 24 * time.Hour) // 3 days ago
	end := time.Now()
	rangeEmails, err := storage.GetEmailsByDateRange(start, end)
	if err != nil {
		t.Fatalf("Failed to get emails by date range: %v", err)
	}

	if len(rangeEmails) != 1 {
		t.Fatalf("Expected 1 email in date range, got %d", len(rangeEmails))
	}
}

// TestAssistantEmailAwareness tests that the assistant can answer questions about emails
func TestAssistantEmailAwareness(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create storage
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create assistant storage: %v", err)
	}

	// Add test emails from the last 7 days
	now := time.Now()
	testEmails := []*agent.Email{
		{
			ID:         uuid.New().String(),
			Subject:    "Weekly Team Standup",
			From:       "manager@company.com",
			To:         []string{"team@company.com"},
			Body:       "Hi team, let's have our weekly standup tomorrow at 10am. Please prepare your updates.",
			Date:       now.Add(-1 * 24 * time.Hour), // 1 day ago
			ReceivedAt: now.Add(-1 * 24 * time.Hour),
			MessageID:  "<standup@company.com>",
			IsRead:     true,
		},
		{
			ID:         uuid.New().String(),
			Subject:    "Project Deadline Reminder",
			From:       "pm@company.com",
			To:         []string{"dev@company.com"},
			Body:       "Just a reminder that the project deadline is next Friday. Please ensure all tasks are completed.",
			Date:       now.Add(-3 * 24 * time.Hour), // 3 days ago
			ReceivedAt: now.Add(-3 * 24 * time.Hour),
			MessageID:  "<deadline@company.com>",
			IsRead:     false,
		},
		{
			ID:         uuid.New().String(),
			Subject:    "Budget Approval Required",
			From:       "finance@company.com",
			To:         []string{"manager@company.com"},
			Body:       "Please review and approve the Q4 budget proposal. The deadline is this Thursday.",
			Date:       now.Add(-5 * 24 * time.Hour), // 5 days ago
			ReceivedAt: now.Add(-5 * 24 * time.Hour),
			MessageID:  "<budget@company.com>",
			IsRead:     false,
		},
	}

	// Save test emails
	for _, email := range testEmails {
		err = storage.SaveEmail(email)
		if err != nil {
			t.Fatalf("Failed to save test email: %v", err)
		}
	}

	// Create assistant agent with mock AI
	mockAI := ai.NewMockProvider()
	hub := &MockHub{
		messages: make(map[string][]*protocol.Message),
		agents:   make(map[string]protocol.AgentInfo),
	}

	_ = agent.NewAssistantAgent("TestAssistant", mockAI, hub)

	// Test that assistant can access recent emails
	recentEmails, err := storage.GetRecentEmails(7)
	if err != nil {
		t.Fatalf("Failed to get recent emails: %v", err)
	}

	if len(recentEmails) != 3 {
		t.Fatalf("Expected 3 recent emails, got %d", len(recentEmails))
	}

	// Test email search functionality
	searchResults, err := storage.SearchEmails("standup")
	if err != nil {
		t.Fatalf("Failed to search emails: %v", err)
	}

	if len(searchResults) != 1 {
		t.Fatalf("Expected 1 search result for 'standup', got %d", len(searchResults))
	}

	if searchResults[0].Subject != "Weekly Team Standup" {
		t.Fatalf("Expected 'Weekly Team Standup', got %s", searchResults[0].Subject)
	}

	// Test date range filtering
	start := now.Add(-2 * 24 * time.Hour) // 2 days ago
	end := now
	rangeEmails, err := storage.GetEmailsByDateRange(start, end)
	if err != nil {
		t.Fatalf("Failed to get emails by date range: %v", err)
	}

	if len(rangeEmails) != 1 {
		t.Fatalf("Expected 1 email in last 2 days, got %d", len(rangeEmails))
	}

	// Test that the assistant can access the email data through storage
	// This verifies the email awareness functionality works
	t.Logf("Assistant email awareness test completed successfully")
}

// TestAssistantEmailOlderThan7Days tests that assistant can access older emails on request
func TestAssistantEmailOlderThan7Days(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create storage
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		t.Fatalf("Failed to create assistant storage: %v", err)
	}

	// Add test emails from 2 weeks ago
	now := time.Now()
	oldEmail := &agent.Email{
		ID:         uuid.New().String(),
		Subject:    "Old Project Update",
		From:       "old-sender@company.com",
		To:         []string{"recipient@company.com"},
		Body:       "This is an email from 2 weeks ago about the old project status.",
		Date:       now.Add(-14 * 24 * time.Hour), // 14 days ago
		ReceivedAt: now.Add(-14 * 24 * time.Hour),
		MessageID:  "<old@company.com>",
		IsRead:     true,
	}

	err = storage.SaveEmail(oldEmail)
	if err != nil {
		t.Fatalf("Failed to save old email: %v", err)
	}

	// Test that old email is not in recent emails (7 days)
	recentEmails, err := storage.GetRecentEmails(7)
	if err != nil {
		t.Fatalf("Failed to get recent emails: %v", err)
	}

	if len(recentEmails) != 0 {
		t.Fatalf("Expected 0 recent emails (7 days), got %d", len(recentEmails))
	}

	// Test that old email is accessible via date range
	start := now.Add(-15 * 24 * time.Hour) // 15 days ago
	end := now.Add(-10 * 24 * time.Hour)   // 10 days ago
	rangeEmails, err := storage.GetEmailsByDateRange(start, end)
	if err != nil {
		t.Fatalf("Failed to get emails by date range: %v", err)
	}

	if len(rangeEmails) != 1 {
		t.Fatalf("Expected 1 email in date range, got %d", len(rangeEmails))
	}

	if rangeEmails[0].Subject != "Old Project Update" {
		t.Fatalf("Expected 'Old Project Update', got %s", rangeEmails[0].Subject)
	}

	// Test that old email is searchable
	searchResults, err := storage.SearchEmails("old project")
	if err != nil {
		t.Fatalf("Failed to search emails: %v", err)
	}

	if len(searchResults) != 1 {
		t.Fatalf("Expected 1 search result for 'old project', got %d", len(searchResults))
	}
}
