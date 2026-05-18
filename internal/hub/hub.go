package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
)

func collaborationWorkspaceContextSnapshot(snap *collaboration.Collaboration) map[string]interface{} {
	if snap == nil || strings.TrimSpace(snap.WorkingDirectory) == "" {
		return nil
	}
	name := strings.TrimSpace(snap.Title)
	if name == "" {
		name = "Collaboration"
	}
	return map[string]interface{}{
		"workspace_name": fmt.Sprintf("Collaboration: %s", name),
		"workspace_path": snap.WorkingDirectory,
		"file_tree":      ".  (sandbox — empty until agents create files)\n",
		"open_files":     []interface{}{},
	}
}

// CollaborationCanDispatchTasks is true when collaboration_task messages may be sent.
func (h *Hub) CollaborationCanDispatchTasks(snap *collaboration.Collaboration) bool {
	if snap == nil || snap.Phase != collaboration.PhaseExecuting {
		return false
	}
	if strings.TrimSpace(snap.WorkingDirectory) == "" {
		return true
	}
	return snap.WorkspaceAcknowledged
}

// AcknowledgeCollaborationWorkspace marks the execution sandbox as user-confirmed,
// dispatches task prompts once, and broadcasts a collaboration_status update.
func (h *Hub) AcknowledgeCollaborationWorkspace(collabID string) error {
	if h.collabManager == nil {
		return fmt.Errorf("collaboration manager unavailable")
	}
	already, _, err := h.collabManager.AcknowledgeWorkspace(collabID)
	if err != nil {
		return err
	}
	snap, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil {
		return fmt.Errorf("collaboration snapshot: %w", err)
	}
	if snap == nil {
		return fmt.Errorf("collaboration snapshot: not found")
	}
	if !already && len(snap.Tasks) > 0 {
		h.dispatchCollabTaskMessages(snap, nil, false)
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		snap.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("✅ **Collaboration workspace ready** (`%s`).", collabID[:8]),
	)
	statusMsg.SetCollaborationID(collabID)
	statusMsg.SetCollaborationPhase(string(snap.Phase))
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = map[string]interface{}{}
	}
	statusMsg.Metadata["collab_skip_attach_dispatch"] = true
	return h.SendMessage(statusMsg)
}

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

	// Tool approval manager for CLI agent tool call approvals
	toolApprovalManager *ToolApprovalManager

	// Collaboration manager for multi-agent collaboration sessions
	collabManager *collaboration.CollaborationManager

	// Per-agent custom rules (markdown), persisted on disk
	agentRulesStore *agent.AgentCustomRulesStorage

	// Session snapshot save synchronization and observability.
	sessionSaveMu   sync.Mutex
	sessionHealthMu sync.RWMutex
	sessionHealth   SessionSaveHealth

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
	hub.CreateChannelWithType("general", "General discussion", "", protocol.ChannelTypePublic, "system")

	// Initialize command handler
	commandHandler, err := NewCommandHandler(hub)
	if err != nil {
		log.Printf("Warning: failed to initialize command handler: %v", err)
	}
	hub.commandHandler = commandHandler

	// Initialize file change manager
	executor := filechange.NewFileChangeExecutor(".")
	hub.fileChangeManager = filechange.NewFileChangeManager(executor)

	// Initialize workspace manager
	workspaceManager, err := NewWorkspaceManager()
	if err != nil {
		log.Printf("Warning: failed to initialize workspace manager: %v", err)
	} else {
		hub.workspaceManager = workspaceManager
	}

	// Initialize tool approval manager
	hub.toolApprovalManager = NewToolApprovalManager(hub)

	// Initialize collaboration manager
	hub.collabManager = collaboration.NewCollaborationManager(hub)

	rulesStore, err := agent.NewAgentCustomRulesStorage()
	if err != nil {
		log.Printf("Warning: agent custom rules storage unavailable: %v", err)
	} else {
		hub.agentRulesStore = rulesStore
	}

	return hub
}

// GetWorkspaceManager returns the workspace manager
func (h *Hub) GetWorkspaceManager() *WorkspaceManager {
	return h.workspaceManager
}

// CreateChannel creates a new channel
func (h *Hub) CreateChannel(name, description, project string) *protocol.Channel {
	return h.CreateChannelWithType(name, description, project, protocol.ChannelTypePublic, "")
}

