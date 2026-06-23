package hub

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/collaboration/actions"
	"github.com/camronwood/neural-junkie/internal/filechange"
	"github.com/camronwood/neural-junkie/internal/hub/gitchange"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
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
	gitChangeManager  *gitchange.Manager
	fileChangeBackendFn func(workspaceRoot string) filechange.WorkspaceIO

	// Workspace manager for handling workspace operations
	workspaceManager *WorkspaceManager

	worktreeBackendFn func(repoPath string) workspacebackend.Backend

	// Tool approval manager for CLI agent tool call approvals
	toolApprovalManager *ToolApprovalManager

	// Collaboration manager for multi-agent collaboration sessions
	collabManager *collaboration.CollaborationManager

	// Per-agent custom rules (markdown), persisted on disk
	agentRulesStore *agent.AgentCustomRulesStorage

	// Global user rules (markdown), persisted on disk keyed by username or default
	userRulesStore *agent.UserRulesStorage

	// Session snapshot save synchronization and observability.
	sessionSaveMu   sync.Mutex
	sessionHealthMu sync.RWMutex
	sessionHealth   SessionSaveHealth

	// Per-channel session summaries (dm/custom); persisted via ChannelSnapshot fields.
	channelContext           map[string]*ChannelContextState
	channelSummaryRefreshGen map[string]uint64
	channelSummaryGen        ChannelSummaryGenerator
	channelSummaryModel      string

	// channelHolds: user interject (Stop) — agents defer new turns until a human message.
	channelHolds map[string]ChannelHold

	persistentStore PersistentMessageStore
	durableChannels map[string]bool

	// Collaboration idle watchdog (in-memory, not persisted).
	collabWatchdogMu           sync.Mutex
	collabWatchdogRedispatch   map[string]int
	collabWatchdogAutoAckTried map[string]bool

	collabActionConfigMu sync.RWMutex
	collabActionConfig   actions.Config

	mu sync.RWMutex
}

// NewHub creates a new chat hub
func NewHub() *Hub {
	hub := &Hub{
		channels:                   make(map[string]*protocol.Channel),
		agents:                     make(map[string]*protocol.AgentInfo),
		messages:                   make(map[string][]*protocol.Message),
		threads:                    make(map[string][]*protocol.Message),
		threadMetadata:             make(map[string]*protocol.ThreadMetadata),
		threadParentAuthors:        make(map[string]string),
		subscribers:                make(map[string][]chan *protocol.Message),
		threadSubscribers:          make(map[string][]chan *protocol.Message),
		removedAgents:              make(map[string]*protocol.AgentInfo),
		channelContext:             make(map[string]*ChannelContextState),
		channelHolds:               make(map[string]ChannelHold),
		collabWatchdogRedispatch:   make(map[string]int),
		collabWatchdogAutoAckTried: make(map[string]bool),
		channelSummaryRefreshGen:   make(map[string]uint64),
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
	hub.gitChangeManager = gitchange.NewManager()

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
	hub.wireCollaborationRecaps()

	rulesStore, err := agent.NewAgentCustomRulesStorage()
	if err != nil {
		log.Printf("Warning: agent custom rules storage unavailable: %v", err)
	} else {
		hub.agentRulesStore = rulesStore
	}

	userRulesStore, err := agent.NewUserRulesStorage()
	if err != nil {
		log.Printf("Warning: user rules storage unavailable: %v", err)
	} else {
		hub.userRulesStore = userRulesStore
	}

	agent.SetUserRulesLookup(func(username string) string {
		if hub.userRulesStore == nil {
			return ""
		}
		return hub.userRulesStore.Resolve(username)
	})

	return hub
}
// GetWorkspaceManager returns the workspace manager
func (h *Hub) GetWorkspaceManager() *WorkspaceManager {
	return h.workspaceManager
}

// RegisterAgent registers a new agent
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

// ResolveUserRulesMarkdown returns persisted global user rules for username (with default fallback).
func (h *Hub) ResolveUserRulesMarkdown(username string) string {
	if h.userRulesStore == nil {
		return ""
	}
	return h.userRulesStore.Resolve(username)
}

// GetUserRulesMarkdown returns rules for the session user (API GET).
func (h *Hub) GetUserRulesMarkdown(username string) string {
	return h.ResolveUserRulesMarkdown(username)
}

// SetUserRulesMarkdown persists global user rules for username (empty username → default key).
func (h *Hub) SetUserRulesMarkdown(username, markdown string) error {
	if h.userRulesStore == nil {
		return fmt.Errorf("user rules storage unavailable")
	}
	return h.userRulesStore.Set(username, markdown)
}

func (h *Hub) collabExecutionTimeoutOverride() int {
	if h.commandHandler == nil || h.commandHandler.appConfig == nil {
		return 0
	}
	return h.commandHandler.appConfig.Collaboration.ExecutionTimeoutSeconds
}

func (h *Hub) collabUserRulesMarkdown(snap *collaboration.Collaboration, inheritFrom *protocol.Message) string {
	username := ""
	if inheritFrom != nil && protocol.IsUserLikeSender(inheritFrom.From) {
		username = inheritFrom.From.Name
	} else if snap != nil {
		username = snap.CreatedBy
	}
	return h.ResolveUserRulesMarkdown(username)
}

// AnnotateInboundUserMessage stamps session identity and injects persisted user rules for human senders.
// sessionUsername is the authenticated hub session name when the message arrived via the API.
func (h *Hub) AnnotateInboundUserMessage(msg *protocol.Message, sessionUsername string) {
	h.annotateInboundUserMessage(msg, sessionUsername)
}

func (h *Hub) annotateInboundUserMessage(msg *protocol.Message, sessionUsername string) {
	if msg == nil || !protocol.IsUserLikeSender(msg.From) {
		return
	}
	if sessionUsername = strings.TrimSpace(sessionUsername); sessionUsername != "" {
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}
		msg.Metadata[agent.MetadataHubSessionUsername] = sessionUsername
	}
	agent.AttachUserRulesMetadataIfMissing(msg)
	h.enrichRemoteWorkspaceMetadata(msg)
}

