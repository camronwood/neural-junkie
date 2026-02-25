package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// TestHubCreation tests basic hub creation and initialization
func TestHubCreation(t *testing.T) {
	h := hub.NewHub()
	if h == nil {
		t.Fatal("Expected hub to be created")
	}
}

// TestChannelCreation tests channel creation and management
func TestChannelCreation(t *testing.T) {
	h := hub.NewHub()

	// Create a channel
	channel := h.CreateChannel("test-channel", "Test channel", "test-project")
	if channel == nil {
		t.Fatal("Expected channel to be created")
	}

	if channel.Name != "test-channel" {
		t.Errorf("Expected channel name 'test-channel', got '%s'", channel.Name)
	}

	if channel.Description != "Test channel" {
		t.Errorf("Expected channel description 'Test channel', got '%s'", channel.Description)
	}

	// Test getting the channel
	retrievedChannel, err := h.GetChannel("test-channel")
	if err != nil {
		t.Fatalf("Expected to get channel, got error: %v", err)
	}

	if retrievedChannel.Name != "test-channel" {
		t.Errorf("Expected retrieved channel name 'test-channel', got '%s'", retrievedChannel.Name)
	}

	// Test listing channels (should have 2: general + test-channel)
	channels := h.ListChannels()
	if len(channels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(channels))
	}

	// Find the test channel
	var testChannel *protocol.Channel
	for _, ch := range channels {
		if ch.Name == "test-channel" {
			testChannel = ch
			break
		}
	}
	if testChannel == nil {
		t.Error("Expected to find test-channel in channel list")
	}
}

// TestChannelNotFound tests error handling for non-existent channels
func TestChannelNotFound(t *testing.T) {
	h := hub.NewHub()

	_, err := h.GetChannel("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent channel")
	}
}

// TestAgentRegistration tests agent registration and management
func TestAgentRegistration(t *testing.T) {
	h := hub.NewHub()

	// Create test agent
	agent := &protocol.AgentInfo{
		ID:        "agent-123",
		Name:      "TestAgent",
		Type:      protocol.AgentTypeBackend,
		Expertise: []string{"go", "apis"},
		Status:    "active",
	}

	// Register agent
	err := h.RegisterAgent(agent)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	// Test getting agent
	retrievedAgent, err := h.GetAgent("agent-123")
	if err != nil {
		t.Fatalf("Expected to get agent, got error: %v", err)
	}

	if retrievedAgent.Name != "TestAgent" {
		t.Errorf("Expected agent name 'TestAgent', got '%s'", retrievedAgent.Name)
	}

	// Test listing agents
	agents := h.ListAgents()
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}

	if agents[0].Name != "TestAgent" {
		t.Errorf("Expected agent name 'TestAgent', got '%s'", agents[0].Name)
	}

	// Test unregistering agent
	err = h.UnregisterAgent("agent-123")
	if err != nil {
		t.Fatalf("Expected agent unregistration to succeed, got error: %v", err)
	}

	// Test getting unregistered agent
	_, err = h.GetAgent("agent-123")
	if err == nil {
		t.Error("Expected error for unregistered agent")
	}
}

// TestAgentChannelJoining tests agent joining and leaving channels
func TestAgentChannelJoining(t *testing.T) {
	h := hub.NewHub()

	// Create channel and agent
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")
	agent := &protocol.AgentInfo{
		ID:        "agent-123",
		Name:      "TestAgent",
		Type:      protocol.AgentTypeBackend,
		Expertise: []string{"go", "apis"},
		Status:    "active",
	}

	// Register agent
	err := h.RegisterAgent(agent)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	// Join channel
	err = h.JoinChannel("agent-123", "test-channel")
	if err != nil {
		t.Fatalf("Expected agent to join channel, got error: %v", err)
	}

	// Test getting channel agents
	channelAgents, err := h.GetChannelAgents("test-channel")
	if err != nil {
		t.Fatalf("Expected to get channel agents, got error: %v", err)
	}

	if len(channelAgents) != 1 {
		t.Errorf("Expected 1 agent in channel, got %d", len(channelAgents))
	}

	if channelAgents[0].Name != "TestAgent" {
		t.Errorf("Expected agent name 'TestAgent', got '%s'", channelAgents[0].Name)
	}

	// Leave channel
	err = h.LeaveChannel("agent-123", "test-channel")
	if err != nil {
		t.Fatalf("Expected agent to leave channel, got error: %v", err)
	}

	// Test channel is now empty
	channelAgents, err = h.GetChannelAgents("test-channel")
	if err != nil {
		t.Fatalf("Expected to get channel agents, got error: %v", err)
	}

	if len(channelAgents) != 0 {
		t.Errorf("Expected 0 agents in channel after leaving, got %d", len(channelAgents))
	}
}