// CreateChannelWithType creates a new channel with an explicit type and creator
func (h *Hub) CreateChannelWithType(name, description, project string, channelType protocol.ChannelType, createdBy string) *protocol.Channel {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Return existing channel if one with this name already exists
	if existing, ok := h.channels[name]; ok {
		return existing
	}

	channel := &protocol.Channel{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Project:     project,
		Type:        channelType,
		CreatedBy:   createdBy,
		Created:     time.Now(),
		Agents:      []protocol.AgentInfo{},
		Members:     []string{},
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

// SetChannelDescription updates the sidebar-visible description for a channel.
func (h *Hub) SetChannelDescription(name, description string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[name]
	if !ok {
		return fmt.Errorf("channel %s not found", name)
	}
	ch.Description = strings.TrimSpace(description)
	return nil
}

// inferChannelTypeForName fixes DM classification when legacy snapshots omitted
// "type" or stored the wrong value — the UI sidebar keys off type === dm.
func inferChannelTypeForName(name string, t protocol.ChannelType) protocol.ChannelType {
	n := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(n, "dm-") {
		return protocol.ChannelTypeDM
	}
	if strings.HasPrefix(n, "collab-") {
		return protocol.ChannelTypeCollaboration
	}
	if t == "" {
		return protocol.ChannelTypePublic
	}
	return t
}

func (h *Hub) repairChannelTypesLocked() {
	for _, ch := range h.channels {
		if ch == nil {
			continue
		}
		if want := inferChannelTypeForName(ch.Name, ch.Type); want != ch.Type {
			ch.Type = want
		}
	}
}

// ListChannels returns all available channels in a stable order:
// public first, then custom, then collaboration, then DM, alphabetical within each group.
func (h *Hub) ListChannels() []*protocol.Channel {
	h.mu.Lock()
	h.repairChannelTypesLocked()
	channels := make([]*protocol.Channel, 0, len(h.channels))
	for _, ch := range h.channels {
		channels = append(channels, ch)
	}
	h.mu.Unlock()

	typeOrder := map[protocol.ChannelType]int{
		protocol.ChannelTypePublic:        0,
		protocol.ChannelTypeCustom:        1,
		protocol.ChannelTypeCollaboration: 2,
		protocol.ChannelTypeDM:            3,
	}
	sort.Slice(channels, func(i, j int) bool {
		oi, oj := typeOrder[channels[i].Type], typeOrder[channels[j].Type]
		if oi != oj {
			return oi < oj
		}
		return channels[i].Name < channels[j].Name
	})

	return channels
}

// RegisterAgent registers a new agent
func (h *Hub) RegisterAgent(agent *protocol.AgentInfo) error {
	h.mu.Lock()

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
		log.Printf("Removed duplicate agent registration: %s (old id %s)", oldAgent.Name, id[:8])
	}

	// Register the new agent
	h.agents[agent.ID] = agent
	if h.agentRulesStore != nil {
		if md, ok := h.agentRulesStore.Get(agent.ID); ok {
			agent.CustomRulesMarkdown = md
		}
	}
	h.mu.Unlock()

	if h.collabManager != nil {
		h.collabManager.ReconcileRestoredAgentIDs()
	}
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

// JoinChannel adds an agent to a channel. An optional greeting can be
// provided to replace the default join message content -- this avoids
// the need for a separate SendMessage call that would create a duplicate.
func (h *Hub) JoinChannel(agentID, channelName string, greeting ...string) error {
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

	if !h.shouldSkipJoinAnnouncementLocked(channelName, agent) {
		content := fmt.Sprintf("%s (%s) has joined the channel", agent.Name, agent.Type)
		if len(greeting) > 0 && greeting[0] != "" {
			content = greeting[0]
		}

		joinMsg := protocol.NewMessage(
			protocol.MessageTypeAgentJoin,
			channelName,
			*agent,
			content,
		)

		h.appendChannelMessageLocked(channelName, joinMsg)
		h.broadcast(channelName, joinMsg)
	}

	return nil
}

// shouldSkipJoinAnnouncementLocked avoids duplicate join lines when agents rebind after
// hub restart (DM restore, specialist boot) while history already records a prior join.
// Caller must hold h.mu (write lock).
func (h *Hub) shouldSkipJoinAnnouncementLocked(channelName string, agent *protocol.AgentInfo) bool {
	if agent == nil {
		return true
	}
	msgs := h.messages[channelName]
	for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-40; i-- {
		m := msgs[i]
		if m == nil || m.Type != protocol.MessageTypeAgentJoin {
			continue
		}
		if m.From.ID == agent.ID || m.From.Name == agent.Name {
			return true
		}
	}
	return false
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

	h.appendChannelMessageLocked(channelName, leaveMsg)
	h.broadcast(channelName, leaveMsg)

	return nil
}

// SendMessage sends a message to a channel
func (h *Hub) SendMessage(msg *protocol.Message) error {
	if err := h.enforceExecutionMessageBudget(msg); err != nil {
		return err
	}
	h.processCollaborationLifecycle(msg)
	h.attachCollaborationData(msg)

	// Only parse actionable @mentions for user-like senders. Agent responses
	// naturally contain @ symbols (file paths, code references) that should
	// not be treated as mention attempts.
	mentionStrings := []string{}
	allowMentionValidationErrors := protocol.ShouldParseMentions(msg.Type, msg.From)
	if allowMentionValidationErrors || h.shouldParseCollaborationMentions(msg) {
		mentionStrings = protocol.ParseMentions(msg.Content)
	}
	hasInvalidMentions := false

	if len(mentionStrings) > 0 {
		// Resolve mentions and check for unresolved ones
		resolvedMentions := make(map[string]bool) // track which mentions were resolved
		agentIDs := h.ResolveMentionsWithValidation(mentionStrings, resolvedMentions, msg.Channel)
		msg.Mentions = agentIDs

		h.maybeExpandCollaborationParticipants(msg, agentIDs)

		// Send system messages for unresolved mentions (user-authored mentions only).
		if allowMentionValidationErrors {
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
					h.appendChannelMessageLocked(msg.Channel, errorMsg)
					h.broadcast(msg.Channel, errorMsg)
					h.mu.Unlock()
				}
			}
		}

		// If all mentions were invalid, don't process the message further
		// This prevents agents from responding to invalid @mentions
		if allowMentionValidationErrors && hasInvalidMentions && len(agentIDs) == 0 {
			// Set mentions to a dummy value so agents will see HasMentions() = true
			// but IsMentioned(agentID) = false, preventing all agents from responding
			msg.Mentions = []string{"__INVALID__"}

			// Store the message for history so user can see what they typed
			h.mu.Lock()
			h.appendChannelMessageLocked(msg.Channel, msg)
			h.mu.Unlock()
			return nil
		}
	}

	// Check if it's a command - process commands from both chat and question types
	if h.commandHandler != nil && len(msg.Content) > 0 && msg.Content[0] == '/' {
		// Process command (unlock mutex for command processing)
		ctx := context.Background()
		response, err := h.commandHandler.ProcessCommand(ctx, msg)
		if err != nil {
			return fmt.Errorf("command processing error: %w", err)
		}

		// If command was processed, send the response instead
		if response != nil {
			msg = response
			// Re-attach collaboration snapshot when the handler returns a new message
			// (e.g. /cancel-plan) so metadata includes collaboration_data for the UI.
			h.attachCollaborationData(msg)
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

	// Ephemeral message types -- broadcast to subscribers but don't persist in history
	if msg.Type == protocol.MessageTypeStreamDelta || msg.Type == protocol.MessageTypeStreamEnd || msg.Type == protocol.MessageTypeAgentStatus {
		h.broadcast(msg.Channel, msg)
		return nil
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
		h.appendThreadMessageLocked(threadID, msg)

		// Update thread metadata
		h.updateThreadMetadata(threadID, msg)

		// Broadcast to thread subscribers (for thread panel UI updates)
		h.broadcastToThread(threadID, msg)

		// ALSO add to channel message history so agents can see it when polling
		h.appendChannelMessageLocked(msg.Channel, msg)

		// ALSO broadcast to channel subscribers (so agents can see mentions)
		h.broadcast(msg.Channel, msg)
	} else {
		// Regular channel message
		h.appendChannelMessageLocked(msg.Channel, msg)
		h.broadcast(msg.Channel, msg)
	}

	return nil
}

func (h *Hub) shouldParseCollaborationMentions(msg *protocol.Message) bool {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return false
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return false
	}
	if msg.Type != protocol.MessageTypeCollabDiscussion &&
		msg.Type != protocol.MessageTypeChat &&
		msg.Type != protocol.MessageTypeAnswer &&
		msg.Type != protocol.MessageTypeCollabPlan {
		return false
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return false
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return false
	}
	return snapshot.Phase == collaboration.PhasePlanning || snapshot.Phase == collaboration.PhaseReviewing
}

func (h *Hub) maybeExpandCollaborationParticipants(msg *protocol.Message, mentionedAgentIDs []string) {
	if msg == nil || h.collabManager == nil || msg.IsFromSystem() {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" || len(mentionedAgentIDs) == 0 {
		return
	}
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return
	}
	if snapshot.Phase != collaboration.PhasePlanning && snapshot.Phase != collaboration.PhaseReviewing {
		return
	}
	if !h.collabManager.IsParticipant(collabID, msg.From.ID) {
		return
	}

	candidates := make([]string, 0, len(mentionedAgentIDs))
	for _, agentID := range mentionedAgentIDs {
		if agentID == "" || h.collabManager.IsParticipant(collabID, agentID) {
			continue
		}
		candidates = append(candidates, agentID)
	}
	if len(candidates) == 0 {
		return
	}

	added, err := h.collabManager.AddParticipants(collabID, candidates)
	if err != nil || len(added) == 0 {
		return
	}

	for _, participant := range added {
		if err := h.AddAgentToChannel(participant.AgentID, msg.Channel); err != nil {
			log.Printf("[Collaboration] Failed to add %s to channel %s: %v", participant.AgentName, msg.Channel, err)
			continue
		}
		if h.commandHandler != nil {
			if err := h.commandHandler.EnsureAgentSubscribedToChannel(context.Background(), participant.AgentID, msg.Channel); err != nil {
				log.Printf("[Collaboration] Failed to subscribe %s to %s: %v", participant.AgentName, msg.Channel, err)
			}
		}
	}

	if h.commandHandler != nil {
		client := h.NewCollaborationClientAdapter()
		for _, participant := range added {
			h.commandHandler.setCollabClientOnAgent(participant.AgentID, participant.AgentName, client)
		}
	}

	parts := make([]string, 0, len(added))
	for _, participant := range added {
		parts = append(parts, fmt.Sprintf("@%s (%s)", participant.AgentName, participant.Role))
	}
	notice := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("➕ Added to collaboration `%s`: %s", collabID[:8], strings.Join(parts, ", ")),
	)
	notice.SetCollaborationID(collabID)
	notice.SetCollaborationPhase(string(snapshot.Phase))
	if notice.Metadata == nil {
		notice.Metadata = map[string]interface{}{}
	}
	notice.Metadata["collab_internal_event"] = true
	if err := h.SendMessage(notice); err != nil {
		log.Printf("[Collaboration] Failed to broadcast participant add notice: %v", err)
	}
}

