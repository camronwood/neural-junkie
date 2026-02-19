package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

// Hub manages the chat room, message routing, and agent connections
type Hub struct {
	channels map[string]*protocol.Channel
	agents   map[string]*protocol.AgentInfo
	messages map[string][]*protocol.Message // channel -> messages

	// Thread management
	threads             map[string][]*protocol.Message      // thread ID -> thread messages
	threadMetadata      map[string]*protocol.ThreadMetadata // thread ID -> metadata
	threadParentAuthors map[string]string                   // thread ID -> parent message author ID

	// Subscribers for real-time updates
	subscribers       map[string][]chan *protocol.Message // channel -> subscriber channels
	threadSubscribers map[string][]chan *protocol.Message // thread ID -> subscriber channels

	// Removed agents tracking (agents not in any channel but still registered)
	removedAgents map[string]*protocol.AgentInfo // agent ID -> agent info

	// Command handler for processing chat commands
	commandHandler *CommandHandler

	// File change manager for handling file change approvals
	fileChangeManager *filechange.FileChangeManager

	// Workspace manager for handling workspace operations
	workspaceManager *WorkspaceManager

	mu sync.RWMutex
}

// NewHub creates a new chat hub
func NewHub() *Hub {
	hub := &Hub{
		channels:            make(map[string]*protocol.Channel),
		agents:              make(map[string]*protocol.AgentInfo),
		messages:            make(map[string][]*protocol.Message),
		threads:             make(map[string][]*protocol.Message),
		threadMetadata:      make(map[string]*protocol.ThreadMetadata),
		threadParentAuthors: make(map[string]string),
		subscribers:         make(map[string][]chan *protocol.Message),
		threadSubscribers:   make(map[string][]chan *protocol.Message),
		removedAgents:       make(map[string]*protocol.AgentInfo),
	}

	// Create default channel
	hub.CreateChannel("general", "General discussion", "")

	// Initialize command handler
	commandHandler, err := NewCommandHandler(hub)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize command handler: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Command handler initialized successfully\n")
	}
	hub.commandHandler = commandHandler

	// Initialize file change manager
	executor := filechange.NewFileChangeExecutor(".")
	hub.fileChangeManager = filechange.NewFileChangeManager(executor)
	fmt.Printf("DEBUG: File change manager initialized successfully\n")

	// Initialize workspace manager
	workspaceManager, err := NewWorkspaceManager()
	if err != nil {
		fmt.Printf("Warning: Failed to initialize workspace manager: %v\n", err)
	} else {
		hub.workspaceManager = workspaceManager
		fmt.Printf("DEBUG: Workspace manager initialized successfully\n")
	}

	return hub
}

// GetWorkspaceManager returns the workspace manager
func (h *Hub) GetWorkspaceManager() *WorkspaceManager {
	return h.workspaceManager
}

// CreateChannel creates a new channel
func (h *Hub) CreateChannel(name, description, project string) *protocol.Channel {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel := &protocol.Channel{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Project:     project,
		Created:     time.Now(),
		Agents:      []protocol.AgentInfo{},
		Tags:        []string{},
	}

	h.channels[name] = channel
	h.messages[name] = []*protocol.Message{}
	h.subscribers[name] = []chan *protocol.Message{}

	return channel
}

// GetChannel returns a channel by name
func (h *Hub) GetChannel(name string) (*protocol.Channel, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channel, ok := h.channels[name]
	if !ok {
		return nil, fmt.Errorf("channel %s not found", name)
	}

	return channel, nil
}

// ListChannels returns all available channels
func (h *Hub) ListChannels() []*protocol.Channel {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channels := make([]*protocol.Channel, 0, len(h.channels))
	for _, ch := range h.channels {
		channels = append(channels, ch)
	}

	return channels
}