// TestHubMessageSending tests message sending and broadcasting
func TestHubMessageSending(t *testing.T) {
	h := hub.NewHub()

	// Create channel
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	// Create test message
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"Hello, world!",
	)

	// Send message
	err := h.SendMessage(msg)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Test getting messages
	messages, err := h.GetMessages("test-channel", 10)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello, world!" {
		t.Errorf("Expected message content 'Hello, world!', got '%s'", messages[0].Content)
	}
}

// TestMessageBroadcasting tests message broadcasting to subscribers
func TestMessageBroadcasting(t *testing.T) {
	h := hub.NewHub()

	// Create channel
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	// Subscribe to channel
	subCh, err := h.Subscribe("test-channel")
	if err != nil {
		t.Fatalf("Expected subscription to succeed, got error: %v", err)
	}

	// Create test message
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"Broadcast test",
	)

	// Send message and wait for broadcast
	done := make(chan bool)
	go func() {
		select {
		case receivedMsg := <-subCh:
			if receivedMsg.Content != "Broadcast test" {
				t.Errorf("Expected message content 'Broadcast test', got '%s'", receivedMsg.Content)
			}
			done <- true
		case <-time.After(1 * time.Second):
			t.Error("Expected to receive broadcast message within 1 second")
			done <- true
		}
	}()

	err = h.SendMessage(msg)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Wait for broadcast
	select {
	case <-done:
		// Test passed
	case <-time.After(2 * time.Second):
		t.Error("Test timed out waiting for broadcast")
	}
}

func TestHubSystemMentionNoiseSuppressed(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	systemMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		"test-channel",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"❌ Agent @ghost not found",
	)
	if err := h.SendMessage(systemMsg); err != nil {
		t.Fatalf("expected system message send to succeed, got %v", err)
	}

	msgs, err := h.GetMessages("test-channel", 10)
	if err != nil {
		t.Fatalf("expected to get messages, got %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected exactly one stored system message, got %d", len(msgs))
	}
	if len(msgs[0].Mentions) != 0 {
		t.Fatalf("expected no mentions on system message, got %v", msgs[0].Mentions)
	}
}

func TestSessionSnapshotHealthAndAtomicSaves(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")
	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{ID: "user-1", Name: "User", Type: "human"},
		"hello snapshot",
	)
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("expected send to succeed: %v", err)
	}

	path := filepath.Join(t.TempDir(), "last-session.json")
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.SaveSessionToFile(path); err != nil {
				t.Errorf("save failed: %v", err)
			}
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected snapshot file to exist, got %v", err)
	}
	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("expected valid JSON snapshot, got %v", err)
	}

	health := h.GetSessionSaveHealth()
	if health["last_size_bytes"].(int) <= 0 {
		t.Fatalf("expected positive snapshot size, got %v", health["last_size_bytes"])
	}
	if health["age_seconds"].(int64) < 0 {
		t.Fatalf("expected non-negative snapshot age, got %v", health["age_seconds"])
	}
}

// TestHubConcurrentOperations tests concurrent hub operations
func TestHubConcurrentOperations(t *testing.T) {
	h := hub.NewHub()

	// Create channel
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	// Test concurrent message sending
	numMessages := 10
	done := make(chan bool, numMessages)

	for i := 0; i < numMessages; i++ {
		go func(i int) {
			msg := protocol.NewMessage(
				protocol.MessageTypeChat,
				"test-channel",
				protocol.AgentInfo{
					ID:   "user-123",
					Name: "TestUser",
					Type: protocol.AgentTypeGeneral,
				},
				"Concurrent message",
			)
			err := h.SendMessage(msg)
			if err != nil {
				t.Errorf("Expected message sending to succeed, got error: %v", err)
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
			return
		}
	}

	// Verify all messages were received
	messages, err := h.GetMessages("test-channel", 20)
	if err != nil {
		t.Fatalf("Expected to get messages, got error: %v", err)
	}

	if len(messages) != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, len(messages))
	}
}