func (h *Hub) processCollaborationLifecycle(msg *protocol.Message) {
	if msg == nil {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" || h.collabManager == nil {
		return
	}
	if msg.Metadata != nil {
		if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
			return
		}
	}

	h.maybeIngestPlanArtifact(msg, collabID)
	h.maybeUpdateTaskStatus(msg, collabID)
}

func (h *Hub) maybeIngestPlanArtifact(msg *protocol.Message, collabID string) {
	if msg.IsFromSystem() {
		return
	}
	if phase := msg.GetCollaborationPhase(); phase != "" && phase != string(collaboration.PhasePlanning) {
		return
	}
	if msg.Type != protocol.MessageTypeChat &&
		msg.Type != protocol.MessageTypeAnswer &&
		msg.Type != protocol.MessageTypeCollabDiscussion {
		return
	}

	planContent := collaboration.ExtractPlanFromResponse(msg.Content)
	if strings.TrimSpace(planContent) == "" {
		return
	}

	collabSnapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnapshot == nil {
		return
	}
	if collabSnapshot.Plan != nil && strings.TrimSpace(collabSnapshot.Plan.Content) == strings.TrimSpace(planContent) {
		return
	}

	if err := h.collabManager.UpdateArtifact(collabID, msg.From.ID, msg.From.Name, planContent); err != nil {
		log.Printf("[Collaboration] Failed to auto-update plan artifact for %s: %v", collabID[:8], err)
		return
	}

	updated, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || updated == nil {
		return
	}
	extractedTasks := collaboration.ExtractTasksFromPlan(planContent, updated.Agents)
	if len(extractedTasks) > 0 {
		if err := h.collabManager.SetTasks(collabID, extractedTasks); err != nil {
			log.Printf("[Collaboration] Failed to auto-set tasks for %s: %v", collabID[:8], err)
		}
		updated, _ = h.collabManager.GetCollaborationSnapshot(collabID)
	}

	planMsg := protocol.NewMessage(
		protocol.MessageTypeCollabPlan,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("🧩 Updated collaboration plan (v%d) based on @%s's proposal.",
			func() int {
				if updated != nil && updated.Plan != nil {
					return updated.Plan.Version
				}
				return 0
			}(),
			msg.From.Name,
		),
	)
	planMsg.SetCollaborationID(collabID)
	planMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	planMsg.SetArtifactAction("edit")
	if planMsg.Metadata == nil {
		planMsg.Metadata = map[string]interface{}{}
	}
	planMsg.Metadata["collab_internal_event"] = true
	if err := h.SendMessage(planMsg); err != nil {
		log.Printf("[Collaboration] Failed to broadcast plan update message: %v", err)
	}
}