// RegisterAgent registers a new agent
func (h *Hub) RegisterAgent(agent *protocol.AgentInfo) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove ALL existing agents with the same name/type (handles restarts and duplicates)
	// When an agent restarts, it gets a new ID, so we need to clean up old entries
	idsToDelete := []string{}
	for id, existingAgent := range h.agents {
		if existingAgent.Name == agent.Name && existingAgent.Type == agent.Type && id != agent.ID {
			idsToDelete = append(idsToDelete, id)
		}
	}

	// Delete all old agents with this name/type
	for _, id := range idsToDelete {
		oldAgent := h.agents[id]
		// Remove from all channels
		for _, channel := range h.channels {
			newAgents := []protocol.AgentInfo{}
			for _, a := range channel.Agents {
				if a.ID != id {
					newAgents = append(newAgents, a)
				}
			}
			channel.Agents = newAgents
		}
		delete(h.agents, id)
		fmt.Printf("🧹 Removed duplicate agent: %s (ID: %s)\n", oldAgent.Name, id[:8])
	}

	// Register the new agent
	h.agents[agent.ID] = agent
	return nil
}

// UnregisterAgent removes an agent
func (h *Hub) UnregisterAgent(agentID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.agents[agentID]; !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	delete(h.agents, agentID)

	// Remove from all channels
	for _, channel := range h.channels {
		for i, agent := range channel.Agents {
			if agent.ID == agentID {
				channel.Agents = append(channel.Agents[:i], channel.Agents[i+1:]...)
				break
			}
		}
	}

	return nil
}

// JoinChannel adds an agent to a channel
func (h *Hub) JoinChannel(agentID, channelName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	channel, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	// Check if already in channel
	for _, a := range channel.Agents {
		if a.ID == agentID {
			return nil // Already in channel
		}
	}

	channel.Agents = append(channel.Agents, *agent)

	// Send join message
	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		channelName,
		*agent,
		fmt.Sprintf("%s (%s) has joined the channel", agent.Name, agent.Type),
	)

	h.messages[channelName] = append(h.messages[channelName], joinMsg)
	h.broadcast(channelName, joinMsg)

	return nil
}

// LeaveChannel removes an agent from a channel
func (h *Hub) LeaveChannel(agentID, channelName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	channel, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	// Remove from channel
	for i, a := range channel.Agents {
		if a.ID == agentID {
			channel.Agents = append(channel.Agents[:i], channel.Agents[i+1:]...)
			break
		}
	}

	// Send leave message
	leaveMsg := protocol.NewMessage(
		protocol.MessageTypeAgentLeave,
		channelName,
		*agent,
		fmt.Sprintf("%s has left the channel", agent.Name),
	)

	h.messages[channelName] = append(h.messages[channelName], leaveMsg)
	h.broadcast(channelName, leaveMsg)

	return nil
}