// TestMentionResolution tests mention resolution functionality
func TestMentionResolution(t *testing.T) {
	h := hub.NewHub()

	// Create agents
	agent1 := &protocol.AgentInfo{
		ID:        "agent-1",
		Name:      "BackendExpert",
		Type:      protocol.AgentTypeBackend,
		Expertise: []string{"go", "apis"},
		Status:    "active",
	}

	agent2 := &protocol.AgentInfo{
		ID:        "agent-2",
		Name:      "FrontendExpert",
		Type:      protocol.AgentTypeFrontend,
		Expertise: []string{"react", "ui"},
		Status:    "active",
	}

	// Register agents
	err := h.RegisterAgent(agent1)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	err = h.RegisterAgent(agent2)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	// Test mention resolution
	mentions := []string{"BackendExpert", "FrontendExpert"}
	resolved := h.ResolveMentions(mentions)

	if len(resolved) != 2 {
		t.Errorf("Expected 2 resolved mentions, got %d", len(resolved))
	}

	// Test with validation
	resolvedMap := make(map[string]bool)
	validated := h.ResolveMentionsWithValidation(mentions, resolvedMap)

	if len(validated) != 2 {
		t.Errorf("Expected 2 validated mentions, got %d", len(validated))
	}

	if !resolvedMap["BackendExpert"] || !resolvedMap["FrontendExpert"] {
		t.Error("Expected resolved map to contain both agent names")
	}
}

// TestThreadManagement tests thread creation and management
func TestThreadManagement(t *testing.T) {
	h := hub.NewHub()

	// Create channel
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	// Create parent message
	parentMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"Parent message",
	)

	// Send parent message
	err := h.SendMessage(parentMsg)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Create thread message
	threadMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-456",
			Name: "TestUser2",
			Type: protocol.AgentTypeGeneral,
		},
		"Thread reply",
	)
	threadMsg.ThreadID = parentMsg.ID
	threadMsg.IsThreadReply = true

	// Send thread message
	err = h.SendMessage(threadMsg)
	if err != nil {
		t.Fatalf("Expected thread message sending to succeed, got error: %v", err)
	}

	// Test getting thread messages
	threadMessages, err := h.GetThreadMessages(parentMsg.ID, 10)
	if err != nil {
		t.Fatalf("Expected to get thread messages, got error: %v", err)
	}

	if len(threadMessages) != 1 {
		t.Errorf("Expected 1 thread message, got %d", len(threadMessages))
	}

	if threadMessages[0].Content != "Thread reply" {
		t.Errorf("Expected thread message content 'Thread reply', got '%s'", threadMessages[0].Content)
	}

	// Test getting thread parent author
	parentAuthor := h.GetThreadParentAuthor(parentMsg.ID)
	if parentAuthor != "user-123" {
		t.Errorf("Expected thread parent author 'user-123', got '%s'", parentAuthor)
	}
}

// TestThreadSubscriptions tests thread subscription functionality
func TestThreadSubscriptions(t *testing.T) {
	h := hub.NewHub()

	// Create channel
	_ = h.CreateChannel("test-channel", "Test channel", "test-project")

	// Create parent message
	parentMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-123",
			Name: "TestUser",
			Type: protocol.AgentTypeGeneral,
		},
		"Parent message",
	)

	// Send parent message
	err := h.SendMessage(parentMsg)
	if err != nil {
		t.Fatalf("Expected message sending to succeed, got error: %v", err)
	}

	// Subscribe to thread
	threadSubCh, err := h.SubscribeToThread(parentMsg.ID)
	if err != nil {
		t.Fatalf("Expected thread subscription to succeed, got error: %v", err)
	}

	// Create thread message
	threadMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		"test-channel",
		protocol.AgentInfo{
			ID:   "user-456",
			Name: "TestUser2",
			Type: protocol.AgentTypeGeneral,
		},
		"Thread reply",
	)
	threadMsg.ThreadID = parentMsg.ID
	threadMsg.IsThreadReply = true

	// Send thread message and wait for broadcast
	done := make(chan bool)
	go func() {
		select {
		case receivedMsg := <-threadSubCh:
			if receivedMsg.Content != "Thread reply" {
				t.Errorf("Expected thread message content 'Thread reply', got '%s'", receivedMsg.Content)
			}
			done <- true
		case <-time.After(1 * time.Second):
			t.Error("Expected to receive thread broadcast message within 1 second")
			done <- true
		}
	}()

	err = h.SendMessage(threadMsg)
	if err != nil {
		t.Fatalf("Expected thread message sending to succeed, got error: %v", err)
	}

	// Wait for broadcast
	select {
	case <-done:
		// Test passed
	case <-time.After(2 * time.Second):
		t.Error("Test timed out waiting for thread broadcast")
	}
}