func (h *Hub) maybeUpdateTaskStatus(msg *protocol.Message, collabID string) {
	// Task-assignment prompts carry task_id/task_status for routing; they are not
	// assignee status reports. Treating them here spuriously "updates" tasks to
	// pending and broadcasts duplicate collab_status noise.
	if msg.Type == protocol.MessageTypeCollabTask {
		return
	}

	taskID := msg.GetTaskID()
	if taskID == "" {
		return
	}

	collabSnapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || collabSnapshot == nil {
		return
	}

	var task *collaboration.CollaborationTask
	for i := range collabSnapshot.Tasks {
		if collabSnapshot.Tasks[i].ID == taskID {
			task = &collabSnapshot.Tasks[i]
			break
		}
	}
	if task == nil {
		return
	}

	status, ok := normalizeTaskStatus(msg.GetTaskStatus())
	if !ok {
		inferred := inferTaskStatusFromContent(msg.Content)
		if inferred != "" {
			status = inferred
			ok = true
		} else if task.Status == collaboration.TaskPending && msg.From.ID == task.AssignedTo && !msg.IsFromSystem() {
			status = collaboration.TaskInProgress
			ok = true
		}
	}
	if !ok {
		return
	}

	output := strings.TrimSpace(msg.GetTaskOutput())
	if output == "" && (status == collaboration.TaskCompleted || status == collaboration.TaskBlocked) {
		output = strings.TrimSpace(msg.Content)
	}
	if err := h.collabManager.UpdateTaskStatus(collabID, taskID, status, output); err != nil {
		log.Printf("[Collaboration] Failed to update task %s in %s: %v", taskID, collabID[:8], err)
		return
	}

	updated, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err == nil && updated != nil {
		for i := range updated.Tasks {
			if updated.Tasks[i].ID == taskID {
				task = &updated.Tasks[i]
				break
			}
		}
	}

	statusMsg := protocol.NewMessage(
		protocol.MessageTypeCollabStatus,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("📌 Task update: **%s** is now **%s** (assigned to @%s).",
			func() string {
				if task != nil {
					return task.Title
				}
				return taskID
			}(),
			status,
			func() string {
				if task != nil && task.AssignedName != "" {
					return task.AssignedName
				}
				return "unknown"
			}(),
		),
	)
	statusMsg.SetCollaborationID(collabID)
	statusMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
	statusMsg.SetTaskID(taskID)
	statusMsg.SetTaskStatus(string(status))
	if output != "" {
		statusMsg.SetTaskOutput(output)
	}
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = map[string]interface{}{}
	}
	statusMsg.Metadata["collab_internal_event"] = true
	if err := h.SendMessage(statusMsg); err != nil {
		log.Printf("[Collaboration] Failed to broadcast task status update message: %v", err)
	}

	if h.collabManager.AllTasksComplete(collabID) {
		if _, err := h.collabManager.CompleteCollaboration(collabID); err != nil {
			log.Printf("[Collaboration] Failed to complete collaboration %s: %v", collabID[:8], err)
			return
		}
		completedMsg := protocol.NewMessage(
			protocol.MessageTypeCollabStatus,
			msg.Channel,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("✅ Collaboration `%s` completed. All tasks are done.", collabID[:8]),
		)
		completedMsg.SetCollaborationID(collabID)
		completedMsg.SetCollaborationPhase(string(collaboration.PhaseCompleted))
		if completedMsg.Metadata == nil {
			completedMsg.Metadata = map[string]interface{}{}
		}
		completedMsg.Metadata["collab_internal_event"] = true
		if err := h.SendMessage(completedMsg); err != nil {
			log.Printf("[Collaboration] Failed to broadcast collaboration completion message: %v", err)
		}
	}
}

func normalizeTaskStatus(raw string) (collaboration.TaskStatus, bool) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(collaboration.TaskPending):
		return collaboration.TaskPending, true
	case string(collaboration.TaskInProgress):
		return collaboration.TaskInProgress, true
	case string(collaboration.TaskCompleted):
		return collaboration.TaskCompleted, true
	case string(collaboration.TaskBlocked):
		return collaboration.TaskBlocked, true
	default:
		return "", false
	}
}

func inferTaskStatusFromContent(content string) collaboration.TaskStatus {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "blocked") || strings.Contains(lower, "stuck") || strings.Contains(lower, "cannot proceed") {
		return collaboration.TaskBlocked
	}
	if strings.Contains(lower, "completed") ||
		strings.Contains(lower, "done") ||
		strings.Contains(lower, "finished") ||
		strings.Contains(lower, "implemented") {
		return collaboration.TaskCompleted
	}
	if strings.Contains(lower, "working on") || strings.Contains(lower, "in progress") || strings.Contains(lower, "started") {
		return collaboration.TaskInProgress
	}
	return ""
}

func (h *Hub) attachCollaborationData(msg *protocol.Message) {
	if msg == nil || h.collabManager == nil {
		return
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return
	}
	_, _ = h.collabManager.EnsureExecutionTasks(collabID)
	snapshot, err := h.collabManager.GetCollaborationSnapshot(collabID)
	if err != nil || snapshot == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["collaboration_data"] = snapshot.ToUIPayload()
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
	return h.ResolveMentionsWithValidation(mentions, resolved, "")
}

// ResolveMentionsWithValidation converts @mention strings to agent IDs and tracks which were resolved.
// When scopeChannel is non-empty, @here / @channel / @everyone resolve to every agent currently in that channel.
func (h *Hub) ResolveMentionsWithValidation(mentions []string, resolvedMap map[string]bool, scopeChannel string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	seen := make(map[string]bool)
	var agentIDs []string

	for _, mention := range mentions {
		mentionLower := strings.ToLower(mention)
		found := false

		if scopeChannel != "" && (mentionLower == "here" || mentionLower == "channel" || mentionLower == "everyone") {
			if ch, ok := h.channels[scopeChannel]; ok {
				for _, agent := range ch.Agents {
					if !seen[agent.ID] {
						agentIDs = append(agentIDs, agent.ID)
						seen[agent.ID] = true
						found = true
					}
				}
			}
			if resolvedMap != nil {
				resolvedMap[mention] = found
			}
			continue
		}

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

	ch := make(chan *protocol.Message, 512)
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

	dropped := 0
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("[Hub] broadcast: dropped %d/%d messages on channel %q (subscriber buffer full)", dropped, len(subs), channelName)
	}
}

// BroadcastDirect sends a message to all subscribers of a channel without
// storing it in message history. Used for ephemeral messages like stream
// deltas that should reach the frontend but not pollute the history.
func (h *Hub) BroadcastDirect(channelName string, msg *protocol.Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.broadcast(channelName, msg)
	if msg != nil && msg.IsInThread() {
		if threadID := msg.GetThreadID(); threadID != "" {
			h.broadcastToThread(threadID, msg)
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

// FindLiveAgentByDisplayName returns a copy of a registered agent matching
// name (case-insensitive). When typ is non-empty, the agent type must match.
func (h *Hub) FindLiveAgentByDisplayName(name string, typ protocol.AgentType) *protocol.AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return nil
	}
	for _, a := range h.agents {
		if a == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(a.Name)) != want {
			continue
		}
		if typ != "" && a.Type != typ {
			continue
		}
		cp := *a
		return &cp
	}
	return nil
}

// SetAgentCustomRulesMarkdown updates persisted per-agent instructions (markdown).
func (h *Hub) SetAgentCustomRulesMarkdown(agentID, markdown string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ag, ok := h.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}
	ag.CustomRulesMarkdown = strings.TrimSpace(markdown)
	if h.agentRulesStore != nil {
		if err := h.agentRulesStore.Set(agentID, ag.CustomRulesMarkdown); err != nil {
			return err
		}
	}
	h.syncAgentInfoCopiesInChannelsLocked(agentID, ag)
	return nil
}