// SendMessage sends a message to a channel
func (h *Hub) SendMessage(msg *protocol.Message) error {
	// Parse and resolve @mentions before processing
	mentionStrings := protocol.ParseMentions(msg.Content)
	hasInvalidMentions := false

	if len(mentionStrings) > 0 {
		// Resolve mentions and check for unresolved ones
		resolvedMentions := make(map[string]bool) // track which mentions were resolved
		agentIDs := h.ResolveMentionsWithValidation(mentionStrings, resolvedMentions)
		msg.Mentions = agentIDs

		// Send system messages for unresolved mentions
		for _, mention := range mentionStrings {
			if !resolvedMentions[mention] {
				hasInvalidMentions = true

				// Send error message for not found agent
				errorMsg := protocol.NewMessage(
					protocol.MessageTypeSystemInfo,
					msg.Channel,
					protocol.AgentInfo{
						ID:   "system",
						Name: "System",
						Type: protocol.AgentTypeGeneral,
					},
					fmt.Sprintf("❌ Agent @%s not found. Available agents: %s",
						mention, h.getAgentListString()),
				)

				// Lock and send error message immediately
				h.mu.Lock()
				h.messages[msg.Channel] = append(h.messages[msg.Channel], errorMsg)
				h.broadcast(msg.Channel, errorMsg)
				h.mu.Unlock()
			}
		}

		// If all mentions were invalid, don't process the message further
		// This prevents agents from responding to invalid @mentions
		if hasInvalidMentions && len(agentIDs) == 0 {
			// Set mentions to a dummy value so agents will see HasMentions() = true
			// but IsMentioned(agentID) = false, preventing all agents from responding
			msg.Mentions = []string{"__INVALID__"}

			// Store the message for history so user can see what they typed
			h.mu.Lock()
			h.messages[msg.Channel] = append(h.messages[msg.Channel], msg)
			h.mu.Unlock()
			return nil
		}
	}

	// Check if it's a command - process commands from both chat and question types
	if h.commandHandler != nil && len(msg.Content) > 0 && msg.Content[0] == '/' {
		fmt.Printf("DEBUG: Command detected: %s\n", msg.Content)
		// Process command (unlock mutex for command processing)
		ctx := context.Background()
		response, err := h.commandHandler.ProcessCommand(ctx, msg)
		if err != nil {
			return fmt.Errorf("command processing error: %w", err)
		}

		// If command was processed, send the response instead
		if response != nil {
			msg = response
		}
	}

	// Check for automatic repository agent creation
	if h.commandHandler != nil && !msg.IsFromSystem() {
		// Detect local file paths in the message
		pathResult := protocol.DetectLocalPaths(msg.Content)
		if pathResult.Found {
			// Get the best path for repository analysis
			bestPath := protocol.GetBestPathForRepository(pathResult.Paths)
			if bestPath != nil {
				// Check if we should auto-create a repo agent
				shouldAutoCreate := h.shouldAutoCreateRepoAgent(msg, bestPath.Path)
				if shouldAutoCreate {
					// Auto-create repository agent
					h.autoCreateRepoAgent(msg, bestPath.Path)
				}
			}
		}
	}

	// Intercept file change proposals and register with FileChangeManager
	if msg.Type == protocol.MessageTypeFileChange && msg.Metadata != nil {
		if proposalRaw, ok := msg.Metadata["file_change_proposal"]; ok {
			h.registerFileChangeProposal(msg, proposalRaw)
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[msg.Channel]; !ok {
		return fmt.Errorf("channel %s not found", msg.Channel)
	}

	// Handle thread messages separately
	if msg.IsInThread() {
		threadID := msg.GetThreadID()

		// Track the parent message author (first time we see this thread)
		if _, exists := h.threadParentAuthors[threadID]; !exists {
			// Find the parent message (threadID == parent message ID)
			for _, channelMsg := range h.messages[msg.Channel] {
				if channelMsg.ID == threadID {
					h.threadParentAuthors[threadID] = channelMsg.From.ID
					break
				}
			}
		}

		// Add to thread storage
		h.threads[threadID] = append(h.threads[threadID], msg)

		// Update thread metadata
		h.updateThreadMetadata(threadID, msg)

		// Broadcast to thread subscribers (for thread panel UI updates)
		h.broadcastToThread(threadID, msg)

		// ALSO add to channel message history so agents can see it when polling
		h.messages[msg.Channel] = append(h.messages[msg.Channel], msg)

		// ALSO broadcast to channel subscribers (so agents can see mentions)
		h.broadcast(msg.Channel, msg)
	} else {
		// Regular channel message
		h.messages[msg.Channel] = append(h.messages[msg.Channel], msg)
		h.broadcast(msg.Channel, msg)
	}

	return nil
}

// GetMessages returns messages from a channel
func (h *Hub) GetMessages(channelName string, limit int) ([]*protocol.Message, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgs, ok := h.messages[channelName]
	if !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}

	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}

	return msgs[start:], nil
}

// ResolveMentions converts @mention strings (names/types) to agent IDs
// Supports both agent names and agent types
// Example: mentions = ["alice", "backend"] returns IDs for agent "Alice" + all backend agents
func (h *Hub) ResolveMentions(mentions []string) []string {
	resolved := make(map[string]bool)
	return h.ResolveMentionsWithValidation(mentions, resolved)
}

// ResolveMentionsWithValidation converts @mention strings to agent IDs and tracks which were resolved
func (h *Hub) ResolveMentionsWithValidation(mentions []string, resolvedMap map[string]bool) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	seen := make(map[string]bool)
	var agentIDs []string

	for _, mention := range mentions {
		mentionLower := strings.ToLower(mention)
		found := false

		// Check for exact agent name match (case-insensitive)
		for _, agent := range h.agents {
			if strings.EqualFold(agent.Name, mentionLower) && !seen[agent.ID] {
				agentIDs = append(agentIDs, agent.ID)
				seen[agent.ID] = true
				found = true
			}
		}

		// Check for agent type match
		for _, agent := range h.agents {
			if strings.EqualFold(string(agent.Type), mentionLower) && !seen[agent.ID] {
				agentIDs = append(agentIDs, agent.ID)
				seen[agent.ID] = true
				found = true
			}
		}

		// Track if this mention was resolved
		if resolvedMap != nil {
			resolvedMap[mention] = found
		}
	}

	return agentIDs
}