// TestRemovedAgents tests removed agents functionality
func TestRemovedAgents(t *testing.T) {
	h := hub.NewHub()

	// Create agent
	agent := &protocol.AgentInfo{
		ID:        "agent-123",
		Name:      "TestAgent",
		Type:      protocol.AgentTypeBackend,
		Expertise: []string{"go", "apis"},
		Status:    "active",
	}

	// Register agent
	err := h.RegisterAgent(agent)
	if err != nil {
		t.Fatalf("Expected agent registration to succeed, got error: %v", err)
	}

	// Test agent is not removed initially
	if h.IsAgentRemoved("agent-123") {
		t.Error("Expected agent to not be removed initially")
	}

	// Add agent to removed agents
	h.AddRemovedAgent(agent)

	// Test agent is now removed
	if !h.IsAgentRemoved("agent-123") {
		t.Error("Expected agent to be removed")
	}

	// Test getting removed agents
	removedAgents := h.GetRemovedAgents()
	if len(removedAgents) != 1 {
		t.Errorf("Expected 1 removed agent, got %d", len(removedAgents))
	}

	if removedAgents[0].Name != "TestAgent" {
		t.Errorf("Expected removed agent name 'TestAgent', got '%s'", removedAgents[0].Name)
	}

	// Remove from removed agents
	h.RemoveFromRemovedAgents("agent-123")

	// Test agent is no longer removed
	if h.IsAgentRemoved("agent-123") {
		t.Error("Expected agent to not be removed after removal")
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	h := hub.NewHub()

	// Test joining non-existent channel
	err := h.JoinChannel("agent-123", "non-existent")
	if err == nil {
		t.Error("Expected error when joining non-existent channel")
	}

	// Test leaving non-existent channel
	err = h.LeaveChannel("agent-123", "non-existent")
	if err == nil {
		t.Error("Expected error when leaving non-existent channel")
	}

	// Test getting agents from non-existent channel
	_, err = h.GetChannelAgents("non-existent")
	if err == nil {
		t.Error("Expected error when getting agents from non-existent channel")
	}

	// Test getting messages from non-existent channel
	_, err = h.GetMessages("non-existent", 10)
	if err == nil {
		t.Error("Expected error when getting messages from non-existent channel")
	}

	// Test subscribing to non-existent channel
	_, err = h.Subscribe("non-existent")
	if err == nil {
		t.Error("Expected error when subscribing to non-existent channel")
	}
}

func TestHubAttachesCollaborationData(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-collab", "Collaboration channel", "test")

	agentA := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	agentB := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	if err := h.RegisterAgent(agentA); err != nil {
		t.Fatalf("register agentA: %v", err)
	}
	if err := h.RegisterAgent(agentB); err != nil {
		t.Fatalf("register agentB: %v", err)
	}

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("Build feature", []string{"a1", "a2"}, "test-collab", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}

	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "test-collab", *agentA, "Initial proposal")
	msg.SetCollaborationID(collab.ID)
	msg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	if err := h.SendMessage(msg); err != nil {
		t.Fatalf("send message: %v", err)
	}

	stored, err := h.GetMessages("test-collab", 10)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("expected at least one message")
	}
	if stored[0].Metadata == nil || stored[0].Metadata["collaboration_data"] == nil {
		t.Fatalf("expected collaboration_data metadata to be attached")
	}
}