// syncAgentInfoCopiesInChannelsLocked updates channel member snapshots when AgentInfo mutates.
// Caller must hold h.mu write lock.
func (h *Hub) syncAgentInfoCopiesInChannelsLocked(agentID string, ag *protocol.AgentInfo) {
	for _, ch := range h.channels {
		for i := range ch.Agents {
			if ch.Agents[i].ID == agentID {
				ch.Agents[i] = *ag
			}
		}
	}
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
			Channel:       msg.Channel,
			ReplyCount:    0,
			LastReplyTime: time.Time{},
			Participants:  []string{},
		}
		h.threadMetadata[threadID] = metadata
	}

	if metadata.Channel == "" && msg.Channel != "" {
		metadata.Channel = msg.Channel
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

// CreateDMChannel creates (or returns an existing) DM channel between a user and an agent.
// The agent is automatically joined to the channel.
func (h *Hub) CreateDMChannel(username, agentID string) (*protocol.Channel, error) {
	agent, err := h.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	dmName := fmt.Sprintf("dm-%s-%s", strings.ToLower(username), strings.ToLower(agent.Name))

	// Check if it already exists
	h.mu.RLock()
	if existing, ok := h.channels[dmName]; ok {
		h.mu.RUnlock()
		// Channel was restored or left over from a prior session — still join this
		// agent so channel.Agents, GetAgentChannels, and DM rebind logic stay correct.
		if err := h.JoinChannel(agentID, dmName); err != nil {
			return nil, fmt.Errorf("failed to join agent to existing DM %s: %w", dmName, err)
		}
		return existing, nil
	}
	h.mu.RUnlock()

	ch := h.CreateChannelWithType(
		dmName,
		fmt.Sprintf("Direct message with %s", agent.Name),
		"",
		protocol.ChannelTypeDM,
		username,
	)

	// Auto-join the agent to the DM channel
	if err := h.JoinChannel(agentID, dmName); err != nil {
		return nil, fmt.Errorf("failed to join agent to DM channel: %w", err)
	}

	return ch, nil
}

// GetAgentChannels returns the names of all channels an agent is currently in
func (h *Hub) GetAgentChannels(agentID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var channels []string
	for name, channel := range h.channels {
		for _, a := range channel.Agents {
			if a.ID == agentID {
				channels = append(channels, name)
				break
			}
		}
	}
	return channels
}

// AddAgentToChannel joins an agent to a channel and records it as an explicit member
func (h *Hub) AddAgentToChannel(agentID, channelName string) error {
	if err := h.JoinChannel(agentID, channelName); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return nil
	}

	// Track as explicit member (deduplicated)
	for _, m := range ch.Members {
		if m == agentID {
			return nil
		}
	}
	ch.Members = append(ch.Members, agentID)
	return nil
}

// RemoveAgentFromChannel removes an agent from a channel and its member list
func (h *Hub) RemoveAgentFromChannel(agentID, channelName string) error {
	if err := h.LeaveChannel(agentID, channelName); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return nil
	}

	// Remove from explicit members
	for i, m := range ch.Members {
		if m == agentID {
			ch.Members = append(ch.Members[:i], ch.Members[i+1:]...)
			break
		}
	}
	return nil
}

// DeleteChannel removes a channel entirely. The general channel, public rooms, and DM
// channels cannot be deleted; custom and collaboration channels may be removed.
func (h *Hub) DeleteChannel(channelName string) error {
	if channelName == "general" {
		return fmt.Errorf("cannot delete the general channel")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}

	effectiveType := inferChannelTypeForName(ch.Name, ch.Type)
	switch effectiveType {
	case protocol.ChannelTypePublic, protocol.ChannelTypeDM:
		return fmt.Errorf("cannot delete channel %q (type %s); only custom and collaboration channels can be deleted", channelName, effectiveType)
	}

	// Close all subscriber channels
	for _, sub := range h.subscribers[channelName] {
		close(sub)
	}

	delete(h.channels, channelName)
	delete(h.messages, channelName)
	delete(h.subscribers, channelName)

	return nil
}