// getAgentListString returns a comma-separated list of agent names for error messages
func (h *Hub) getAgentListString() string {
	var names []string
	for _, agent := range h.agents {
		names = append(names, "@"+agent.Name)
	}
	if len(names) == 0 {
		return "(no agents available)"
	}
	return strings.Join(names, ", ")
}

// Subscribe creates a subscription to a channel for real-time updates
func (h *Hub) Subscribe(channelName string) (chan *protocol.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channelName]; !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	ch := make(chan *protocol.Message, 100)
	h.subscribers[channelName] = append(h.subscribers[channelName], ch)

	return ch, nil
}

// Unsubscribe removes a subscription
func (h *Hub) Unsubscribe(channelName string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[channelName]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			h.subscribers[channelName] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcast sends a message to all subscribers of a channel (must be called with lock held)
func (h *Hub) broadcast(channelName string, msg *protocol.Message) {
	subs, ok := h.subscribers[channelName]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// GetChannelAgents returns all agents in a channel
func (h *Hub) GetChannelAgents(channelName string) ([]protocol.AgentInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channel, ok := h.channels[channelName]
	if !ok {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	return channel.Agents, nil
}

// GetAgent returns agent info by ID
func (h *Hub) GetAgent(agentID string) (*protocol.AgentInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agent, ok := h.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// ListAgents returns all registered agents
func (h *Hub) ListAgents() []*protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*protocol.AgentInfo, 0, len(h.agents))
	for _, agent := range h.agents {
		agents = append(agents, agent)
	}

	return agents
}

// GetThreadMessages returns messages from a thread
func (h *Hub) GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgs, ok := h.threads[threadID]
	if !ok {
		// Thread doesn't exist yet, return empty list (not an error)
		return []*protocol.Message{}, nil
	}

	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}

	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}

	return msgs[start:], nil
}

// GetThreadMetadata returns metadata for a thread
func (h *Hub) GetThreadMetadata(threadID string) (*protocol.ThreadMetadata, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	metadata, ok := h.threadMetadata[threadID]
	if !ok {
		// Return empty metadata if thread doesn't exist yet
		return &protocol.ThreadMetadata{
			ThreadID:      threadID,
			ReplyCount:    0,
			LastReplyTime: time.Time{},
			Participants:  []string{},
		}, nil
	}

	return metadata, nil
}

// GetThreadParentAuthor returns the agent ID of the author of the parent message for a thread
func (h *Hub) GetThreadParentAuthor(threadID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	authorID, ok := h.threadParentAuthors[threadID]
	if !ok {
		return "" // Thread doesn't exist or parent not tracked
	}

	return authorID
}

// updateThreadMetadata updates thread metadata when a new message is added (must be called with lock held)
func (h *Hub) updateThreadMetadata(threadID string, msg *protocol.Message) {
	metadata, ok := h.threadMetadata[threadID]
	if !ok {
		// Create new metadata
		metadata = &protocol.ThreadMetadata{
			ThreadID:      threadID,
			ReplyCount:    0,
			LastReplyTime: time.Time{},
			Participants:  []string{},
		}
		h.threadMetadata[threadID] = metadata
	}

	// Increment reply count
	metadata.ReplyCount++

	// Update last reply time
	metadata.LastReplyTime = msg.Timestamp

	// Add participant if not already present
	participantName := msg.From.Name
	found := false
	for _, p := range metadata.Participants {
		if p == participantName {
			found = true
			break
		}
	}
	if !found {
		metadata.Participants = append(metadata.Participants, participantName)
	}
}

// SubscribeToThread creates a subscription to a thread for real-time updates
func (h *Hub) SubscribeToThread(threadID string) (chan *protocol.Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan *protocol.Message, 100)
	h.threadSubscribers[threadID] = append(h.threadSubscribers[threadID], ch)

	return ch, nil
}