// syncAgentInfoCopiesInChannelsLocked updates channel member snapshots when AgentInfo mutates.
// Caller must hold h.mu write lock.
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
// shouldAutoCreateRepoAgent determines if we should auto-create a repo agent for this message
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

func (h *Hub) GetGitChangeManager() *gitchange.Manager {
	return h.gitChangeManager
}

// GetToolApprovalManager returns the tool approval manager for external access
func (h *Hub) GetToolApprovalManager() *ToolApprovalManager {
	return h.toolApprovalManager
}

// GetCollaborationManager returns the collaboration manager
func (h *Hub) GetCollaborationManager() *collaboration.CollaborationManager {
	return h.collabManager
}

// SetCollaborationAssetsRootResolver configures where per-collaboration execution
// sandboxes are created (<root>/<collaboration-id>/).
func (h *Hub) SetCollaborationAssetsRootResolver(fn func() string) {
	if h.collabManager != nil {
		h.collabManager.SetAssetsRootResolver(fn)
	}
}

// GetCollaborationAssetsRoot returns the configured collaboration assets parent directory.
func (h *Hub) GetCollaborationAssetsRoot() string {
	if h.collabManager == nil {
		return ""
	}
	dir, err := h.collabManager.CollabAssetsBaseDir()
	if err != nil {
		return ""
	}
	return dir
}

func (h *Hub) ListCollaborationSnapshots(channel string, includeTerminal bool) []*collaboration.Collaboration {
	if h.collabManager == nil {
		return []*collaboration.Collaboration{}
	}
	snaps, healFlags := h.collabManager.ListSnapshots(channel, includeTerminal)
	for i, snap := range snaps {
		if snap == nil {
			continue
		}
		h.annotateCollaborationTaskRouting(snap)
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
// prompts for executing collaborations that still have open work. It also repairs
// review-phase recap prompts that were marked pending before a facilitator was
// assigned. Session restore reloads tasks and assignees intact, so
// EnsureExecutionTasks usually returns false and ListCollaborationSnapshots does
// not redispatch; agent runtimes still need a fresh task message to continue
// (same effect as /resume-plan while executing).
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
		if h.isChannelDurable(chName) {
			continue
		}
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