// GetChannelType returns the type of the named channel
func (h *Hub) GetChannelType(channelName string) protocol.ChannelType {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if ch, ok := h.channels[channelName]; ok {
		return ch.Type
	}
	return protocol.ChannelTypePublic
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
	h.appendChannelMessageLocked(originalMsg.Channel, feedbackMsg)
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
			h.appendChannelMessageLocked(originalMsg.Channel, errorMsg)
			h.broadcast(originalMsg.Channel, errorMsg)
			h.mu.Unlock()

			// Remove from pending reviews
			h.commandHandler.RemovePendingReview(repoPath)
		} else if response != nil {
			// Send the response message
			h.mu.Lock()
			h.appendChannelMessageLocked(originalMsg.Channel, response)
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

// GetToolApprovalManager returns the tool approval manager for external access
func (h *Hub) GetToolApprovalManager() *ToolApprovalManager {
	return h.toolApprovalManager
}

// GetCollaborationManager returns the collaboration manager
func (h *Hub) GetCollaborationManager() *collaboration.CollaborationManager {
	return h.collabManager
}

// ListCollaborationSnapshots returns collaboration snapshots suitable for UI
// consumption. Data is deep-copied by the collaboration manager.
func (h *Hub) ListCollaborationSnapshots(channel string, includeTerminal bool) []*collaboration.Collaboration {
	if h.collabManager == nil {
		return []*collaboration.Collaboration{}
	}
	snaps, healFlags := h.collabManager.ListSnapshots(channel, includeTerminal)
	for i, snap := range snaps {
		if snap == nil {
			continue
		}
		if i < len(healFlags) && healFlags[i] &&
			snap.Phase == collaboration.PhaseExecuting && len(snap.Tasks) > 0 &&
			!snap.TasksDispatched &&
			h.CollaborationCanDispatchTasks(snap) {
			h.dispatchCollabTaskMessages(snap, nil, false)
		}
	}
	return snaps
}

// RedispatchOpenCollaborationTasksAfterSessionRestore re-sends collaboration_task
// prompts for executing collaborations that still have open work. Session restore
// reloads tasks and assignees intact, so EnsureExecutionTasks usually returns false
// and ListCollaborationSnapshots does not redispatch; agent runtimes still need a
// fresh task message to continue (same effect as /resume-plan while executing).
func (h *Hub) RedispatchOpenCollaborationTasksAfterSessionRestore() {
	if h.collabManager == nil {
		return
	}
	h.collabManager.ReconcileRestoredAgentIDs()
	open := func(t collaboration.CollaborationTask) bool {
		return t.Status == collaboration.TaskPending ||
			t.Status == collaboration.TaskInProgress ||
			t.Status == collaboration.TaskBlocked
	}
	for _, c := range h.collabManager.ListActive() {
		if c == nil || c.Phase != collaboration.PhaseExecuting {
			continue
		}
		if _, err := h.collabManager.EnsureExecutionTasks(c.ID); err != nil {
			log.Printf("[Collaboration] session-restore redispatch EnsureExecutionTasks for %s: %v", c.ID[:8], err)
			continue
		}
		snap, err := h.collabManager.GetCollaborationSnapshot(c.ID)
		if err != nil || snap == nil {
			continue
		}
		n := 0
		for _, t := range snap.Tasks {
			if open(t) {
				n++
			}
		}
		if n == 0 {
			continue
		}
		h.dispatchCollabTaskMessagesFilter(snap, nil, open, true)
		log.Printf("[Collaboration] Session restore: re-sent %d open task prompt(s) for executing collaboration %s", n, c.ID[:8])
	}
}

// dispatchCollabTaskMessages sends collaboration_task messages so assignees
// receive task_assigned_to metadata (mirrors /approve-plan). Used after the
// manager heals missing assignees on executing collaborations.
func (h *Hub) dispatchCollabTaskMessages(snap *collaboration.Collaboration, inheritFrom *protocol.Message, forceRedispatch bool) {
	h.dispatchCollabTaskMessagesFilter(snap, inheritFrom, nil, forceRedispatch)
}

// dispatchCollabTaskMessagesFilter sends collaboration_task messages for tasks
// where include returns true. A nil include sends every task.
// When forceRedispatch is false and tasks were already dispatched, this is a no-op.
func (h *Hub) dispatchCollabTaskMessagesFilter(snap *collaboration.Collaboration, inheritFrom *protocol.Message, include func(collaboration.CollaborationTask) bool, forceRedispatch bool) {
	if snap == nil || snap.Phase != collaboration.PhaseExecuting || len(snap.Tasks) == 0 {
		return
	}
	if !h.CollaborationCanDispatchTasks(snap) {
		return
	}
	collabID := snap.ID
	if !forceRedispatch && h.collabManager != nil {
		fresh, err := h.collabManager.GetCollaborationSnapshot(collabID)
		if err == nil && fresh != nil && fresh.TasksDispatched {
			return
		}
	}
	ch := snap.Channel
	sent := 0
	for _, task := range snap.Tasks {
		if include != nil && !include(task) {
			continue
		}
		mentionName := task.AssignedName
		if mentionName == "" {
			mentionName = "team"
		}
		taskMsg := protocol.NewMessage(
			protocol.MessageTypeCollabTask,
			ch,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("@%s -- Your assigned task:\n\n**%s**\n\n%s\n\nPlease complete this task. You can @mention other collaboration participants if you need their input.",
				mentionName, task.Title, task.Description),
		)
		taskMsg.SetCollaborationID(collabID)
		taskMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
		taskMsg.SetTaskID(task.ID)
		taskMsg.SetTaskStatus(string(task.Status))
		if ws := collaborationWorkspaceContextSnapshot(snap); ws != nil {
			if taskMsg.Metadata == nil {
				taskMsg.Metadata = map[string]interface{}{}
			}
			taskMsg.Metadata["workspace_context"] = ws
		}
		if task.AssignedTo != "" {
			taskMsg.Mentions = []string{task.AssignedTo}
			if taskMsg.Metadata == nil {
				taskMsg.Metadata = map[string]interface{}{}
			}
			taskMsg.Metadata["task_assigned_to"] = task.AssignedTo
		}
		if err := h.SendMessage(taskMsg); err != nil {
			log.Printf("[Collaboration] Failed to send task message (redispatch): %v", err)
			continue
		}
		sent++
	}
	if sent > 0 && h.collabManager != nil {
		if err := h.collabManager.MarkTasksDispatched(collabID); err != nil {
			log.Printf("[Collaboration] MarkTasksDispatched %s: %v", collabID[:8], err)
		}
	}
}

// NewCollaborationClientAdapter creates an adapter that implements
// agent.CollaborationClient by delegating to the real CollaborationManager.
func (h *Hub) NewCollaborationClientAdapter() agent.CollaborationClient {
	return &collabClientAdapter{cm: h.collabManager}
}

// collabClientAdapter bridges the agent.CollaborationClient interface
// to the concrete collaboration.CollaborationManager.
type collabClientAdapter struct {
	cm *collaboration.CollaborationManager
}

func (a *collabClientAdapter) IsParticipant(collabID, agentID string) bool {
	return a.cm.IsParticipant(collabID, agentID)
}

func (a *collabClientAdapter) IsAgentTurn(collabID, agentID string) bool {
	return a.cm.IsAgentTurn(collabID, agentID)
}

func (a *collabClientAdapter) IsActive(collabID string) bool {
	return a.cm.IsActive(collabID)
}

func (a *collabClientAdapter) AgentOutOfTurnMentionAllowed(collabID string) bool {
	return a.cm.AgentOutOfTurnMentionAllowed(collabID)
}

func (a *collabClientAdapter) GetCurrentTurnAgent(collabID string) (string, error) {
	return a.cm.GetCurrentTurnAgent(collabID)
}

func (a *collabClientAdapter) GetCollaborationForAgent(agentID string) agent.CollaborationInfo {
	c := a.cm.GetCollaborationForAgent(agentID)
	if c == nil {
		return agent.CollaborationInfo{}
	}

	agentRole := ""
	for _, ag := range c.Agents {
		if ag.AgentID == agentID {
			agentRole = ag.Role
			break
		}
	}

	agents := make([]agent.CollaborationAgentSummary, 0, len(c.Agents))
	for _, ag := range c.Agents {
		agents = append(agents, agent.CollaborationAgentSummary{
			Name:      ag.AgentName,
			Type:      string(ag.AgentType),
			Role:      ag.Role,
			Expertise: ag.Expertise,
		})
	}

	planContent := ""
	planVersion := 0
	if c.Plan != nil {
		planContent = c.Plan.Content
		planVersion = c.Plan.Version
	}

	return agent.CollaborationInfo{
		ID:               c.ID,
		Description:      c.Description,
		Phase:            string(c.Phase),
		PlanContent:      planContent,
		PlanVersion:      planVersion,
		AgentRole:        agentRole,
		Agents:           agents,
		Channel:          c.Channel,
		WorkingDirectory: c.WorkingDirectory,
	}
}

func (a *collabClientAdapter) GetCollaborationWorkingDirectory(collabID string) string {
	if collabID == "" {
		return ""
	}
	c, err := a.cm.GetCollaboration(collabID)
	if err != nil || c == nil {
		return ""
	}
	return c.WorkingDirectory
}

func (a *collabClientAdapter) RecordMessage(collabID string, msg *protocol.Message) error {
	return a.cm.RecordMessage(collabID, msg)
}

func (a *collabClientAdapter) AnalyzeConsensus(collabID string, msg *protocol.Message) string {
	return string(a.cm.AnalyzeConsensus(collabID, msg))
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

	// Resolve workspace root from message context. No default workspace fallback is allowed.
	wsRoot := h.resolveWorkspaceRoot(msg)
	if wsRoot == "" {
		log.Printf("[FileChange] Missing workspace context for proposal from %s on channel %s",
			msg.From.Name, msg.Channel)
		return
	}

	// Resolve file path against workspace
	filePath, err := h.resolveWorkspacePath(proposal.FilePath, wsRoot)
	if err != nil {
		log.Printf("[FileChange] Failed to resolve file path %q: %v", proposal.FilePath, err)
		return
	}

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
		oldPath, err = h.resolveWorkspacePath(proposal.OldPath, wsRoot)
		if err != nil {
			log.Printf("[FileChange] Failed to resolve move old path %q: %v", proposal.OldPath, err)
			return
		}
		newPath, err = h.resolveWorkspacePath(proposal.NewPath, wsRoot)
		if err != nil {
			log.Printf("[FileChange] Failed to resolve move new path %q: %v", proposal.NewPath, err)
			return
		}
	}

	// Bind executor to the resolved workspace root from the message context.
	h.fileChangeManager.GetExecutor().SetWorkspaceRoot(wsRoot)

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

// resolveWorkspacePath resolves a potentially relative file path against the provided workspace root.
func (h *Hub) resolveWorkspacePath(filePath, workspaceRoot string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("empty file path")
	}
	if workspaceRoot == "" {
		return "", fmt.Errorf("missing workspace root for path: %s", filePath)
	}
	var candidate string
	if filepath.IsAbs(filePath) {
		candidate = filePath
	} else {
		candidate = filepath.Join(workspaceRoot, filePath)
	}
	return pathutil.WithinRoot(workspaceRoot, candidate)
}