// UnsubscribeFromThread removes a thread subscription
func (h *Hub) UnsubscribeFromThread(threadID string, ch chan *protocol.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.threadSubscribers[threadID]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			h.threadSubscribers[threadID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// broadcastToThread sends a message to all thread subscribers (must be called with lock held)
func (h *Hub) broadcastToThread(threadID string, msg *protocol.Message) {
	subs, ok := h.threadSubscribers[threadID]
	if !ok {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// GetRemovedAgents returns all agents that have been removed from conversations
func (h *Hub) GetRemovedAgents() []*protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agents := make([]*protocol.AgentInfo, 0, len(h.removedAgents))
	for _, agent := range h.removedAgents {
		agents = append(agents, agent)
	}
	return agents
}

// IsAgentRemoved checks if an agent is in the removed state
func (h *Hub) IsAgentRemoved(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.removedAgents[agentID]
	return exists
}

// AddRemovedAgent adds an agent to the removed agents list
func (h *Hub) AddRemovedAgent(agent *protocol.AgentInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.removedAgents[agent.ID] = agent
}

// RemoveFromRemovedAgents removes an agent from the removed agents list
func (h *Hub) RemoveFromRemovedAgents(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.removedAgents, agentID)
}

// IsAgentInAnyChannel checks if an agent is currently in any channel
func (h *Hub) IsAgentInAnyChannel(agentID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, channel := range h.channels {
		for _, agent := range channel.Agents {
			if agent.ID == agentID {
				return true
			}
		}
	}
	return false
}

// shouldAutoCreateRepoAgent determines if we should auto-create a repo agent for this message
func (h *Hub) shouldAutoCreateRepoAgent(msg *protocol.Message, repoPath string) bool {
	// Don't auto-create if it's a system message
	if msg.IsFromSystem() {
		return false
	}

	// Don't auto-create if there's already a pending review for this path
	if h.commandHandler.HasPendingReview(repoPath) {
		return false
	}

	// Don't auto-create if there's already a repo agent for this path
	agents := h.ListAgents()
	for _, agent := range agents {
		if agent.Type == protocol.AgentTypeRepo && agent.RepositoryPath == repoPath {
			return false
		}
	}

	// Only auto-create if the message mentions a regular agent (not repo agent)
	// and contains repository-related keywords
	hasRegularAgentMention := false
	hasRepoKeywords := false

	// Check for regular agent mentions
	for _, mention := range msg.Mentions {
		agent, err := h.GetAgent(mention)
		if err == nil && agent.Type != protocol.AgentTypeRepo {
			hasRegularAgentMention = true
			break
		}
	}

	// Check for repository-related keywords
	content := strings.ToLower(msg.Content)
	repoKeywords := []string{
		"review", "analyze", "check", "examine", "look at", "code review",
		"architecture", "structure", "codebase", "repository", "project",
		"help with", "assist with", "understand", "explain",
	}

	for _, keyword := range repoKeywords {
		if strings.Contains(content, keyword) {
			hasRepoKeywords = true
			break
		}
	}

	return hasRegularAgentMention && hasRepoKeywords
}

// autoCreateRepoAgent automatically creates a repository agent for the given path
func (h *Hub) autoCreateRepoAgent(originalMsg *protocol.Message, repoPath string) {
	// Generate agent name from path
	agentName := protocol.NormalizeAgentName(filepath.Base(repoPath) + "Expert")

	// Send initial feedback message
	feedbackMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		originalMsg.Channel,
		protocol.AgentInfo{
			ID:   "system",
			Name: "System",
			Type: protocol.AgentTypeGeneral,
		},
		fmt.Sprintf("🤖 Detected repository path: %s\n"+
			"Creating repository expert agent automatically...\n"+
			"This will take 30-60 seconds for first-time indexing.\n"+
			"The agent will respond to your question once ready.",
			repoPath),
	)

	// Send feedback message
	h.mu.Lock()
	h.messages[originalMsg.Channel] = append(h.messages[originalMsg.Channel], feedbackMsg)
	h.broadcast(originalMsg.Channel, feedbackMsg)
	h.mu.Unlock()

	// Add to pending reviews
	h.commandHandler.AddPendingReview(repoPath, originalMsg, agentName)

	// Create the repo agent in a goroutine to avoid blocking
	go func() {
		ctx := context.Background()

		// Create a synthetic message for the command handler
		createMsg := &protocol.Message{
			ID:      originalMsg.ID + "_auto_create",
			Type:    protocol.MessageTypeQuestion,
			Channel: originalMsg.Channel,
			From:    originalMsg.From,
			Content: fmt.Sprintf("/create-repo-agent %s %s", repoPath, agentName),
			Metadata: map[string]interface{}{
				"auto_created":    true,
				"original_msg_id": originalMsg.ID,
			},
		}

		// Process the create command
		response, err := h.commandHandler.ProcessCommand(ctx, createMsg)
		if err != nil {
			// Send error message
			errorMsg := protocol.NewMessage(
				protocol.MessageTypeSystemInfo,
				originalMsg.Channel,
				protocol.AgentInfo{
					ID:   "system",
					Name: "System",
					Type: protocol.AgentTypeGeneral,
				},
				fmt.Sprintf("❌ Failed to auto-create repository agent: %v", err),
			)

			h.mu.Lock()
			h.messages[originalMsg.Channel] = append(h.messages[originalMsg.Channel], errorMsg)
			h.broadcast(originalMsg.Channel, errorMsg)
			h.mu.Unlock()

			// Remove from pending reviews
			h.commandHandler.RemovePendingReview(repoPath)
		} else if response != nil {
			// Send the response message
			h.mu.Lock()
			h.messages[originalMsg.Channel] = append(h.messages[originalMsg.Channel], response)
			h.broadcast(originalMsg.Channel, response)
			h.mu.Unlock()
		}
	}()
}

// GetCommandHandler returns the command handler for external access
func (h *Hub) GetCommandHandler() agent.CommandHandlerInterface {
	return h.commandHandler
}

// GetCommandDefinitions returns the metadata for all slash commands.
// Returns nil if no command handler is configured.
func (h *Hub) GetCommandDefinitions() []protocol.CommandDefinition {
	if h.commandHandler == nil {
		return nil
	}
	return h.commandHandler.GetCommandDefinitions()
}

// GetFileChangeManager returns the file change manager for external access
func (h *Hub) GetFileChangeManager() *filechange.FileChangeManager {
	return h.fileChangeManager
}

// registerFileChangeProposal extracts a FileChangeProposal from message metadata
// and registers it with the FileChangeManager so it appears in the pending changes UI.
func (h *Hub) registerFileChangeProposal(msg *protocol.Message, proposalRaw interface{}) {
	// Convert the raw proposal to typed struct via JSON round-trip
	proposalBytes, err := json.Marshal(proposalRaw)
	if err != nil {
		log.Printf("[FileChange] Failed to marshal proposal: %v", err)
		return
	}

	var proposal protocol.FileChangeProposal
	if err := json.Unmarshal(proposalBytes, &proposal); err != nil {
		log.Printf("[FileChange] Failed to unmarshal proposal: %v", err)
		return
	}

	// Resolve file path against workspace
	filePath := h.resolveWorkspacePath(proposal.FilePath, msg)

	// Map proposal operation string to filechange.FileOperation
	var operation filechange.FileOperation
	switch proposal.Operation {
	case "create":
		operation = filechange.FileOperationCreate
	case "edit":
		operation = filechange.FileOperationEdit
	case "delete":
		operation = filechange.FileOperationDelete
	case "move":
		operation = filechange.FileOperationMove
	default:
		log.Printf("[FileChange] Unknown operation: %s", proposal.Operation)
		return
	}

	// Resolve paths for move operations
	oldPath := proposal.OldPath
	newPath := proposal.NewPath
	if operation == filechange.FileOperationMove {
		oldPath = h.resolveWorkspacePath(proposal.OldPath, msg)
		newPath = h.resolveWorkspacePath(proposal.NewPath, msg)
	}

	// Update the executor workspace root if we can resolve it
	if wsRoot := h.resolveWorkspaceRoot(msg); wsRoot != "" {
		h.fileChangeManager.GetExecutor().SetWorkspaceRoot(wsRoot)
	}

	// Register with FileChangeManager
	change, err := h.fileChangeManager.ProposeFileChange(
		operation,
		filePath,
		oldPath,
		newPath,
		proposal.OldContent,
		proposal.NewContent,
		msg.From,
		msg.Channel,
	)
	if err != nil {
		log.Printf("[FileChange] Failed to register proposal: %v", err)
		return
	}

	// Update the message metadata with the registered change ID so the UI can link them
	msg.Metadata["registered_change_id"] = change.ID

	log.Printf("[FileChange] Registered %s proposal for %s (change ID: %s) from %s",
		proposal.Operation, filePath, change.ID, msg.From.Name)
}

// resolveWorkspacePath resolves a potentially relative file path against the workspace.
// It checks message metadata for workspace_context first, then falls back to WorkspaceManager.
func (h *Hub) resolveWorkspacePath(filePath string, msg *protocol.Message) string {
	// If path is already absolute, return as-is
	if filepath.IsAbs(filePath) {
		return filePath
	}

	// Try to get workspace path from message metadata (workspace_context)
	if msg.Metadata != nil {
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
				if wsPath, ok := ctxMap["workspace_path"].(string); ok && wsPath != "" {
					return filepath.Join(wsPath, filePath)
				}
			}
		}
	}

	// Fallback: try the first workspace from WorkspaceManager
	if h.workspaceManager != nil {
		workspaces := h.workspaceManager.ListWorkspaces()
		if len(workspaces) > 0 {
			return filepath.Join(workspaces[0].Path, filePath)
		}
	}

	// Last resort: return as-is (relative to CWD)
	return filePath
}