func TestHubAutoIngestsPlanAndTasks(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-collab", "Collaboration channel", "test")

	agentA := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	agentB := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agentA)
	_ = h.RegisterAgent(agentB)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("Plan feature", []string{"a1", "a2"}, "test-collab", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}

	planMsg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "test-collab", *agentA, "Here's my proposal\n\n## Plan\n\n- Task 1: @AgentA - Build API\n- Task 2: @AgentB - Build UI")
	planMsg.SetCollaborationID(collab.ID)
	planMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	if err := h.SendMessage(planMsg); err != nil {
		t.Fatalf("send plan message: %v", err)
	}

	updated, err := cm.GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("get collaboration: %v", err)
	}
	if updated.Plan == nil || updated.Plan.Version == 0 {
		t.Fatalf("expected plan artifact to be updated")
	}
	if len(updated.Tasks) == 0 {
		t.Fatalf("expected tasks to be auto-extracted from plan")
	}
}

func TestHubTaskLifecycleAutoCompletesCollaboration(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-collab", "Collaboration channel", "test")

	agentA := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	agentB := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agentA)
	_ = h.RegisterAgent(agentB)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("Execute tasks", []string{"a1", "a2"}, "test-collab", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	err = cm.SetTasks(collab.ID, []collaboration.CollaborationTask{
		{
			ID:           "task-1",
			Title:        "Build API",
			Description:  "Implement endpoints",
			AssignedTo:   "a1",
			AssignedName: "AgentA",
			Status:       collaboration.TaskPending,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("set tasks: %v", err)
	}

	workingMsg := protocol.NewMessage(protocol.MessageTypeChat, "test-collab", *agentA, "Working on this now")
	workingMsg.SetCollaborationID(collab.ID)
	workingMsg.SetTaskID("task-1")
	if err := h.SendMessage(workingMsg); err != nil {
		t.Fatalf("send working message: %v", err)
	}

	afterWorking, _ := cm.GetCollaboration(collab.ID)
	if len(afterWorking.Tasks) == 0 || afterWorking.Tasks[0].Status != collaboration.TaskInProgress {
		t.Fatalf("expected task to transition to in_progress, got %+v", afterWorking.Tasks)
	}

	doneMsg := protocol.NewMessage(protocol.MessageTypeChat, "test-collab", *agentA, "Task completed and validated")
	doneMsg.SetCollaborationID(collab.ID)
	doneMsg.SetTaskID("task-1")
	if err := h.SendMessage(doneMsg); err != nil {
		t.Fatalf("send completed message: %v", err)
	}

	afterDone, _ := cm.GetCollaboration(collab.ID)
	if afterDone.Tasks[0].Status != collaboration.TaskCompleted {
		t.Fatalf("expected task to be completed, got %s", afterDone.Tasks[0].Status)
	}
	if afterDone.Phase != collaboration.PhaseCompleted {
		t.Fatalf("expected collaboration to auto-complete, got %s", afterDone.Phase)
	}
}

func TestSessionSnapshotRestoresCollaborations(t *testing.T) {
	h := hub.NewHub()
	_ = h.CreateChannel("test-collab", "Collaboration channel", "test")

	agentA := &protocol.AgentInfo{ID: "a1", Name: "AgentA", Type: protocol.AgentTypeBackend, Status: "active"}
	agentB := &protocol.AgentInfo{ID: "a2", Name: "AgentB", Type: protocol.AgentTypeFrontend, Status: "active"}
	_ = h.RegisterAgent(agentA)
	_ = h.RegisterAgent(agentB)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("Persist me", []string{"a1", "a2"}, "test-collab", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	if err := cm.UpdateArtifact(collab.ID, "a1", "AgentA", "## Plan\n\n- Task 1: @AgentA - Persist this"); err != nil {
		t.Fatalf("update artifact: %v", err)
	}

	path := filepath.Join(t.TempDir(), "session.json")
	if err := h.SaveSessionToFile(path); err != nil {
		t.Fatalf("save session: %v", err)
	}

	restored := hub.NewHub()
	if err := restored.LoadSessionFromFile(path); err != nil {
		t.Fatalf("load session: %v", err)
	}

	restoredCollab, err := restored.GetCollaborationManager().GetCollaboration(collab.ID)
	if err != nil {
		t.Fatalf("expected restored collaboration, got %v", err)
	}
	if restoredCollab.Plan == nil || restoredCollab.Plan.Version == 0 {
		t.Fatalf("expected restored plan artifact, got %+v", restoredCollab.Plan)
	}
}