// resolveWorkspaceRoot returns the workspace root path from message context only.
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

	return ""
}

// MetadataKeyHistoryResync is set on ephemeral agent_status messages after channel
// history pruning so clients and agents refetch from the hub.
const MetadataKeyHistoryResync = "history_resync"

// PruneMessagesOlderThan removes channel and thread messages whose Timestamp is
// strictly before time.Now()-maxAge. Empty threads are deleted with metadata.
// It broadcasts an ephemeral agent_status per affected channel (see MetadataKeyHistoryResync).
// Returns the number of messages dropped from storage.
func (h *Hub) PruneMessagesOlderThan(maxAge time.Duration) (removed int) {
	if maxAge <= 0 {
		return 0
	}
	cutoff := time.Now().Add(-maxAge)
	affectedChannels := make(map[string]struct{})
	removedIDs := make(map[string]struct{})

	h.mu.Lock()

	for chName, msgs := range h.messages {
		next := make([]*protocol.Message, 0, len(msgs))
		for _, m := range msgs {
			if m == nil {
				continue
			}
			if m.Timestamp.Before(cutoff) {
				if m.ID != "" {
					if _, seen := removedIDs[m.ID]; !seen {
						removedIDs[m.ID] = struct{}{}
						removed++
					}
				} else {
					removed++
				}
				continue
			}
			next = append(next, m)
		}
		if len(next) != len(msgs) {
			h.messages[chName] = next
			affectedChannels[chName] = struct{}{}
		}
	}

	for threadID, msgs := range h.threads {
		next := make([]*protocol.Message, 0, len(msgs))
		for _, m := range msgs {
			if m == nil {
				continue
			}
			if m.Timestamp.Before(cutoff) {
				if m.ID != "" {
					if _, seen := removedIDs[m.ID]; !seen {
						removedIDs[m.ID] = struct{}{}
						removed++
					}
				} else {
					removed++
				}
				continue
			}
			next = append(next, m)
		}
		if len(next) == 0 {
			delete(h.threads, threadID)
			delete(h.threadMetadata, threadID)
			delete(h.threadParentAuthors, threadID)
		} else if len(next) != len(msgs) {
			h.threads[threadID] = next
		}
	}

	h.mu.Unlock()

	systemFrom := protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral}
	for chName := range affectedChannels {
		resync := protocol.NewMessage(protocol.MessageTypeAgentStatus, chName, systemFrom, "")
		resync.Metadata[MetadataKeyHistoryResync] = true
		h.BroadcastDirect(chName, resync)
	}

	return removed
}

// --- Session Recording ---

// SessionSnapshot captures the full state of a chat session for debugging/review.
type SessionSnapshot struct {
	SavedAt        time.Time                               `json:"saved_at"`
	Channels       map[string]*ChannelSnapshot             `json:"channels"`
	Threads        map[string]*ThreadSnapshot              `json:"threads"`
	Agents         []*protocol.AgentInfo                   `json:"agents"`
	Collaborations map[string]*collaboration.Collaboration `json:"collaborations,omitempty"`
}