// resolveWorkspaceRoot returns the workspace root path from message context or WorkspaceManager.
func (h *Hub) resolveWorkspaceRoot(msg *protocol.Message) string {
	// Try to get workspace path from message metadata (workspace_context)
	if msg.Metadata != nil {
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
				if wsPath, ok := ctxMap["workspace_path"].(string); ok && wsPath != "" {
					return wsPath
				}
			}
		}
	}

	// Fallback: try the first workspace from WorkspaceManager
	if h.workspaceManager != nil {
		workspaces := h.workspaceManager.ListWorkspaces()
		if len(workspaces) > 0 {
			return workspaces[0].Path
		}
	}

	return ""
}

// --- Session Recording ---

// SessionSnapshot captures the full state of a chat session for debugging/review.
type SessionSnapshot struct {
	SavedAt  time.Time                   `json:"saved_at"`
	Channels map[string]*ChannelSnapshot `json:"channels"`
	Threads  map[string]*ThreadSnapshot  `json:"threads"`
	Agents   []*protocol.AgentInfo       `json:"agents"`
}

// ChannelSnapshot holds all messages for a single channel.
type ChannelSnapshot struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Messages    []*protocol.Message `json:"messages"`
}

// ThreadSnapshot holds all messages and metadata for a single thread.
type ThreadSnapshot struct {
	ThreadID string                   `json:"thread_id"`
	Metadata *protocol.ThreadMetadata `json:"metadata"`
	Messages []*protocol.Message      `json:"messages"`
}

// TakeSessionSnapshot returns a deep copy of the current session state.
func (h *Hub) TakeSessionSnapshot() *SessionSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := &SessionSnapshot{
		SavedAt:  time.Now(),
		Channels: make(map[string]*ChannelSnapshot),
		Threads:  make(map[string]*ThreadSnapshot),
		Agents:   make([]*protocol.AgentInfo, 0),
	}

	// Capture channel messages
	for name, ch := range h.channels {
		cs := &ChannelSnapshot{
			Name:        ch.Name,
			Description: ch.Description,
			Messages:    make([]*protocol.Message, 0),
		}
		if msgs, ok := h.messages[name]; ok {
			cs.Messages = append(cs.Messages, msgs...)
		}
		snapshot.Channels[name] = cs
	}

	// Capture threads
	for threadID, msgs := range h.threads {
		ts := &ThreadSnapshot{
			ThreadID: threadID,
			Messages: make([]*protocol.Message, len(msgs)),
		}
		copy(ts.Messages, msgs)
		if meta, ok := h.threadMetadata[threadID]; ok {
			ts.Metadata = meta
		}
		snapshot.Threads[threadID] = ts
	}

	// Capture active agents
	for _, a := range h.agents {
		snapshot.Agents = append(snapshot.Agents, a)
	}

	return snapshot
}

// SaveSessionToFile writes the current session snapshot to a JSON file.
// It writes to a temp file first, then renames for atomic replacement.
func (h *Hub) SaveSessionToFile(path string) error {
	snapshot := h.TakeSessionSnapshot()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write to temp file then rename for atomic save
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to finalize session file: %w", err)
	}

	log.Printf("💾 Session saved to %s (%d bytes)", path, len(data))
	return nil
}

// DefaultSessionPath returns the default path for the last session file.
func DefaultSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".neural-junkie", "last-session.json")
}