// SessionSaveHealth tracks snapshot save freshness and failures.
type SessionSaveHealth struct {
	LastSavedAt    time.Time `json:"last_saved_at,omitempty"`
	LastFailureAt  time.Time `json:"last_failure_at,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	LastSizeBytes  int       `json:"last_size_bytes,omitempty"`
	LastDurationMs int64     `json:"last_duration_ms,omitempty"`
	SaveCount      int64     `json:"save_count"`
	FailureCount   int64     `json:"failure_count"`
	LastPath       string    `json:"last_path,omitempty"`
}

// ChannelSnapshot holds all messages for a single channel.
type ChannelSnapshot struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Type        protocol.ChannelType `json:"type"`
	CreatedBy   string               `json:"created_by,omitempty"`
	Members     []string             `json:"members,omitempty"`
	Messages    []*protocol.Message  `json:"messages"`
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
		SavedAt:        time.Now(),
		Channels:       make(map[string]*ChannelSnapshot),
		Threads:        make(map[string]*ThreadSnapshot),
		Agents:         make([]*protocol.AgentInfo, 0),
		Collaborations: make(map[string]*collaboration.Collaboration),
	}

	// Capture channel messages
	for name, ch := range h.channels {
		cs := &ChannelSnapshot{
			Name:        ch.Name,
			Description: ch.Description,
			Type:        ch.Type,
			CreatedBy:   ch.CreatedBy,
			Members:     ch.Members,
			Messages:    make([]*protocol.Message, 0),
		}
		if msgs, ok := h.messages[name]; ok {
			for _, m := range msgs {
				if m.Type == protocol.MessageTypeAgentStatus ||
					m.Type == protocol.MessageTypeStreamDelta ||
					m.Type == protocol.MessageTypeStreamEnd {
					continue
				}
				cs.Messages = append(cs.Messages, m)
			}
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
	if h.collabManager != nil {
		snapshot.Collaborations = h.collabManager.Snapshot()
	}

	return snapshot
}

// LoadSessionFromFile restores channels/messages/threads and collaboration
// state from a previous snapshot. It is safe to call on startup.
func (h *Hub) LoadSessionFromFile(path string) error {
	load, _, statErr := sessionFileReadyToLoad(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return nil
		}
		return fmt.Errorf("failed stat session file: %w", statErr)
	}
	if !load {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed reading session file: %w", err)
	}

	var snapshot SessionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		if archErr := archiveUnusableSessionFile(path, "corrupt JSON"); archErr != nil {
			return fmt.Errorf("failed to unmarshal session file: %w (archive failed: %v)", err, archErr)
		}
		return nil
	}

	h.mu.Lock()
	h.channels = make(map[string]*protocol.Channel)
	h.messages = make(map[string][]*protocol.Message)
	h.threads = make(map[string][]*protocol.Message)
	h.threadMetadata = make(map[string]*protocol.ThreadMetadata)
	h.subscribers = make(map[string][]chan *protocol.Message)
	h.threadSubscribers = make(map[string][]chan *protocol.Message)

	for name, ch := range snapshot.Channels {
		if ch == nil {
			continue
		}
		channel := &protocol.Channel{
			ID:          uuid.New().String(),
			Name:        ch.Name,
			Description: ch.Description,
			Type:        inferChannelTypeForName(ch.Name, ch.Type),
			CreatedBy:   ch.CreatedBy,
			Created:     snapshot.SavedAt,
			Agents:      []protocol.AgentInfo{},
			Members:     append([]string(nil), ch.Members...),
			Tags:        []string{},
		}
		h.channels[name] = channel
		h.messages[name] = append([]*protocol.Message(nil), ch.Messages...)
		h.subscribers[name] = []chan *protocol.Message{}
	}
	if len(h.channels) == 0 {
		channel := &protocol.Channel{
			ID:          uuid.New().String(),
			Name:        "general",
			Description: "General discussion",
			Type:        protocol.ChannelTypePublic,
			CreatedBy:   "system",
			Created:     time.Now(),
			Agents:      []protocol.AgentInfo{},
			Members:     []string{},
			Tags:        []string{},
		}
		h.channels[channel.Name] = channel
		h.messages[channel.Name] = []*protocol.Message{}
		h.subscribers[channel.Name] = []chan *protocol.Message{}
	}
	for threadID, thread := range snapshot.Threads {
		if thread == nil {
			continue
		}
		h.threads[threadID] = append([]*protocol.Message(nil), thread.Messages...)
		if thread.Metadata != nil {
			h.threadMetadata[threadID] = thread.Metadata
		}
		h.threadSubscribers[threadID] = []chan *protocol.Message{}
	}
	h.repairChannelTypesLocked()
	h.hydrateCollabChannelsFromCollaborationsLocked(snapshot.Collaborations)
	h.trimAllChannelAndThreadHistoryLocked()
	h.mu.Unlock()

	if h.collabManager != nil && snapshot.Collaborations != nil {
		h.collabManager.Restore(snapshot.Collaborations)
		pruned := h.collabManager.PruneTerminalCollaborations(maxPersistTerminalCollaborations)
		if pruned > 0 {
			log.Printf("💾 Pruned %d terminal collaboration(s) from memory after restore", pruned)
		}
	}

	log.Printf("💾 Session restored from %s (%d channels, %d threads, %d collaborations)",
		path, len(snapshot.Channels), len(snapshot.Threads), len(snapshot.Collaborations))
	return nil
}

// SaveSessionToFile writes the current session snapshot to a JSON file.
// It writes to a temp file first, then renames for atomic replacement.
func (h *Hub) SaveSessionToFile(path string) error {
	h.sessionSaveMu.Lock()
	defer h.sessionSaveMu.Unlock()
	startedAt := time.Now()
	snapshot := h.TakeSessionSnapshot()
	prepareSessionSnapshotForPersist(snapshot)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write to a unique temp file, fsync it, then rename atomically.
	tmpFile, err := os.CreateTemp(dir, "last-session-*.tmp")
	if err != nil {
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to create temp session file: %w", err)
	}
	tmpPath := tmpFile.Name()
	enc := json.NewEncoder(tmpFile)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snapshot); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to encode session file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to sync session file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to close session file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to set session file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		h.recordSessionSaveFailure(path, startedAt, err)
		return fmt.Errorf("failed to finalize session file: %w", err)
	}
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	written := 0
	if fi, err := os.Stat(path); err == nil {
		written = int(fi.Size())
	}
	h.recordSessionSaveSuccess(path, startedAt, written)
	log.Printf("💾 Session saved to %s (%d bytes)", path, written)
	return nil
}

func (h *Hub) recordSessionSaveSuccess(path string, startedAt time.Time, size int) {
	h.sessionHealthMu.Lock()
	defer h.sessionHealthMu.Unlock()
	h.sessionHealth.LastSavedAt = time.Now()
	h.sessionHealth.LastSizeBytes = size
	h.sessionHealth.LastDurationMs = time.Since(startedAt).Milliseconds()
	h.sessionHealth.LastPath = path
	h.sessionHealth.SaveCount++
	h.sessionHealth.LastError = ""
}

func (h *Hub) recordSessionSaveFailure(path string, startedAt time.Time, err error) {
	h.sessionHealthMu.Lock()
	defer h.sessionHealthMu.Unlock()
	h.sessionHealth.LastFailureAt = time.Now()
	h.sessionHealth.LastDurationMs = time.Since(startedAt).Milliseconds()
	h.sessionHealth.LastPath = path
	h.sessionHealth.FailureCount++
	h.sessionHealth.LastError = err.Error()
}

// GetSessionSaveHealth returns the latest session save diagnostics including
// freshness (age in seconds) for observability endpoints.
func (h *Hub) GetSessionSaveHealth() map[string]interface{} {
	h.sessionHealthMu.RLock()
	health := h.sessionHealth
	h.sessionHealthMu.RUnlock()

	ageSeconds := int64(-1)
	if !health.LastSavedAt.IsZero() {
		ageSeconds = int64(time.Since(health.LastSavedAt).Seconds())
	}
	return map[string]interface{}{
		"last_saved_at":    health.LastSavedAt,
		"last_failure_at":  health.LastFailureAt,
		"last_error":       health.LastError,
		"last_size_bytes":  health.LastSizeBytes,
		"last_duration_ms": health.LastDurationMs,
		"save_count":       health.SaveCount,
		"failure_count":    health.FailureCount,
		"last_path":        health.LastPath,
		"age_seconds":      ageSeconds,
	}
}

// DefaultSessionPath returns the default path for the last session file.
func DefaultSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".neural-junkie", "last-session.json")
}
