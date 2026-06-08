package agent

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/server"
)

const (
	customChannelBroadPromptResponderCap = 2
	customChannelRelevanceMinScore       = 2
	collabTaskMinReplyInterval           = 3 * time.Second
)

// Agent represents an AI agent that can participate in chat rooms
type Agent struct {
	Info    protocol.AgentInfo
	AI      ai.AIProvider
	Hub     HubClient
	Context *ConversationContext

	stopCh chan struct{}
	msgCh  chan *protocol.Message

	// Track messages we've already responded to
	respondedMessages map[string]bool
	respondedMutex    sync.Mutex

	// Vision capability
	SupportsVision bool

	// MCP server for tool execution (optional)
	MCPServer MCPServerInterface

	// Provider switching
	providerMutex sync.RWMutex

	// WorkspacePath is the root directory agents use to resolve relative file paths.
	// Set from WORKSPACE_PATH env var or extracted from workspace context metadata.
	WorkspacePath string

	// Multi-channel support
	activeChannels map[string]context.CancelFunc // channel name -> cancel func for its listener
	channelMu      sync.Mutex
	contextMu      sync.RWMutex // protects Context.History map and slices

	// When true, Start does not run discoverChannels (no polling of Hub.GetAgentChannels).
	// Dedicated DM instances use this so they only listen on channels passed to Start/AddChannel.
	DisableChannelDiscovery bool

	// Collaboration support (set by the hub after creation)
	Collab CollaborationClient

	// collabTaskReplyAt rate-limits responses to collaboration_task prompts.
	collabTaskReplyMu sync.Mutex
	collabTaskReplyAt map[string]time.Time

	// activeGens tracks in-flight generation cancel funcs per channel (interject/stop).
	activeGenMu sync.Mutex
	activeGens  map[string]map[string]context.CancelFunc // channel -> genID -> cancel

	// lastDelegationConsulted holds specialist names consulted on the previous turn (for metadata).
	delegationMu            sync.Mutex
	lastDelegationConsulted []string

	// cadWrittenPaths tracks workspace-relative .scad paths written during the current turn.
	cadWrittenMu    sync.Mutex
	cadWrittenPaths []string

	// Optional pre-processing hook for specialized agents. When set and it
	// returns true, the message is considered fully handled and the base
	// response pipeline is skipped.
	messageInterceptor func(context.Context, *protocol.Message) bool

	// Optional full prompt builder (Assistant uses buildAssistantPrompt).
	customPromptBuilder func(*protocol.Message) string
}

// MCPServerInterface defines the interface for MCP servers
type MCPServerInterface interface {
	GetMCPServer() *server.MCPServer
	Start() error
}

// AIProvider is now defined in the ai package

// HubClient defines the interface for interacting with the chat hub
type HubClient interface {
	SendMessage(msg *protocol.Message) error
	BroadcastDirect(channelName string, msg *protocol.Message)
	Subscribe(channelName string) (chan *protocol.Message, error)
	GetMessages(channelName string, limit int) ([]*protocol.Message, error)
	GetChannelAgents(channelName string) ([]protocol.AgentInfo, error)
	GetThreadParentAuthor(threadID string) string
	GetCommandHandler() CommandHandlerInterface
	GetAgentChannels(agentID string) []string
	GetChannelType(channelName string) protocol.ChannelType
	GetChannelSessionSummary(channel string) string
	GetThreadMessages(threadID string, limit int) ([]*protocol.Message, error)
	// IsChannelHeld is true when the user has interjected (agents should not start new turns).
	IsChannelHeld(channel string) bool
	// Image generation (hub OpenAI Images API when OPENAI_API_KEY is set).
	ImageGenerationEnabled() bool
	GenerateAndPostImage(ctx context.Context, channel string, from protocol.AgentInfo, prompt, size string) error
}

// CollaborationClient is the subset of CollaborationManager that agents
// need to check collaboration state. Defined as an interface here to
// avoid a circular dependency on the collaboration package.
type CollaborationClient interface {
	IsParticipant(collabID, agentID string) bool
	IsAgentTurn(collabID, agentID string) bool
	IsActive(collabID string) bool
	GetCurrentTurnAgent(collabID string) (string, error)
	GetCollaborationForAgent(agentID string) CollaborationInfo
	// GetCollaboration returns state for a specific collaboration the agent participates in.
	GetCollaboration(collabID, agentID string) CollaborationInfo
	// GetCollaborationWorkingDirectory returns the on-disk sandbox for an executing collaboration.
	GetCollaborationWorkingDirectory(collabID string) string
	RecordMessage(collabID string, msg *protocol.Message) error
	AnalyzeConsensus(collabID string, msg *protocol.Message) string
	// AgentOutOfTurnMentionAllowed is false when planning/review discussion
	// has stopped accepting turns (e.g. budget_exhausted).
	AgentOutOfTurnMentionAllowed(collabID string) bool
}

// CollaborationInfo carries the subset of collaboration state an agent
// needs when building prompts and deciding whether to respond.
type CollaborationInfo struct {
	ID               string
	Description      string
	Phase            string
	PlanContent      string
	PlanVersion      int
	AgentRole        string
	Agents           []CollaborationAgentSummary
	Channel          string
	ExecutionMode    string // sandbox | worktree
	SourceRepoPath         string
	SourceWorkspaceContext map[string]interface{}
	WorktreeBranch         string
	WorkingDirectory       string // collaboration execution root (absolute path)
}

// CollaborationAgentSummary describes another agent in a collaboration
// (used for prompt construction without importing the collaboration package).
type CollaborationAgentSummary struct {
	Name      string
	Type      string
	Role      string
	Expertise []string
}

// ExportableAgent interface for agents that can be exported to MCP format
type ExportableAgent interface {
	ExportToMCP() (interface{}, error)
	GetExportMetadata() interface{}
}

// ConversationContext maintains the agent's understanding of conversations
type ConversationContext struct {
	CurrentChannel string
	History        map[string][]*protocol.Message // channel -> messages
	MaxHistory     int
}

// NewAgent creates a new agent
func NewAgent(agentType protocol.AgentType, name string, expertise []string, aiProviderInstance ai.AIProvider, hub HubClient) *Agent {
	// Determine provider type and model
	aiProvider := "claude"
	aiModel := aiProviderInstance.GetModel()

	// Check provider type by checking the provider instance type
	switch aiProviderInstance.(type) {
	case *ai.OllamaProvider:
		aiProvider = "ollama"
	case *ai.LMStudioProvider:
		aiProvider = "lmstudio"
	default:
		// Check if it's an Ollama provider by checking the model name (fallback)
		if strings.Contains(aiModel, "llama") || strings.Contains(aiModel, "mistral") ||
			strings.Contains(aiModel, "phi") || strings.Contains(aiModel, "gemma") ||
			strings.Contains(aiModel, "codellama") {
			aiProvider = "ollama"
		}
	}

	agent := &Agent{
		Info: protocol.AgentInfo{
			ID:         uuid.New().String(),
			Name:       name,
			Type:       agentType,
			Expertise:  expertise,
			Status:     "active",
			Model:      aiProviderInstance.GetModel(),
			AIProvider: aiProvider,
			AIModel:    aiModel,
		},
		AI:  aiProviderInstance,
		Hub: hub,
		Context: &ConversationContext{
			History:    make(map[string][]*protocol.Message),
			MaxHistory: 50,
		},
		stopCh:            make(chan struct{}),
		msgCh:             make(chan *protocol.Message, 100),
		respondedMessages: make(map[string]bool),
		activeChannels:    make(map[string]context.CancelFunc),
		activeGens:        make(map[string]map[string]context.CancelFunc),
		WorkspacePath:     os.Getenv("WORKSPACE_PATH"),
	}

	// Set vision capability in Info
	agent.Info.SupportsVision = agent.SupportsVision
	return agent
}

// NewAgentWithProvider creates a new agent with explicit provider selection
func NewAgentWithProvider(agentType protocol.AgentType, name string, expertise []string, ai ai.AIProvider, hub HubClient, provider string, model string) *Agent {
	agent := &Agent{
		Info: protocol.AgentInfo{
			ID:         uuid.New().String(),
			Name:       name,
			Type:       agentType,
			Expertise:  expertise,
			Status:     "active",
			Model:      model,
			AIProvider: provider,
			AIModel:    model,
		},
		AI:  ai,
		Hub: hub,
		Context: &ConversationContext{
			History:    make(map[string][]*protocol.Message),
			MaxHistory: 50,
		},
		stopCh:            make(chan struct{}),
		msgCh:             make(chan *protocol.Message, 100),
		respondedMessages: make(map[string]bool),
		activeChannels:    make(map[string]context.CancelFunc),
		activeGens:        make(map[string]map[string]context.CancelFunc),
	}

	// Set vision capability in Info
	agent.Info.SupportsVision = agent.SupportsVision
	return agent
}

// Start begins the agent's message processing loop on a single channel
func (a *Agent) Start(ctx context.Context, channel string) error {
	a.Context.CurrentChannel = channel
	if err := a.AddChannel(ctx, channel); err != nil {
		return err
	}

	if !a.DisableChannelDiscovery {
		// Periodic discovery: pick up channels this agent was joined to after Start.
		go a.discoverChannels(ctx)
	}

	return nil
}

// StartMultiChannel starts the agent listening on multiple channels
func (a *Agent) StartMultiChannel(ctx context.Context, channels []string) error {
	if len(channels) == 0 {
		return fmt.Errorf("at least one channel is required")
	}
	a.Context.CurrentChannel = channels[0]

	for _, ch := range channels {
		if err := a.AddChannel(ctx, ch); err != nil {
			log.Printf("[%s] Warning: failed to subscribe to channel %s: %v", a.Info.Name, ch, err)
		}
	}

	if !a.DisableChannelDiscovery {
		go a.discoverChannels(ctx)
	}
	return nil
}

// AddChannel subscribes the agent to an additional channel dynamically
func (a *Agent) AddChannel(ctx context.Context, channel string) error {
	a.channelMu.Lock()
	if cancel, exists := a.activeChannels[channel]; exists {
		cancel()
		delete(a.activeChannels, channel)
	}

	subCh, err := a.Hub.Subscribe(channel)
	if err != nil {
		a.channelMu.Unlock()
		return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	history, err := a.Hub.GetMessages(channel, 20)
	if err == nil {
		a.replaceChannelHistory(channel, history)
	}

	// Listener lifetime must not follow short-lived caller contexts (e.g. HTTP handlers).
	listenerCtx, listenerCancel := context.WithCancel(context.Background())
	a.activeChannels[channel] = listenerCancel
	a.channelMu.Unlock()

	log.Printf("[%s] Agent listening on channel: %s", a.Info.Name, channel)

	// Check history for any unanswered messages (handles the race where a
	// message arrived between channel creation and agent subscription).
	if history != nil {
		go a.processUnrespondedHistory(listenerCtx, channel, history)
	}

	go func() {
		for {
			select {
			case <-listenerCtx.Done():
				return
			case <-a.stopCh:
				return
			case msg := <-subCh:
				if msg == nil {
					return
				}
				a.handleMessage(listenerCtx, msg)
			}
		}
	}()

	return nil
}

// processUnrespondedHistory scans recent history for actionable messages that
// this agent may have missed between channel join and subscription readiness.
func (a *Agent) processUnrespondedHistory(ctx context.Context, channel string, history []*protocol.Message) {
	if len(history) == 0 {
		return
	}

	var pending []*protocol.Message
	for i := 0; i < len(history); i++ {
		candidate := history[i]
		if candidate == nil {
			continue
		}
		if candidate.From.ID == a.Info.ID || candidate.From.Name == a.Info.Name {
			continue
		}
		if candidate.Type == protocol.MessageTypeAgentStatus {
			continue
		}
		if !a.shouldRespond(candidate) {
			continue
		}
		if messageTooOldForUnansweredReplay(candidate) {
			continue
		}
		if agentRespondedToUser(history, i, a.Info.ID, a.Info.Name, candidate.ID) {
			continue
		}
		pending = append(pending, candidate)
	}

	for _, candidate := range pending {
		log.Printf("[%s] Found unanswered message in %s history, processing...", a.Info.Name, channel)
		a.handleMessage(ctx, candidate)
	}
}

// discoverChannels periodically checks for new channels this agent was added to.
// Runs every second so agents respond promptly when added to new DM channels.
func (a *Agent) discoverChannels(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			channels := a.Hub.GetAgentChannels(a.Info.ID)
			for _, ch := range channels {
				a.channelMu.Lock()
				_, exists := a.activeChannels[ch]
				a.channelMu.Unlock()
				if !exists {
					if err := a.AddChannel(ctx, ch); err != nil {
						log.Printf("[%s] Failed to add discovered channel %s: %v", a.Info.Name, ch, err)
					}
				}
			}
		}
	}
}

// Stop stops the agent
func (a *Agent) Stop() {
	close(a.stopCh)
	a.channelMu.Lock()
	for _, cancel := range a.activeChannels {
		cancel()
	}
	a.activeChannels = make(map[string]context.CancelFunc)
	a.channelMu.Unlock()
}

// handleMessage processes incoming messages and decides if/how to respond
func (a *Agent) handleMessage(ctx context.Context, msg *protocol.Message) {
	// Handle agent status messages for provider/model updates
	if msg.Type == protocol.MessageTypeAgentStatus {
		if msg.Metadata != nil {
			if v, ok := msg.Metadata[protocol.MetadataChannelInterjectAbort].(bool); ok && v && msg.Channel != "" {
				a.AbortChannel(msg.Channel)
				return
			}
			if _, ok := msg.Metadata[protocol.MetadataChannelHold]; ok && msg.Channel != "" {
				return
			}
			if v, ok := msg.Metadata["history_resync"].(bool); ok && v && msg.Channel != "" {
				if old := a.channelHistory(msg.Channel); len(old) > 0 {
					for _, m := range old {
						if m != nil && m.ID != "" {
							delete(a.respondedMessages, m.ID)
						}
					}
				}
				if hist, err := a.Hub.GetMessages(msg.Channel, 20); err == nil {
					a.replaceChannelHistory(msg.Channel, hist)
				}
				return
			}
		}
		// Check if this status message is for us (updating our provider)
		if msg.From.ID == a.Info.ID {
			// Extract provider info from metadata
			if aiProvider, ok := msg.Metadata["ai_provider"].(string); ok {
				aiModel, _ := msg.Metadata["ai_model"].(string)
				// Update our info to match
				a.Info.AIProvider = aiProvider
				a.Info.AIModel = aiModel
				if aiModel != "" {
					a.Info.Model = aiModel
				}
				// Note: The actual AI provider instance would need to be updated separately
				// This is a limitation - we'd need access to create a new provider instance
				log.Printf("[%s] 📝 Updated provider info to %s (%s)", a.Info.Name, aiProvider, aiModel)
			}
		}
		// Don't process agent status messages further
		return
	}

	// Ignore own messages - check BOTH ID and Name (since ID changes on restart)
	if msg.From.ID == a.Info.ID || msg.From.Name == a.Info.Name {
		return
	}

	// Specialized-agent interception path (e.g., Assistant deterministic actions).
	if a.messageInterceptor != nil && a.messageInterceptor(ctx, msg) {
		return
	}

	// Check if we've already responded to this message (atomic check-and-set)
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMutex.Unlock()

	// Add to history first (so we have context for decision)
	a.addToHistory(msg)

	if a.Hub != nil && a.Hub.IsChannelHeld(msg.Channel) {
		return
	}

	// Decide if we should respond BEFORE marking as responded
	// This allows other agents to process the message if we don't respond
	if !a.shouldRespond(msg) {
		return
	}

	// Reserve this message so duplicate listeners do not start a second generation.
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMessages[msg.ID] = true
	a.respondedMutex.Unlock()
	clearResponded := func() {
		a.respondedMutex.Lock()
		delete(a.respondedMessages, msg.ID)
		a.respondedMutex.Unlock()
	}

	// Log that we're processing this message
	log.Printf("[%s] ⬇️ RECEIVED msg ID %s from %s (mentions: %v)", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Mentions)
	log.Printf("[%s] ✅ MARKED msg %s as responded", a.Info.Name, msg.ID[:8])

	log.Printf("[%s] 💬 WILL RESPOND to msg %s from %s: %s", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Content[:min(50, len(msg.Content))])
	log.Printf("[%s] 🔍 Message details - ThreadID: '%s', IsThreadReply: %v, ReplyTo: '%s'", a.Info.Name, msg.ThreadID, msg.IsThreadReply, msg.ReplyTo)

	// Send thinking status
	a.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)
	a.resetCADWrittenPaths()

	genCtx, genCancel := context.WithCancel(ctx)
	genID := a.registerGenCancel(msg.Channel, genCancel)
	defer a.unregisterGenCancel(msg.Channel, genID)

	// Try streaming path first, fall back to batch
	var response string
	var streamMsgID string
	var reasoningText string
	var err error

	eff := a.EffectiveAIProvider(genCtx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}

	var implSessionProposed bool
	var implSessionFiles []string
	if shouldRunImplementationSession(a, msg) {
		log.Printf("[%s] 🔧 Implementation session...", a.Info.Name)
		if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
			streamMsgID = uuid.New().String()
			response, streamMsgID, implSessionProposed, implSessionFiles, err = a.runImplementationSessionStreaming(genCtx, msg, eff, streamMsgID)
			reasoningText = ""
		} else {
			response, implSessionProposed, implSessionFiles, err = a.runImplementationSession(genCtx, msg, eff)
		}
	} else if sp, ok := eff.(ai.StreamingProvider); ok && sp.SupportsStreaming() {
		log.Printf("[%s] 📡 Streaming response...", a.Info.Name)
		response, streamMsgID, reasoningText, err = a.generateResponseStreaming(genCtx, msg, eff)
	} else {
		log.Printf("[%s] 📝 Generating response (batch)...", a.Info.Name)
		response, err = a.generateResponse(genCtx, msg, eff)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[%s] Generation cancelled (interject)", a.Info.Name)
			clearResponded()
			a.sendThinkingStatus(msg, protocol.ThinkingStatusAborted)
			return
		}
		log.Printf("[%s] Error generating response: %v", a.Info.Name, err)
		clearResponded()
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)

		a.sendGenerationFailureMessages(msg, err)
		return
	}
	response = sanitizeInternalToolNames(response)
	collabCtx := a.getCollaborationContext(msg)
	if collabCtx.ID != "" {
		response = sanitizeCollabDiscussionResponse(response, collabCtx, a.Info.Type)
	}
	var proposedFileChange bool
	var proposalErr error
	if implSessionProposed {
		proposedFileChange = true
	} else if !shouldRunImplementationSession(a, msg) {
		response, proposedFileChange, proposalErr = a.maybeSubmitFileChangeFromResponse(context.Background(), response, msg.Channel, msg)
	}
	if proposalErr != nil {
		log.Printf("[%s] Failed to submit file change proposal from response: %v", a.Info.Name, proposalErr)
	}
	if proposedFileChange {
		if strings.TrimSpace(response) == "" {
			response = "I submitted a file change proposal for your approval."
		} else {
			response += "\n\nI submitted a file change proposal for your approval."
		}
	}
	log.Printf("[%s] ✍️  Generated response: %s", a.Info.Name, response[:min(50, len(response))])

	// Send response -- reuse the stream message ID when available so the
	// frontend can correlate deltas with the final persisted message.
	responseType := protocol.MessageTypeChat
	if msg.GetCollaborationID() != "" &&
		a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		responseType = protocol.MessageTypeCollabDiscussion
	}
	responseMsg := protocol.NewMessage(
		responseType,
		msg.Channel,
		a.Info,
		response,
	)
	if streamMsgID != "" {
		responseMsg.ID = streamMsgID
	}
	if strings.TrimSpace(reasoningText) != "" {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["reasoning_text"] = reasoningText
	}
	if consulted := a.TakeDelegationConsulted(); len(consulted) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata["delegation_consulted"] = consulted
	}
	if shouldRunImplementationSession(a, msg) {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaImplementationComplete] = true
		if len(implSessionFiles) > 0 {
			responseMsg.Metadata[protocol.IdeMetaImplementationFiles] = implSessionFiles
		}
	}
	if paths := a.takeCADWrittenPaths(); len(paths) > 0 {
		if responseMsg.Metadata == nil {
			responseMsg.Metadata = make(map[string]interface{})
		}
		responseMsg.Metadata[protocol.IdeMetaCADFilesWritten] = paths
	}
	responseMsg.ReplyTo = msg.ID

	// If responding to a thread message, keep it in the thread
	if msg.IsInThread() {
		responseMsg.ThreadID = msg.ThreadID
		responseMsg.IsThreadReply = true
		log.Printf("[%s] 🧵 Responding in thread %s (IsInThread: true)", a.Info.Name, msg.ThreadID[:8])
	} else {
		log.Printf("[%s] 📢 Responding in main channel (IsInThread: false)", a.Info.Name)
	}
	log.Printf("[%s] 📤 Response details - ThreadID: '%s', IsThreadReply: %v, ReplyTo: '%s'", a.Info.Name, responseMsg.ThreadID, responseMsg.IsThreadReply, responseMsg.ReplyTo)

	// Check if this is a review request and add metadata
	if msg.ReplyTo != "" {
		handledReviewMetadata := false
		// Look for the message being replied to
		for _, histMsg := range a.channelHistory(msg.Channel) {
			if histMsg.ID == msg.ReplyTo {
				// Check if it's from another agent (review scenario)
				isFromAgent := histMsg.From.Type == protocol.AgentTypeFrontend ||
					histMsg.From.Type == protocol.AgentTypeBackend ||
					histMsg.From.Type == protocol.AgentTypeDatabase ||
					histMsg.From.Type == protocol.AgentTypeSecurity ||
					histMsg.From.Type == protocol.AgentTypeArchitecture ||
					histMsg.From.Type == protocol.AgentTypeCodeReview ||
					histMsg.From.Type == protocol.AgentTypeDevOps ||
					histMsg.From.Type == protocol.AgentTypeRepo ||
					histMsg.From.Type == protocol.AgentTypeExpert ||
					histMsg.From.Type == protocol.AgentTypeCLI

				if isFromAgent {
					// This is a review - track metadata
					currentDepth := msg.GetReviewDepth()
					responseMsg.SetReviewDepth(currentDepth + 1)
					responseMsg.SetReviewedMessageID(msg.ReplyTo)

					// Track original question if available
					originalQuestionID := msg.GetOriginalQuestionID()
					if originalQuestionID == "" {
						// Find the original question by looking back in history
						if histMsg.ReplyTo != "" {
							originalQuestionID = histMsg.ReplyTo
						}
					}
					if originalQuestionID != "" {
						responseMsg.SetOriginalQuestionID(originalQuestionID)
					}

					log.Printf("[%s] 📋 Review metadata: depth=%d, reviewing=%s",
						a.Info.Name, currentDepth+1, msg.ReplyTo[:8])
					handledReviewMetadata = true
				}
				break
			}
		}

		// Fallback for review flows where the replied message is not in local
		// history (for example, reply target was ephemeral status metadata).
		if !handledReviewMetadata && (msg.IsReviewRequest() || msg.GetReviewDepth() > 0) {
			currentDepth := msg.GetReviewDepth()
			responseMsg.SetReviewDepth(currentDepth + 1)
			responseMsg.SetReviewedMessageID(msg.ReplyTo)
			if originalQuestionID := msg.GetOriginalQuestionID(); originalQuestionID != "" {
				responseMsg.SetOriginalQuestionID(originalQuestionID)
			}
			log.Printf("[%s] 📋 Review metadata fallback: depth=%d, reviewing=%s",
				a.Info.Name, currentDepth+1, msg.ReplyTo[:8])
		}
	}

	ApplyCollaborationTaskMetadataOnReply(responseMsg, msg, response)

	// Detect commands in the response and add them to metadata
	commandDetector := protocol.NewCommandDetector(nil)
	suggestions := commandDetector.DetectCommands(response, a.Info.Name, responseMsg.ID)
	suggestions = filterCollabCommandSuggestions(msg, suggestions)
	if cwd := collaborationWorkingDirectoryForMessage(a, msg); cwd != "" && len(suggestions) > 0 {
		for i := range suggestions {
			suggestions[i].Cwd = cwd
		}
	}
	if len(suggestions) > 0 {
		responseMsg.Metadata["suggested_commands"] = suggestions
		log.Printf("[%s] 🔧 Detected %d command suggestions", a.Info.Name, len(suggestions))
	}

	responseHasImage := false
	if a.isCLIAgent() {
		workDir := a.resolveCLIWorkDir(msg)
		responseHasImage = AttachGeneratedImageFromResponse(responseMsg, workDir)
		if responseHasImage {
			log.Printf("[%s] 🖼️ Attached generated image from CLI response path", a.Info.Name)
		}
	}

	log.Printf("[%s] 📤 Sending response msg ID %s (replying to %s)...", a.Info.Name, responseMsg.ID[:8], msg.ID[:8])
	if err := a.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Error sending message: %v", a.Info.Name, err)
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}
	learning.MaybeSuggestAfterAgentReply(msg.Channel, a.Info.ID, a.Info.Name, string(a.Info.Type), msg.Content, response)
	log.Printf("[%s] ✅ Response sent successfully!", a.Info.Name)
	// Keep local history in sync with hub (own replies are ignored on Subscribe).
	a.addToHistory(responseMsg)
	a.MaybePostHubGeneratedImageForCLI(msg, responseHasImage)
	a.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)

	// Record planning/review discussion turns; execution is task-driven (no round-robin).
	if collabID := responseMsg.GetCollaborationID(); collabID != "" && a.Collab != nil && msg.Type != protocol.MessageTypeCollabRecap {
		collabPhase := a.Collab.GetCollaboration(collabID, a.Info.ID).Phase
		if collabPhase != "executing" {
			if err := a.Collab.RecordMessage(collabID, responseMsg); err != nil {
				log.Printf("[%s] Warning: failed to record collaboration message: %v", a.Info.Name, err)
			}
			a.Collab.AnalyzeConsensus(collabID, responseMsg)
			a.promptNextCollaborationTurn(responseMsg, collabID)
		}
	}
}

// promptNextCollaborationTurn emits a deterministic handoff prompt so the next
// participant receives an explicit trigger after each accepted collaboration turn.
func (a *Agent) promptNextCollaborationTurn(source *protocol.Message, collabID string) {
	if source == nil || a.Collab == nil || !a.Collab.IsActive(collabID) {
		return
	}
	if a.Hub != nil && a.Hub.IsChannelHeld(source.Channel) {
		return
	}

	nextAgentID, err := a.Collab.GetCurrentTurnAgent(collabID)
	if err != nil || strings.TrimSpace(nextAgentID) == "" || nextAgentID == a.Info.ID {
		return
	}

	// Only prompt when the selected participant is currently eligible to respond.
	if !a.Collab.IsAgentTurn(collabID, nextAgentID) {
		return
	}

	collabInfo := a.Collab.GetCollaboration(collabID, a.Info.ID)
	if collabInfo.Phase == "reviewing" || collabInfo.Phase == "approved" || collabInfo.Phase == "executing" {
		return
	}

	handoffBody := collaborationTurnHandoffBody(collabInfo.Phase)
	if handoffBody == "" {
		return
	}

	turnMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		source.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		handoffBody,
	)
	turnMsg.SetCollaborationID(collabID)
	phase := collabInfo.Phase
	if phase == "" {
		phase = source.GetCollaborationPhase()
	}
	if phase != "" {
		turnMsg.SetCollaborationPhase(phase)
	}
	turnMsg.Mentions = []string{nextAgentID}
	if turnMsg.Metadata == nil {
		turnMsg.Metadata = map[string]interface{}{}
	}
	turnMsg.Metadata["collab_internal_event"] = true
	if messageHasWorkspaceContext(source) {
		if raw, ok := source.Metadata["workspace_context"]; ok {
			if ctx, ok := raw.(map[string]interface{}); ok {
				inheritWorkspaceContextFromCollaboration(turnMsg, ctx)
			}
		}
	} else {
		info := a.Collab.GetCollaboration(collabID, a.Info.ID)
		inheritWorkspaceContextFromCollaboration(turnMsg, info.SourceWorkspaceContext)
	}

	if err := a.Hub.SendMessage(turnMsg); err != nil {
		log.Printf("[%s] Warning: failed to send collaboration turn handoff: %v", a.Info.Name, err)
	}
}

// SetMessageInterceptor sets an optional message pre-processing hook.
func (a *Agent) SetMessageInterceptor(interceptor func(context.Context, *protocol.Message) bool) {
	a.messageInterceptor = interceptor
}

// SetPromptBuilder replaces the default buildPrompt for this agent instance.
func (a *Agent) SetPromptBuilder(builder func(*protocol.Message) string) {
	a.customPromptBuilder = builder
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCollaborationContext returns collaboration info for the message if the
// agent is participating in an active collaboration. Returns a zero-value
// CollaborationInfo if no collaboration is active.
func (a *Agent) getCollaborationContext(msg *protocol.Message) CollaborationInfo {
	if a.Collab == nil {
		return CollaborationInfo{}
	}
	collabID := msg.GetCollaborationID()
	if collabID == "" {
		return CollaborationInfo{}
	}
	if !a.Collab.IsParticipant(collabID, a.Info.ID) {
		return CollaborationInfo{}
	}
	info := a.Collab.GetCollaboration(collabID, a.Info.ID)
	if info.ID == "" {
		return CollaborationInfo{}
	}
	info.WorkingDirectory = a.Collab.GetCollaborationWorkingDirectory(collabID)
	return info
}

func collaborationWorkingDirectoryForMessage(a *Agent, msg *protocol.Message) string {
	if msg == nil || a == nil || a.Collab == nil {
		return ""
	}
	cid := msg.GetCollaborationID()
	if cid == "" {
		return ""
	}
	info := a.Collab.GetCollaboration(cid, a.Info.ID)
	if p := strings.TrimSpace(info.SourceRepoPath); p != "" {
		return p
	}
	if p := strings.TrimSpace(info.WorkingDirectory); p != "" {
		return p
	}
	return a.Collab.GetCollaborationWorkingDirectory(cid)
}

// registerGenCancel tracks a cancellable generation for channel interject.
func (a *Agent) registerGenCancel(channel string, cancel context.CancelFunc) string {
	genID := uuid.New().String()
	a.activeGenMu.Lock()
	defer a.activeGenMu.Unlock()
	if a.activeGens == nil {
		a.activeGens = make(map[string]map[string]context.CancelFunc)
	}
	if a.activeGens[channel] == nil {
		a.activeGens[channel] = make(map[string]context.CancelFunc)
	}
	a.activeGens[channel][genID] = cancel
	return genID
}

func (a *Agent) unregisterGenCancel(channel, genID string) {
	a.activeGenMu.Lock()
	defer a.activeGenMu.Unlock()
	if chGens, ok := a.activeGens[channel]; ok {
		delete(chGens, genID)
		if len(chGens) == 0 {
			delete(a.activeGens, channel)
		}
	}
}

// RegisterGenCancelForTest registers an active generation cancel func (tests only).
func RegisterGenCancelForTest(a *Agent, channel string, cancel context.CancelFunc) string {
	return a.registerGenCancel(channel, cancel)
}

// ActiveGenCount returns the number of in-flight generations on a channel (tests only).
func ActiveGenCountForTest(a *Agent, channel string) int {
	a.activeGenMu.Lock()
	defer a.activeGenMu.Unlock()
	return len(a.activeGens[channel])
}

// AbortChannel cancels all in-flight generations on a channel (user Stop / interject).
func (a *Agent) AbortChannel(channel string) {
	a.activeGenMu.Lock()
	chGens := a.activeGens[channel]
	cancels := make([]context.CancelFunc, 0, len(chGens))
	for _, c := range chGens {
		cancels = append(cancels, c)
	}
	delete(a.activeGens, channel)
	a.activeGenMu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// AbortAllChannels cancels in-flight generations on every channel (e.g. /pause-agent).
func (a *Agent) AbortAllChannels() {
	a.activeGenMu.Lock()
	cancels := make([]context.CancelFunc, 0)
	for ch, gens := range a.activeGens {
		for _, c := range gens {
			cancels = append(cancels, c)
		}
		delete(a.activeGens, ch)
	}
	a.activeGenMu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// sendThinkingStatus sends an agent_status message indicating thinking state
func (a *Agent) sendThinkingStatus(originalMsg *protocol.Message, status protocol.ThinkingStatus) {
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"", // Empty content for status messages
	)
	if statusMsg.Metadata == nil {
		statusMsg.Metadata = make(map[string]interface{})
	}
	statusMsg.Metadata["thinking_status"] = string(status)
	statusMsg.Metadata["question_id"] = originalMsg.ID

	// Fire and forget - don't block on sending status
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send thinking status: %v", a.Info.Name, err)
		}
	}()
}

// effectiveChannelType resolves channel classification for routing. DM rooms use
// the "dm-" name prefix; if hub metadata is missing or wrong, still treat as DM
// so 1:1 agents answer the user.
func (a *Agent) effectiveChannelType(channel string) protocol.ChannelType {
	if channel == "" {
		return protocol.ChannelTypePublic
	}
	t := a.Hub.GetChannelType(channel)
	if t == protocol.ChannelTypeDM {
		return protocol.ChannelTypeDM
	}
	if strings.HasPrefix(strings.ToLower(channel), "dm-") {
		return protocol.ChannelTypeDM
	}
	if t == protocol.ChannelTypeCollaboration {
		return protocol.ChannelTypeCollaboration
	}
	if strings.HasPrefix(strings.ToLower(channel), "collab-") {
		return protocol.ChannelTypeCollaboration
	}
	return t
}

// taskAssigneeFromMetadata reads task_assigned_to from collaboration_task metadata.
// JSON decoding can surface non-string types; normalize so assignee routing matches.
func isHumanCollabSpeaker(msg *protocol.Message) bool {
	if msg == nil || msg.IsFromSystem() {
		return false
	}
	return msg.From.Type == protocol.AgentTypeGeneral
}

// collabOutOfTurnMentionOK allows @mentions to wake an agent outside round-robin.
// During planning/review, only humans and system turn prompts may do so — not
// @mentions embedded in another agent's plan prose (which would skip participants).
func collabOutOfTurnMentionOK(msg *protocol.Message, phase string) bool {
	if msg == nil {
		return false
	}
	if msg.IsFromSystem() {
		return true
	}
	switch phase {
	case "planning", "reviewing":
		return isHumanCollabSpeaker(msg)
	default:
		return true
	}
}

func taskAssigneeFromMetadata(meta map[string]interface{}) (string, bool) {
	if meta == nil {
		return "", false
	}
	v, ok := meta["task_assigned_to"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		return s, s != ""
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		return s, s != ""
	}
}

func recapAssigneeFromMetadata(meta map[string]interface{}) (string, bool) {
	if meta == nil {
		return "", false
	}
	v, ok := meta["recap_assignee"]
	if !ok || v == nil {
		return "", false
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		return s, s != ""
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		return s, s != ""
	}
}

func (a *Agent) collabTaskRateLimitOK(collabID, taskID string) bool {
	key := collabID
	if taskID != "" {
		key = collabID + ":" + taskID
	}
	a.collabTaskReplyMu.Lock()
	defer a.collabTaskReplyMu.Unlock()
	if a.collabTaskReplyAt == nil {
		a.collabTaskReplyAt = make(map[string]time.Time)
	}
	if last, ok := a.collabTaskReplyAt[key]; ok && time.Since(last) < collabTaskMinReplyInterval {
		return false
	}
	a.collabTaskReplyAt[key] = time.Now()
	return true
}

// shouldRespond determines if the agent should respond to a message
func (a *Agent) shouldRespond(msg *protocol.Message) bool {
	if a.Hub != nil && a.Hub.IsChannelHeld(msg.Channel) {
		return false
	}
	// Slack bridge policy routing (e.g. always → Assistant on every line).
	if routed := msg.SlackRoutedAgentID(); routed != "" {
		return routed == a.Info.ID
	}
	// Forward-mode inbox lines are for manual owner reply, not agent auto-response.
	if msg.SlackInboxAwaitingManualReply() {
		return false
	}
	// IDE file-tab routing applies only when the user did not @mention a specific agent.
	if routedType := msg.IdeRouteAgentType(); routedType != "" && !msg.HasMentions() {
		if userRequestsCodeReview(msg.Content) {
			return false
		}
		return strings.EqualFold(string(a.Info.Type), routedType)
	}

	// Repo-path project reviews defer to repo expert agents (pending or indexed).
	if a.Hub != nil && messageDefersToRepoExpert(a, msg) {
		return false
	}

	// Never respond to commands - let the command handler process them
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return false
	}

	if len(protocol.ExtractUserImages(msg)) > 0 && !a.Info.SupportsVision && !a.isCLIAgent() {
		return false
	}

	// Special handling for design analysis requests
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		if !a.Info.SupportsVision {
			return false
		}
		if msg.HasMentions() && msg.IsMentioned(a.Info.ID) {
			log.Printf("[%s] 🎨 DESIGN ANALYSIS request detected - will respond", a.Info.Name)
			return true
		}
		return false
	}

	// COLLABORATION: orchestration messages (turn prompts, tasks) are sent from
	// System — evaluate before the generic "ignore System" rule below.
	if collabID := msg.GetCollaborationID(); collabID != "" && a.Collab != nil {
		if msg.Type == protocol.MessageTypeCollabRecap && msg.Metadata != nil {
			if assignee, ok := recapAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
				log.Printf("[%s] ✅ COLLABORATION RECAP - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			return false
		}
		if msg.Metadata != nil {
			if internal, ok := msg.Metadata["collab_internal_event"].(bool); ok && internal {
				// Seed banners and status noise stay ignored; turn prompts must wake the next speaker.
				if !isCollabTurnPromptForAgent(msg, collabID, a.Info.ID, a.Collab) {
					return false
				}
			}
		}
		if a.Collab.IsParticipant(collabID, a.Info.ID) && a.Collab.IsActive(collabID) {
			collabPhase := a.Collab.GetCollaboration(collabID, a.Info.ID).Phase
			if collabPhase == "executing" {
				if msg.Type == protocol.MessageTypeCollabTask && msg.Metadata != nil {
					if assignee, ok := taskAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
						if !a.collabTaskRateLimitOK(collabID, msg.GetTaskID()) {
							log.Printf("[%s] ⏳ COLLABORATION TASK rate-limited (collab %s)", a.Info.Name, collabID[:8])
							return false
						}
						log.Printf("[%s] ✅ COLLABORATION TASK (assignee metadata) - will respond (collab %s)", a.Info.Name, collabID[:8])
						return true
					}
				}
				if msg.Type == protocol.MessageTypeCollabRecap && msg.Metadata != nil {
					if assignee, ok := recapAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
						log.Printf("[%s] ✅ COLLABORATION RECAP - will respond (collab %s)", a.Info.Name, collabID[:8])
						return true
					}
				}
				if !msg.IsFromSystem() && msg.IsMentioned(a.Info.ID) && isHumanCollabSpeaker(msg) {
					log.Printf("[%s] ✅ HUMAN @mention during execution - will respond (collab %s)", a.Info.Name, collabID[:8])
					return true
				}
				log.Printf("[%s] ⏸ execution phase — task prompts only (collab %s)", a.Info.Name, collabID[:8])
				return false
			}
			if msg.Type == protocol.MessageTypeCollabTask && msg.Metadata != nil {
				if assignee, ok := taskAssigneeFromMetadata(msg.Metadata); ok && assignee == a.Info.ID {
					if !a.collabTaskRateLimitOK(collabID, msg.GetTaskID()) {
						log.Printf("[%s] ⏳ COLLABORATION TASK rate-limited (collab %s)", a.Info.Name, collabID[:8])
						return false
					}
					log.Printf("[%s] ✅ COLLABORATION TASK (assignee metadata) - will respond (collab %s)", a.Info.Name, collabID[:8])
					return true
				}
			}
			if a.Collab.IsAgentTurn(collabID, a.Info.ID) {
				log.Printf("[%s] ✅ COLLABORATION TURN - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			if msg.IsMentioned(a.Info.ID) && a.Collab.AgentOutOfTurnMentionAllowed(collabID) &&
				collabOutOfTurnMentionOK(msg, collabPhase) {
				log.Printf("[%s] ✅ MENTIONED in collaboration - will respond (collab %s)", a.Info.Name, collabID[:8])
				return true
			}
			return false
		}
	}

	// Collaboration channel without metadata: still block agent chatter after discussion limits.
	if a.Collab != nil && a.effectiveChannelType(msg.Channel) == protocol.ChannelTypeCollaboration {
		if info := a.Collab.GetCollaborationForAgent(a.Info.ID); info.ID != "" && info.Channel == msg.Channel {
			isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
				msg.From.Type == protocol.AgentTypeBackend ||
				msg.From.Type == protocol.AgentTypeDatabase ||
				msg.From.Type == protocol.AgentTypeSecurity ||
				msg.From.Type == protocol.AgentTypeRust ||
				msg.From.Type == protocol.AgentTypeArchitecture ||
				msg.From.Type == protocol.AgentTypeCodeReview ||
				msg.From.Type == protocol.AgentTypeBiology ||
				msg.From.Type == protocol.AgentTypeDevOps ||
				msg.From.Type == protocol.AgentTypeRepo ||
				msg.From.Type == protocol.AgentTypeExpert ||
				msg.From.Type == protocol.AgentTypeAssistant ||
				msg.From.Type == protocol.AgentTypeModerator ||
				msg.From.Type == protocol.AgentTypeCLI ||
				msg.From.Type == protocol.AgentTypeConfluence
			if isFromAgent && msg.From.ID != a.Info.ID {
				if info.Phase == "executing" {
					// Peer task replies are collaboration_discussion; only task prompts wake assignees (handled above).
					return false
				}
				if !a.Collab.IsAgentTurn(info.ID, a.Info.ID) && !a.Collab.AgentOutOfTurnMentionAllowed(info.ID) {
					log.Printf("[%s] ⏸ collaboration discussion closed — ignoring (collab %s)", a.Info.Name, info.ID[:8])
					return false
				}
			}
		}
	}

	// Never respond to system messages (errors, notifications, join/leave, etc.)
	if msg.From.Name == "System" || msg.From.ID == "system" {
		return false
	}
	if msg.Type == protocol.MessageTypeSystemInfo || msg.Type == protocol.MessageTypeAgentJoin || msg.Type == protocol.MessageTypeAgentLeave {
		return false
	}

	// Never respond to our own messages
	if msg.From.ID == a.Info.ID {
		return false
	}

	// DM channels: answer the human before any collaboration turn logic. The user
	// is always talking to this agent in a 1:1 room.
	if a.isDMChannel(msg.Channel) {
		isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
			msg.From.Type == protocol.AgentTypeBackend ||
			msg.From.Type == protocol.AgentTypeDatabase ||
			msg.From.Type == protocol.AgentTypeSecurity ||
			msg.From.Type == protocol.AgentTypeRust ||
			msg.From.Type == protocol.AgentTypeArchitecture ||
			msg.From.Type == protocol.AgentTypeCodeReview ||
			msg.From.Type == protocol.AgentTypeBiology ||
			msg.From.Type == protocol.AgentTypeDevOps ||
			msg.From.Type == protocol.AgentTypeRepo ||
			msg.From.Type == protocol.AgentTypeExpert ||
			msg.From.Type == protocol.AgentTypeAssistant ||
			msg.From.Type == protocol.AgentTypeModerator ||
			msg.From.Type == protocol.AgentTypeCLI ||
			msg.From.Type == protocol.AgentTypeConfluence
		if !isFromAgent {
			if DMHumanMessageShouldRespond(msg, a.Info.ID) {
				log.Printf("[%s] ✅ DM CHANNEL - will respond", a.Info.Name)
				return true
			}
			return false
		}
		return false
	}

	channelType := a.effectiveChannelType(msg.Channel)

	// THREAD HANDLING: In threads, respond if mentioned OR if we posted the parent message
	if msg.IsInThread() {
		threadID := msg.GetThreadID()

		// Check if we posted the parent message (thread was created from our message)
		parentAuthorID := a.Hub.GetThreadParentAuthor(threadID)
		if parentAuthorID == a.Info.ID {
			log.Printf("[%s] ✅ THREAD PARENT AUTHOR - will respond (thread created from our message)", a.Info.Name)
			return true
		}

		// Check if explicitly mentioned
		if msg.HasMentions() && msg.IsMentioned(a.Info.ID) {
			log.Printf("[%s] ✅ MENTIONED in thread - will respond", a.Info.Name)
			return true
		}

		return false
	}

	// Check if message is from another agent (not a human)
	// We need to check this BEFORE mention checking, but handle mentions specially
	isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
		msg.From.Type == protocol.AgentTypeBackend ||
		msg.From.Type == protocol.AgentTypeDatabase ||
		msg.From.Type == protocol.AgentTypeSecurity ||
		msg.From.Type == protocol.AgentTypeRust ||
		msg.From.Type == protocol.AgentTypeArchitecture ||
		msg.From.Type == protocol.AgentTypeCodeReview ||
		msg.From.Type == protocol.AgentTypeBiology ||
		msg.From.Type == protocol.AgentTypeDevOps ||
		msg.From.Type == protocol.AgentTypeRepo ||
		msg.From.Type == protocol.AgentTypeExpert ||
		msg.From.Type == protocol.AgentTypeAssistant ||
		msg.From.Type == protocol.AgentTypeModerator ||
		msg.From.Type == protocol.AgentTypeCLI

	// If message has @mentions, ONLY respond if explicitly mentioned
	// This works even for agent-to-agent communication if explicitly mentioned
	if msg.HasMentions() {
		if msg.IsMentioned(a.Info.ID) {
			// Check if this is a review request (replying to another agent's message)
			if msg.ReplyTo != "" {
				// Enforce max review depth from explicit metadata even when the
				// replied message is missing from local history.
				if msg.GetReviewDepth() >= 1 {
					return false
				}

				// Find the message being replied to
				var repliedToMsg *protocol.Message
				for _, histMsg := range a.channelHistory(msg.Channel) {
					if histMsg.ID == msg.ReplyTo {
						repliedToMsg = histMsg
						break
					}
				}

				// Check if the replied-to message is from an agent
				if repliedToMsg != nil {
					isRepliedToAgent := repliedToMsg.From.Type == protocol.AgentTypeFrontend ||
						repliedToMsg.From.Type == protocol.AgentTypeBackend ||
						repliedToMsg.From.Type == protocol.AgentTypeDatabase ||
						repliedToMsg.From.Type == protocol.AgentTypeSecurity ||
						repliedToMsg.From.Type == protocol.AgentTypeRust ||
						repliedToMsg.From.Type == protocol.AgentTypeArchitecture ||
						repliedToMsg.From.Type == protocol.AgentTypeCodeReview ||
						repliedToMsg.From.Type == protocol.AgentTypeBiology ||
						repliedToMsg.From.Type == protocol.AgentTypeDevOps ||
						repliedToMsg.From.Type == protocol.AgentTypeRepo ||
						repliedToMsg.From.Type == protocol.AgentTypeExpert ||
						repliedToMsg.From.Type == protocol.AgentTypeAssistant ||
						repliedToMsg.From.Type == protocol.AgentTypeModerator ||
						repliedToMsg.From.Type == protocol.AgentTypeCLI

					if isRepliedToAgent {
						// This is a review request - check depth limits
						repliedToDepth := repliedToMsg.GetReviewDepth()
						if repliedToDepth >= 1 {
							return false
						}

						// Valid review request (depth 0 -> will become depth 1)
						log.Printf("[%s] ✅ REVIEW REQUEST detected (replied message depth %d, replying to %s)",
							a.Info.Name, repliedToDepth, msg.ReplyTo[:8])
						return true
					}
				}
			}

			// Not a review, or regular mention - always respond
			log.Printf("[%s] ✅ EXPLICITLY MENTIONED - will respond", a.Info.Name)
			return true
		}
		// Not mentioned but message has mentions - don't respond
		return false
	}

	// If no mentions specified, don't respond to other agents to prevent loops
	// Only respond to human messages when not explicitly mentioned
	if isFromAgent {
		return false
	}

	// Always respond if mentioned by name in the content
	if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(a.Info.Name)) {
		return true
	}

	// Respond to questions related to expertise
	content := strings.ToLower(msg.Content)

	// In custom channels we allow intent-style requests (without "?") so
	// relevant specialists can auto-respond without explicit @mentions.
	isQuestion := msg.Type == protocol.MessageTypeQuestion ||
		strings.Contains(content, "?")
	if !isQuestion && (channelType == protocol.ChannelTypeCustom || channelType == protocol.ChannelTypeCollaboration) {
		isQuestion = looksLikeUserRequest(content)
	}

	if !isQuestion {
		return false
	}

	// Check if STRONGLY related to our expertise
	// Use word boundaries to prevent false positives (e.g., "task" matching "task management")
	words := strings.Fields(content)
	wordSet := make(map[string]bool)
	for _, word := range words {
		// Remove punctuation for matching
		word = strings.Trim(word, ".,!?;:")
		if len(word) >= 2 {
			wordSet[word] = true
		}
	}

	// Check expertise keywords - require whole word matches
	relevanceScore := 0
	for _, skill := range a.Info.Expertise {
		skillLower := strings.ToLower(skill)
		skillWords := strings.Fields(skillLower)

		// Check if any significant word from expertise appears in message
		for _, skillWord := range skillWords {
			skillWord = strings.Trim(skillWord, ".,!?;:")
			if len(skillWord) >= 2 && wordSet[skillWord] {
				relevanceScore += 2
			}
		}

		// Also check for full skill phrase match (for multi-word skills like "task management")
		if len(skillWords) > 1 {
			skillPhrase := strings.Join(skillWords, " ")
			if strings.Contains(content, skillPhrase) {
				relevanceScore += 3
			}
		}
	}

	// Check agent type keywords - require whole word matches only
	typeKeywords := a.getTypeKeywords()
	for _, keyword := range typeKeywords {
		// Must be a whole word match to prevent false positives
		if wordSet[keyword] ||
			strings.Contains(content, " "+keyword+" ") ||
			strings.HasPrefix(content, keyword+" ") ||
			strings.HasSuffix(content, " "+keyword) {
			relevanceScore++
		}
	}

	// Custom- and collaboration-channel behavior: prefer expertise-relevant replies, and only fall
	// back to broad prompts with a responder cap to reduce noise.
	if (channelType == protocol.ChannelTypeCustom || channelType == protocol.ChannelTypeCollaboration) && msg.From.Type == "human" && !msg.HasMentions() {
		if relevanceScore >= customChannelRelevanceMinScore {
			return true
		}
		if isCustomChannelPrompt(content) && a.allowCustomChannelBroadPromptReply(msg) {
			return true
		}
		return false
	}

	if relevanceScore > 0 {
		return true
	}

	return false
}

// getTypeKeywords returns keywords related to the agent's type
func (a *Agent) getTypeKeywords() []string {
	switch a.Info.Type {
	case protocol.AgentTypeFrontend:
		return []string{"ui", "frontend", "react", "vue", "angular", "css", "html", "component", "user interface"}
	case protocol.AgentTypeBackend:
		return []string{"api", "backend", "server", "endpoint", "service", "database", "business logic"}
	case protocol.AgentTypeDevOps:
		return []string{"deploy", "deployment", "ci/cd", "docker", "kubernetes", "infrastructure", "monitoring",
			"aws", "azure", "gcp", "cloud", "terraform", "ansible", "pipeline", "ecs", "eks", "lambda"}
	case protocol.AgentTypeDatabase:
		return []string{"database", "sql", "query", "schema", "migration", "postgres", "mysql", "mongodb",
			"db", "documentdb", "dynamodb", "aurora", "rds", "nosql", "redis", "index"}
	case protocol.AgentTypeSecurity:
		return []string{"security", "auth", "authentication", "authorization", "encryption", "vulnerability", "xss", "sql injection",
			"iam", "ssl", "tls", "cors", "csrf", "rbac", "jwt", "oauth2", "secrets"}
	case protocol.AgentTypeRust:
		return []string{"rust", "cargo", "tokio", "ownership", "borrowing", "lifetime", "trait", "async", "unsafe", "wasm", "serde", "crate"}
	case protocol.AgentTypeArchitecture:
		return []string{"architecture", "architect", "system design", "design", "scalability", "reliability", "tradeoff", "migration", "service boundary", "integration"}
	case protocol.AgentTypeCodeReview:
		return []string{"review", "code review", "correctness", "maintainability", "testing", "refactor", "regression", "readability", "quality"}
	case protocol.AgentTypeBiology:
		return []string{"biology", "protein", "gene", "genome", "dna", "rna", "sequence", "assay", "crispr", "enzyme", "mutation", "pathway", "cell", "lab", "protocol"}
	default:
		return []string{}
	}
}

func looksLikeUserRequest(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	requestPrefixes := []string{
		"how ", "what ", "why ", "where ", "when ", "can ", "could ", "would ",
		"please ", "help ", "show ", "build ", "create ", "fix ", "debug ",
		"review ", "explain ", "plan ", "implement ",
	}
	for _, prefix := range requestPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return strings.Contains(trimmed, " please ") || strings.HasSuffix(trimmed, " please")
}

// shouldInjectWorkspaceCode decides whether to proactively inject workspace code
// context for a message. We only do this for code-analysis intents, not for
// capability/permission/tasking questions (e.g. "can you create files?").
func shouldInjectWorkspaceCode(content string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(content))
	if trimmed == "" {
		return false
	}

	// Some "can you ..." prompts are actually concrete code-access requests
	// and should load workspace code context.
	if shouldTreatCapabilityAsCodeRequest(trimmed) {
		return true
	}

	// Capability/permission style prompts should stay direct and not be drowned
	// by large workspace code context.
	capabilityPrefixes := []string{
		"can you ", "could you ", "are you able", "are you allowed",
		"do you support", "can i ", "could i ",
	}
	for _, p := range capabilityPrefixes {
		if strings.HasPrefix(trimmed, p) {
			return false
		}
	}
	capabilityPhrases := []string{
		"create files", "create a file", "add a readme", "write files",
		"edit files", "modify files", "make changes",
	}
	for _, p := range capabilityPhrases {
		if strings.Contains(trimmed, p) {
			return false
		}
	}

	// Positive signals for code-level analysis where source context is helpful.
	codeIntentPhrases := []string{
		"review", "analyze", "audit", "debug", "trace", "walk through",
		"explain this code", "why is this failing", "where is", "find in code",
		"refactor", "fix bug", "line ", "function ", "struct ", "trait ",
	}
	for _, p := range codeIntentPhrases {
		if strings.Contains(trimmed, p) {
			return true
		}
	}

	// Explicit file paths strongly indicate code-context intent.
	return len(DetectFilePaths(content)) > 0
}

func shouldTreatCapabilityAsCodeRequest(trimmedLower string) bool {
	codeAccessSignals := []string{
		"open ", "share ", "show ", "read ", "inspect ",
		"source file", "source files", "source code",
		"implementation details", "implementation", "how it works",
		".rs", ".go", ".py", ".ts", ".tsx", ".js",
		"src/", "cargo.toml", "main.rs", "lib.rs",
	}
	for _, s := range codeAccessSignals {
		if strings.Contains(trimmedLower, s) {
			return true
		}
	}
	return false
}

func isCustomChannelPrompt(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	channelPrompts := []string{
		"who's here", "whos here", "who is here", "who all is here", "who all here",
		"in this channel", "anyone here", "everyone here", "roll call",
		"can you all", "could you all", "all of you", "team", "together",
	}
	for _, phrase := range channelPrompts {
		if strings.Contains(trimmed, phrase) {
			return true
		}
	}
	return false
}

func (a *Agent) allowCustomChannelBroadPromptReply(msg *protocol.Message) bool {
	// Best-effort cap: count existing agent replies to this message.
	recent, err := a.Hub.GetMessages(msg.Channel, 80)
	if err == nil {
		replies := 0
		for _, m := range recent {
			if m.Type != protocol.MessageTypeChat || m.ReplyTo != msg.ID {
				continue
			}
			if isAgentType(m.From.Type) {
				replies++
			}
		}
		if replies >= customChannelBroadPromptResponderCap {
			return false
		}
	}

	// Stable deterministic ordering so the same small subset responds.
	agentIDs := []string{}
	channelAgents, err := a.Hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return true
	}
	for _, ag := range channelAgents {
		if isAgentType(ag.Type) {
			agentIDs = append(agentIDs, ag.ID)
		}
	}
	if len(agentIDs) <= customChannelBroadPromptResponderCap {
		return true
	}
	sort.Slice(agentIDs, func(i, j int) bool {
		hi := sha1.Sum([]byte(msg.ID + ":" + agentIDs[i]))
		hj := sha1.Sum([]byte(msg.ID + ":" + agentIDs[j]))
		return strings.Compare(fmt.Sprintf("%x", hi), fmt.Sprintf("%x", hj)) < 0
	})
	limit := customChannelBroadPromptResponderCap
	for i := 0; i < limit && i < len(agentIDs); i++ {
		if agentIDs[i] == a.Info.ID {
			return true
		}
	}
	return false
}

func isAgentType(t protocol.AgentType) bool {
	return t == protocol.AgentTypeFrontend ||
		t == protocol.AgentTypeBackend ||
		t == protocol.AgentTypeDatabase ||
		t == protocol.AgentTypeSecurity ||
		t == protocol.AgentTypeRust ||
		t == protocol.AgentTypeArchitecture ||
		t == protocol.AgentTypeCodeReview ||
		t == protocol.AgentTypeBiology ||
		t == protocol.AgentTypeDevOps ||
		t == protocol.AgentTypeRepo ||
		t == protocol.AgentTypeExpert ||
		t == protocol.AgentTypeAssistant ||
		t == protocol.AgentTypeModerator ||
		t == protocol.AgentTypeCLI
}

// generateResponse generates an AI response based on the message and context
func (a *Agent) generateResponse(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, error) {
	// Check if this is a design analysis request
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		return a.generateDesignAnalysisResponse(ctx, msg)
	}
	intent := a.classifyTurnIntentForMessage(msg)
	a.logTurnIntent(intent, msg)
	if intent == IntentClosure {
		if resp, ok := tryConversationalClosure(a, msg); ok {
			log.Printf("[%s] Conversational closure (no LLM): %q", a.Info.Name, truncateForLog(msg.Content, 60))
			return resp, nil
		}
	}
	if resp, ok := a.tryWorkspaceVisibilityResponse(msg); ok {
		log.Printf("[%s] Workspace visibility (no LLM): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, nil
	}
	if eff == nil {
		eff = a.GetAIProvider()
	}
	a.prepareCLIInvocation(msg)

	// Track files already included in the prompt so the workspace scanner
	// doesn't duplicate them.
	includedFiles := collectIncludedFilePaths(msg)

	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)

	// Auto-detect and load file paths referenced in the user's message.
	wsPath := a.resolveWorkspacePath(msg)
	referencedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" {
		var referencedFiles strings.Builder
		referencedLoaded = AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
		if userRequestsImplementationForMessage(a, msg) || messageNeedsWorkspaceFileLoad(a, msg) {
			referencedLoaded += AppendImplementationSeedFiles(&referencedFiles, a, msg, wsPath, a.Info.Type, includedFiles)
		}
		if referencedFiles.Len() > 0 {
			prompt += referencedFiles.String()
			for _, p := range DetectFilePaths(msg.Content) {
				includedFiles[p] = true
			}
		}
	}

	// Proactively scan the workspace for domain-relevant source files.
	// This lets specialist agents (BackendEngineer, CodeReviewer, etc.) see project
	// code even when the user doesn't mention specific file paths.
	scannedLoaded := 0
	collabInfo := a.getCollaborationContext(msg)
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" && !a.agentHasDedicatedContext() && !a.hasWorkspaceTools() && shouldProactiveScanWorkspaceForMessage(a, msg) && collaborationProactiveWorkspaceScan(msg, collabInfo) {
		existingContextSize := len(prompt) - len(a.buildPromptForIntent(msg, intent))
		if existingContextSize < maxScanChars/2 {
			scanQuery := BuildWorkspaceScanQuery(msg.Content, a.channelHistory(msg.Channel))
			scannedFiles, loadedCount, err := ScanWorkspaceFiles(wsPath, a.Info.Type, scanQuery, maxScanChars, includedFiles)
			if err != nil {
				log.Printf("[%s] Workspace scan failed: %v", a.Info.Name, err)
			} else if scannedFiles != "" {
				prompt += scannedFiles
				scannedLoaded = loadedCount
			}
		}
	}

	if collaborationWorkspaceGroundingLine(msg, collabInfo) {
		openFileLoaded := len(collectIncludedFilePaths(msg))
		totalLoaded := openFileLoaded + referencedLoaded + scannedLoaded
		if g := workspaceGroundingRequirement(totalLoaded, msg.Content, userRequestsImplementationForMessage(a, msg)); g != "" {
			prompt += g
		}
	}

	history := a.conversationHistoryForIntent(msg, intent)

	prompt = a.augmentPromptWithCLIImages(msg, prompt)

	var budgetStats ContextBudgetStats
	prompt, budgetStats = applyContextBudgetForMessage(msg, prompt)
	if budgetStats.Truncated {
		log.Printf("[%s] context budget applied: %d -> %d bytes", a.Info.Name, budgetStats.OriginalBytes, budgetStats.FinalBytes)
	}

	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) > 0 && a.Info.SupportsVision {
		approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
		if mp, ok := eff.(ai.MultimodalProvider); ok {
			return mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
		}
		if len(imgs) == 1 {
			return eff.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
		}
		return "", fmt.Errorf("multiple images require a multimodal-capable provider")
	}

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	if resp, ok := a.tryBiologyScanToolShortcut(approvalCtx, msg); ok {
		return resp, nil
	}
	if len(a.agentToolDefinitions()) > 0 {
		return a.generateWithAgentTools(approvalCtx, msg, prompt, history, eff)
	}
	response, err := eff.GenerateResponse(approvalCtx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	if a.useOllamaContextGuardrails(msg) &&
		(looksLikeOllamaPromptLeak(response) || looksLikeContextStackEcho(msg, response)) {
		retryPrompt := a.buildUltraCompactOllamaPrompt(msg)
		if a.useCompactAssistantOllamaPrompt(msg) {
			retryPrompt = a.buildUltraCompactAssistantOllamaPrompt(msg)
		}
		retry, err2 := eff.GenerateResponse(approvalCtx, retryPrompt, nil)
		if err2 == nil && strings.TrimSpace(retry) != "" &&
			!looksLikeOllamaPromptLeak(retry) && !looksLikeContextStackEcho(msg, retry) {
			log.Printf("[%s] Ollama context-stack echo; used compact retry", a.Info.Name)
			return retry, nil
		}
	}
	if retry := a.maybeRetryConversationalQuality(ctx, msg, response, history, eff); retry != response {
		return retry, nil
	}
	return a.finalizeWorkspaceVisibilityReply(msg, response), nil
}

// generateResponseStreaming builds the same prompt as generateResponse but
// streams the AI response token-by-token. Each token is broadcast to
// subscribers as a stream_delta message. Returns the full accumulated text
// and the stable stream message ID so the caller can reuse it for the
// final chat message (allowing the frontend to correlate streaming with
// the persisted message).
func (a *Agent) generateResponseStreaming(ctx context.Context, msg *protocol.Message, eff ai.AIProvider) (string, string, string, error) {
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		resp, err := a.generateDesignAnalysisResponse(ctx, msg)
		return resp, "", "", err
	}
	intent := a.classifyTurnIntentForMessage(msg)
	a.logTurnIntent(intent, msg)
	if intent == IntentClosure {
		if resp, ok := tryConversationalClosure(a, msg); ok {
			log.Printf("[%s] Conversational closure (no LLM stream): %q", a.Info.Name, truncateForLog(msg.Content, 60))
			return resp, "", "", nil
		}
	}
	if resp, ok := a.tryWorkspaceVisibilityResponse(msg); ok {
		log.Printf("[%s] Workspace visibility (no LLM stream): %q", a.Info.Name, truncateForLog(msg.Content, 60))
		return resp, "", "", nil
	}
	if eff == nil {
		eff = a.GetAIProvider()
	}
	a.prepareCLIInvocation(msg)

	prompt := a.buildPromptForIntent(msg, intent)
	prompt = a.appendDelegationContext(ctx, msg, prompt)

	includedFiles := collectIncludedFilePaths(msg)
	collabInfo := a.getCollaborationContext(msg)

	wsPath := a.resolveWorkspacePath(msg)
	referencedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" {
		var referencedFiles strings.Builder
		referencedLoaded = AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
		if userRequestsImplementationForMessage(a, msg) || messageNeedsWorkspaceFileLoad(a, msg) {
			referencedLoaded += AppendImplementationSeedFiles(&referencedFiles, a, msg, wsPath, a.Info.Type, includedFiles)
		}
		if referencedFiles.Len() > 0 {
			prompt += referencedFiles.String()
			for _, p := range DetectFilePaths(msg.Content) {
				includedFiles[p] = true
			}
		}
	}

	scannedLoaded := 0
	if a.shouldAugmentPromptWithWorkspace(intent, msg) && wsPath != "" && !a.agentHasDedicatedContext() && !a.hasWorkspaceTools() && shouldProactiveScanWorkspaceForMessage(a, msg) && collaborationProactiveWorkspaceScan(msg, collabInfo) {
		basePrompt := a.buildPromptForIntent(msg, intent)
		existingContextSize := len(prompt) - len(basePrompt)
		if existingContextSize < maxScanChars/2 {
			scanQuery := BuildWorkspaceScanQuery(msg.Content, a.channelHistory(msg.Channel))
			scannedFiles, loadedCount, scanErr := ScanWorkspaceFiles(wsPath, a.Info.Type, scanQuery, maxScanChars, includedFiles)
			if scanErr != nil {
				log.Printf("[%s] Workspace scan failed: %v", a.Info.Name, scanErr)
			} else if scannedFiles != "" {
				prompt += scannedFiles
				scannedLoaded = loadedCount
			}
		}
	}

	if collaborationWorkspaceGroundingLine(msg, collabInfo) {
		openFileLoaded := len(collectIncludedFilePaths(msg))
		totalLoaded := openFileLoaded + referencedLoaded + scannedLoaded
		if g := workspaceGroundingRequirement(totalLoaded, msg.Content, userRequestsImplementationForMessage(a, msg)); g != "" {
			prompt += g
		}
	}

	history := a.conversationHistoryForIntent(msg, intent)

	// Pre-create a stable message ID for the stream so the frontend can
	// correlate deltas with the final message.
	streamMsgID := uuid.New().String()

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)

	prompt = a.augmentPromptWithCLIImages(msg, prompt)

	var streamBudgetStats ContextBudgetStats
	prompt, streamBudgetStats = applyContextBudgetForMessage(msg, prompt)
	if streamBudgetStats.Truncated {
		log.Printf("[%s] context budget applied (stream): %d -> %d bytes", a.Info.Name, streamBudgetStats.OriginalBytes, streamBudgetStats.FinalBytes)
	}

	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) > 0 && a.Info.SupportsVision {
		if mp, ok := eff.(ai.MultimodalProvider); ok {
			tokenCh, err := mp.GenerateMultimodalStream(approvalCtx, prompt, imgs, historyToMessages(history))
			if err == nil {
				return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
			}
			log.Printf("[%s] Multimodal stream failed (%v), falling back to batch multimodal", a.Info.Name, err)
			text, err := mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
			return text, "", "", err
		}
		if len(imgs) == 1 {
			text, err := eff.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
			return text, "", "", err
		}
		return "", "", "", fmt.Errorf("multiple images require a multimodal-capable provider")
	}

	// Tool loop (MCP / image generation) uses batch API; stream the final answer as one chunk.
	// Run whenever tools exist — generateWithAgentTools falls back to qwen when the chat model (e.g. koesn) lacks native tools.
	if resp, ok := a.tryBiologyScanToolShortcut(approvalCtx, msg); ok {
		tokenCh := make(chan ai.StreamToken, 2)
		tokenCh <- ai.StreamToken{Content: resp}
		tokenCh <- ai.StreamToken{Done: true}
		close(tokenCh)
		return a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
	}
	if len(a.agentToolDefinitions()) > 0 {
		toolCtx := ai.WithToolStepObserver(approvalCtx, func(ev ai.ToolStepEvent) {
			a.broadcastToolStep(approvalCtx, msg, streamMsgID, ev)
		})
		text, err := a.generateWithAgentTools(toolCtx, msg, prompt, history, eff)
		if err != nil {
			return "", "", "", err
		}
		text = a.finalizeWorkspaceVisibilityReply(msg, text)
		return a.streamTextAsTokens(approvalCtx, msg, streamMsgID, text)
	}

	sp, ok := eff.(ai.StreamingProvider)
	if !ok || !sp.SupportsStreaming() {
		return "", "", "", fmt.Errorf("internal: expected streaming-capable provider")
	}

	maxAttempts := 1
	if a.useOllamaContextGuardrails(msg) {
		maxAttempts = 3
	}
	var lastErr error
	attemptPrompt := prompt
	streamProvider := eff
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && a.useOllamaContextGuardrails(msg) {
			if a.useCompactAssistantOllamaPrompt(msg) {
				attemptPrompt = a.buildUltraCompactAssistantOllamaPrompt(msg)
			} else {
				attemptPrompt = a.buildUltraCompactOllamaPrompt(msg)
			}
			history = nil
			log.Printf("[%s] Retrying Ollama stream (attempt %d/%d, prompt %d bytes)", a.Info.Name, attempt+1, maxAttempts, len(attemptPrompt))
		}
		attemptSP, ok := streamProvider.(ai.StreamingProvider)
		if !ok {
			return "", "", "", fmt.Errorf("internal: expected streaming-capable provider")
		}
		tokenCh, err := attemptSP.GenerateResponseStream(approvalCtx, attemptPrompt, historyToMessages(history))
		if err != nil {
			return "", "", "", err
		}
		text, id, reasoning, err := a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
		if err != nil {
			if strings.TrimSpace(text) != "" &&
				a.Info.Type == protocol.AgentTypeAssistant &&
				(messageAsksAboutMeetings(msg.Content) || messageAsksAboutEmail(msg.Content) || messageAsksAboutNJApp(msg.Content)) {
				log.Printf("[%s] Keeping partial grounded Assistant stream for meeting/email query despite stream error: %v", a.Info.Name, err)
				return text, id, reasoning, nil
			}
			lastErr = err
			if errors.Is(err, ai.ErrOllamaNoContent) && attempt+1 < maxAttempts {
				continue
			}
			break
		}
		if (looksLikeOllamaPromptLeak(text) || looksLikeContextStackEcho(msg, text)) && attempt+1 < maxAttempts {
			log.Printf("[%s] Ollama reply looked like prompt/context echo; retrying", a.Info.Name)
			continue
		}
		text = a.maybeRetryConversationalQuality(approvalCtx, msg, text, history, eff)
		text = a.finalizeWorkspaceVisibilityReply(msg, text)
		return text, id, reasoning, nil
	}

	if a.useOllamaContextGuardrails(msg) && errors.Is(lastErr, ai.ErrOllamaNoContent) {
		if fb := ollamaFallbackProvider(eff, ai.OllamaBiologyFallbackModel); fb != nil {
			log.Printf("[%s] nj-bio returned empty; trying fallback model %q", a.Info.Name, ai.OllamaBiologyFallbackModel)
			fbSP, ok := fb.(ai.StreamingProvider)
			if ok && fbSP.SupportsStreaming() {
				fbPrompt := a.buildCompactOllamaPrompt(msg)
				tokenCh, err := fbSP.GenerateResponseStream(approvalCtx, fbPrompt, nil)
				if err == nil {
					text, id, reasoning, err := a.collectStreamTokens(approvalCtx, msg, streamMsgID, tokenCh)
					if err == nil && strings.TrimSpace(text) != "" && !looksLikeOllamaPromptLeak(text) {
						return text, id, reasoning, nil
					}
					if err != nil {
						lastErr = err
					}
				}
			}
		}
	}

	if lastErr != nil {
		return "", "", "", lastErr
	}
	return "", "", "", ai.ErrOllamaNoContent
}

// collectStreamTokens drains a stream channel, broadcasts deltas, emits stream_end, and returns full text.
func (a *Agent) collectStreamTokens(ctx context.Context, msg *protocol.Message, streamMsgID string, tokenCh <-chan ai.StreamToken) (string, string, string, error) {
	var fullResponse strings.Builder
	var fullReasoning strings.Builder
	var streamErr error

	for {
		select {
		case <-ctx.Done():
			streamErr = ctx.Err()
			goto finishStream
		case token, ok := <-tokenCh:
			if !ok {
				goto finishStream
			}
			if token.Error != nil {
				if fullResponse.Len() > 0 || fullReasoning.Len() > 0 {
					streamErr = token.Error
					goto finishStream
				}
				return "", "", "", token.Error
			}
			if token.Thinking != "" {
				fullReasoning.WriteString(token.Thinking)
				delta := protocol.NewMessage(
					protocol.MessageTypeStreamDelta,
					msg.Channel,
					a.Info,
					"",
				)
				delta.ID = streamMsgID
				delta.ReplyTo = msg.ID
				if delta.Metadata == nil {
					delta.Metadata = make(map[string]interface{})
				}
				delta.Metadata["reasoning_delta"] = true
				delta.Metadata["reasoning_append"] = token.Thinking
				if msg.IsInThread() {
					delta.ThreadID = msg.ThreadID
					delta.IsThreadReply = true
				}
				a.Hub.BroadcastDirect(msg.Channel, delta)
			}
			if token.Content != "" {
				fullResponse.WriteString(token.Content)
				delta := protocol.NewMessage(
					protocol.MessageTypeStreamDelta,
					msg.Channel,
					a.Info,
					token.Content,
				)
				delta.ID = streamMsgID
				delta.ReplyTo = msg.ID
				if msg.IsInThread() {
					delta.ThreadID = msg.ThreadID
					delta.IsThreadReply = true
				}
				a.Hub.BroadcastDirect(msg.Channel, delta)
			}
			if token.Done {
				goto finishStream
			}
		}
	}

finishStream:
	if streamErr != nil {
		if errors.Is(streamErr, context.Canceled) {
			if fullResponse.Len() > 0 {
				fullResponse.WriteString("\n\n[stopped]")
			}
			endMsg := protocol.NewMessage(protocol.MessageTypeStreamEnd, msg.Channel, a.Info, "")
			endMsg.ID = streamMsgID
			endMsg.ReplyTo = msg.ID
			if msg.IsInThread() {
				endMsg.ThreadID = msg.ThreadID
				endMsg.IsThreadReply = true
			}
			a.Hub.BroadcastDirect(msg.Channel, endMsg)
			return "", "", "", context.Canceled
		}
		log.Printf("[%s] Stream error with partial content (%d bytes): %v", a.Info.Name, fullResponse.Len(), streamErr)
		fullResponse.WriteString("\n\n[")
		fullResponse.WriteString(truncationLabelForError(streamErr))
		fullResponse.WriteString("]")
	}

	endMsg := protocol.NewMessage(
		protocol.MessageTypeStreamEnd,
		msg.Channel,
		a.Info,
		"",
	)
	endMsg.ID = streamMsgID
	endMsg.ReplyTo = msg.ID
	if msg.IsInThread() {
		endMsg.ThreadID = msg.ThreadID
		endMsg.IsThreadReply = true
	}
	a.Hub.BroadcastDirect(msg.Channel, endMsg)

	return fullResponse.String(), streamMsgID, fullReasoning.String(), nil
}

// agentHasDedicatedContext returns true for agent types that already have their
// own file-context strategy (repo agents use their index, CLI agents have shell access).
func (a *Agent) agentHasDedicatedContext() bool {
	switch a.Info.Type {
	case protocol.AgentTypeRepo, protocol.AgentTypeCLI,
		protocol.AgentTypeModerator, protocol.AgentTypeAssistant, protocol.AgentTypeConfluence:
		return true
	default:
		return false
	}
}

func truncationLabelForError(err error) string {
	_, code, _ := classifyUserFacingError(err)
	switch code {
	case "timeout":
		return "Response truncated due to timeout"
	case "rate_limit":
		return "Response truncated due to provider rate limit"
	default:
		return "Response truncated due to provider error"
	}
}

// appendFileChangeMachineBlockDocs writes the canonical [FILE_CHANGE] spec parsed by
// maybeSubmitFileChangeFromResponse. Shared by normal chat and collaboration execution.
func appendFileChangeMachineBlockDocs(sb *strings.Builder) {
	sb.WriteString("[FILE_CHANGE]\n")
	sb.WriteString("operation: create|edit|delete|move\n")
	sb.WriteString("path: relative/path/from/workspace (must include a file extension, e.g. tailwind.config.js or src/App.tsx — never a label like \"File:\")\n")
	sb.WriteString("old_path: relative/path (move only)\n")
	sb.WriteString("new_path: relative/path (move only)\n")
	sb.WriteString("```new\n<new content for create/edit>\n```\n")
	sb.WriteString("```old\n<old content for edit>\n```\n")
	sb.WriteString("[/FILE_CHANGE]\n")
	sb.WriteString("If no file change should be proposed, do not include a FILE_CHANGE block.\n")
}

// collectIncludedFilePaths extracts file paths that are already present in
// the prompt via workspace context (open editor tabs) so the scanner can
// skip them.
func collectIncludedFilePaths(msg *protocol.Message) map[string]bool {
	paths := make(map[string]bool)
	if msg.Metadata == nil {
		return paths
	}
	if raw, ok := msg.Metadata["task_context_paths"]; ok {
		switch v := raw.(type) {
		case []string:
			for _, p := range v {
				if p != "" {
					paths[p] = true
				}
			}
		case []interface{}:
			for _, item := range v {
				if p, ok := item.(string); ok && p != "" {
					paths[p] = true
				}
			}
		}
	}
	wsCtx, ok := msg.Metadata["workspace_context"]
	if !ok {
		return paths
	}
	ctxMap, ok := wsCtx.(map[string]interface{})
	if !ok {
		return paths
	}
	if files, ok := ctxMap["open_files"].([]interface{}); ok {
		for _, f := range files {
			if fm, ok := f.(map[string]interface{}); ok {
				if p, ok := fm["path"].(string); ok && p != "" {
					paths[p] = true
				}
			}
		}
	}
	return paths
}

// resolveWorkspacePath determines the workspace root from available sources.
// Priority: 1) workspace context metadata, 2) agent's stored WorkspacePath
func (a *Agent) resolveWorkspacePath(msg *protocol.Message) string {
	// Try workspace context metadata first (most accurate)
	if msg.Metadata != nil {
		if wsCtx, ok := msg.Metadata["workspace_context"]; ok {
			if ctxMap, ok := wsCtx.(map[string]interface{}); ok {
				if path, ok := ctxMap["workspace_path"].(string); ok && path != "" {
					// Update stored path for future messages without workspace context
					a.WorkspacePath = path
					return path
				}
			}
		}
	}
	return a.WorkspacePath
}

// buildPrompt constructs the prompt for the AI.
// The output is split into two sections separated by ai.SystemPromptSeparator:
//   - SYSTEM section: agent identity, behavioral rules, domain expertise instructions
//   - USER section: the actual user message, workspace context, and response guidance
//
// AI providers that support a "system" role (Ollama, Claude, LM Studio) will
// split on the separator and send the first part as a system message.
func (a *Agent) buildPrompt(msg *protocol.Message, intent ...TurnIntent) string {
	resolvedIntent := IntentSubstantive
	if len(intent) > 0 {
		resolvedIntent = intent[0]
	}
	if a.customPromptBuilder != nil {
		if isCollabRecapMessage(msg) {
			return buildCollabRecapPrompt(a.Info.Name, msg)
		}
		return a.customPromptBuilder(msg)
	}
	if isCollabRecapMessage(msg) {
		return buildCollabRecapPrompt(a.Info.Name, msg)
	}
	if a.useCompactOllamaPrompt(msg) {
		return a.buildCompactOllamaPrompt(msg)
	}

	personaTier := a.promptPersonaTier(msg)
	includeTooling := a.shouldIncludeToolingInPrompt(msg, resolvedIntent)

	var system strings.Builder
	var user strings.Builder

	// ── SYSTEM SECTION ──────────────────────────────────────────────────
	a.writePersonaOpening(&system, msg, personaTier)

	// Self-knowledge: tell the agent what model/provider it's actually running on
	// so it can answer honestly when users ask "what LLM are you?"
	system.WriteString("=== YOUR TECHNICAL IDENTITY ===\n")
	system.WriteString(fmt.Sprintf("You are powered by the %q model via the %q provider.\n", a.Info.AIModel, a.Info.AIProvider))
	system.WriteString("If a user asks what model or LLM you are running, answer honestly with this information.\n")
	system.WriteString("Do NOT fabricate or guess your model architecture. Only state what is listed above.\n\n")

	// Add domain-specific instructions for this agent type
	typeInstructions := getAgentTypeInstructions(a.Info.Type)
	if typeInstructions != "" {
		system.WriteString("=== DOMAIN EXPERTISE ===\n")
		system.WriteString(typeInstructions)
		system.WriteString("\n\n")
	}

	// Check if this message is part of an active collaboration
	collabInfo := a.getCollaborationContext(msg)
	isCollab := collabInfo.ID != ""

	if a.MCPServer != nil && includeTooling && !(isCollab && collabPlanningSuppressMCPTools(collabInfo, a.Info.Type)) {
		appendMCPToolsPrompt(&system, mcpServerFromInterface(a.MCPServer), a.Info.Type)
	}
	if includeTooling && a.imageGenerationToolsEnabled() {
		appendImageGenerationPrompt(&system)
	}

	if isCollab {
		// Collaboration-specific behavioral rules
		system.WriteString("=== COLLABORATION MODE ===\n")
		system.WriteString(fmt.Sprintf("You are participating in a multi-agent collaboration: %s\n", collabInfo.Description))
		system.WriteString(fmt.Sprintf("Current phase: %s\n", collabInfo.Phase))
		system.WriteString(fmt.Sprintf("Your role: %s\n\n", collabInfo.AgentRole))

		system.WriteString("=== COLLABORATION RULES ===\n")
		system.WriteString("1. Provide expert advice grounded in your domain expertise and assigned role.\n")
		system.WriteString("2. You MAY @mention other agents in this collaboration to:\n")
		system.WriteString("   - Ask for their expert opinion on a specific aspect\n")
		system.WriteString("   - Request they review a section of the plan\n")
		system.WriteString("   - Delegate a sub-problem to the agent best suited for it\n")
		system.WriteString("3. Build on other agents' ideas constructively. Acknowledge good points.\n")
		system.WriteString("4. When you agree with the current plan, explicitly say 'I agree' or 'looks good'.\n")
		system.WriteString("5. When you have concerns, state them clearly with alternatives.\n")
		system.WriteString("6. Keep responses focused and concise -- this is a bounded discussion.\n")
		system.WriteString("7. Reference specific file paths when they support your point; avoid re-scanning or re-summarizing the whole repo each turn.\n")
		system.WriteString("8. Answer the collaboration goal and your task first — workspace files are reference material, not the deliverable.\n")
		system.WriteString("9. Stay in **your lane** (below). Do not assign duplicate tasks across agents or absorb peers' responsibilities.\n")

		appendCollaborationLaneInstructions(&system, collabInfo, a.Info)

		if msg.Type == protocol.MessageTypeCollabRecap {
			system.WriteString("\n=== SESSION RECAP (TO USER) ===\n")
			system.WriteString("You are the designated facilitator. Write a clear recap **to the human user**, not to other agents.\n")
			system.WriteString("Use markdown sections: what we set out to do, what was discussed/decided, plan and tasks OR accomplishments, research findings (even if no code shipped), open questions, and what the user should do next.\n")
			system.WriteString("Do NOT emit TASK_STATUS lines, new plan blocks, or @mention other agents unless quoting them.\n")
		} else if collabInfo.Phase == "planning" {
			system.WriteString("\n=== PLANNING PHASE INSTRUCTIONS ===\n")
			system.WriteString("Propose a **minimal** structured plan: **3–6 tasks total**, each with one primary @assignee in that agent's lane (see YOUR LANE / PEER LANES).\n")
			system.WriteString("Each task must name a **concrete deliverable** (verb + path), not meta-work like \"document findings\" or \"specific actions\".\n")
			system.WriteString("Use this format for tasks:\n")
			system.WriteString("- Task N: @AgentName - Write collabs/<collab-id>/findings.md summarizing …\n")
			system.WriteString("  - depends: 1, 2   (optional; 1-based task numbers this task waits on)\n")
			system.WriteString("Assign file deliverables to the best domain owner (@SoftwareArchitect for schema docs, @Assistant for summaries, @BackendEngineer for code, etc.) — any assignee ships via [FILE_CHANGE].\n")
			if strings.TrimSpace(collabInfo.SourceRepoPath) != "" {
				if rel := collaboration.ProjectCollabRelPath(collabInfo.ID); rel != "" {
					system.WriteString(fmt.Sprintf("File deliverables belong under `%s/` (paths relative to the project root).\n", rel))
				}
			}
			system.WriteString("Consider dependencies between tasks and declare them with depends: lines. Defer debate until the task list is drafted.\n")
			if a.Info.Type == protocol.AgentTypeDevOps {
				system.WriteString("For documentation/schema/API planning: write prose and markdown task descriptions only. ")
				system.WriteString("Do NOT emit kubectl, helm, docker-compose, npm, JSON tool-call payloads, or cluster/build commands unless the user explicitly asked for infrastructure or runtime changes.\n")
			}
			appendCollaborationWorkspaceInstructions(&system, collabInfo, a.Info.Type)
			if !collaborationSkipExtraWorkspaceSection(collabInfo) && !messageHasWorkspaceContext(msg) && len(collabInfo.SourceWorkspaceContext) > 0 {
				appendWorkspacePromptSection(&system, ContextScopeOutline, collabInfo.SourceWorkspaceContext)
			}
		} else if collabInfo.Phase == "executing" {
			system.WriteString("\n=== EXECUTION PHASE INSTRUCTIONS ===\n")
			system.WriteString("Execution is task-driven. Do NOT continue open plan discussion or re-summarize the whole repo.\n")
			system.WriteString("Complete your assigned task: produce concrete deliverables ([FILE_CHANGE] under the collabs folder when files are required).\n")
			system.WriteString("Prefer one focused reply with [FILE_CHANGE] over multi-paragraph discussion.\n")
			system.WriteString("Only @mention another agent if you are blocked on a specific question; otherwise work from task context paths and the goal.\n")
			system.WriteString(CollaborationExecutionTaskStatusInstructions())
			if collabInfo.WorkingDirectory != "" {
				if collabInfo.ExecutionMode == "worktree" {
					system.WriteString(fmt.Sprintf("\n**Execution workspace (git worktree):** %s\n", collabInfo.WorkingDirectory))
					if collabInfo.WorktreeBranch != "" {
						system.WriteString(fmt.Sprintf("**Branch:** `%s`\n", collabInfo.WorktreeBranch))
					}
					if collabInfo.SourceRepoPath != "" {
						system.WriteString(fmt.Sprintf("**Source repo:** %s\n", collabInfo.SourceRepoPath))
					}
					system.WriteString("This is a full copy of the project on an isolated branch. Use paths relative to this root; merge the branch from your main checkout when work is done.\n")
				} else if strings.TrimSpace(collabInfo.SourceRepoPath) != "" {
					system.WriteString(fmt.Sprintf("\n**Execution directory:** %s\n", collabInfo.WorkingDirectory))
					system.WriteString(fmt.Sprintf("**Project root (for reading code):** %s\n", collabInfo.SourceRepoPath))
					if rel := collaboration.ProjectCollabRelPath(collabInfo.ID); rel != "" {
						system.WriteString(fmt.Sprintf("**Deliverables:** write under `%s/` using [FILE_CHANGE] paths relative to the project root.\n", rel))
					}
				} else {
					system.WriteString(fmt.Sprintf("\n**Execution workspace:** %s\n", collabInfo.WorkingDirectory))
					system.WriteString("Use this directory as the root for relative paths and shell commands.\n")
					system.WriteString("When no project repo is bound, write deliverables as flat filenames in this workspace (e.g. `scope.md`), not nested `collabs/<id>/` paths.\n")
				}
			}
			appendCollaborationWorkspaceInstructions(&system, collabInfo, a.Info.Type)
			system.WriteString("To actually create or modify files, you MUST emit a [FILE_CHANGE] block (see below). ")
			system.WriteString("Conversation-only replies do not write to disk.\n")
			system.WriteString("For markdown/file deliverable tasks: read reference paths from the project and ship `[FILE_CHANGE]` — do NOT run docker-compose, npm, make, or other build/deploy tooling unless the task text explicitly requires it.\n")
			system.WriteString("For findings/summary markdown files: include at least three substantive bullet lines grounded in project files (README, main.go, etc.) — not a one-line placeholder.\n")
			system.WriteString("Only when the task requires shell work, put runnable commands in ```bash fenced blocks``` so the host can surface **Run**.\n")
			if a.Info.Type == protocol.AgentTypeDevOps {
				system.WriteString("PlatformEngineer doc tasks: describe CI/CD in prose/markdown; do not invoke kubectl, docker, or npm unless the task explicitly asks to change runtime infrastructure.\n")
			}
			system.WriteString("\n**Workspace scope:** File proposals apply under the project root in WORKSPACE CONTEXT. ")
			system.WriteString("When a deliverables folder is set, write new files there (paths relative to the project root).\n")
			appendFileChangeMachineBlockDocs(&system)
		}

		// Show the current plan artifact if it exists
		if collabInfo.PlanContent != "" {
			system.WriteString(fmt.Sprintf("\n=== CURRENT PLAN (v%d) ===\n", collabInfo.PlanVersion))
			system.WriteString(collabInfo.PlanContent)
			system.WriteString("\n")
		}

		// List collaboration participants
		system.WriteString("\n=== COLLABORATION PARTICIPANTS ===\n")
		for _, agent := range collabInfo.Agents {
			marker := ""
			if agent.Name == a.Info.Name {
				marker = " (you)"
			}
			system.WriteString(fmt.Sprintf("- @%s (%s) -- Role: %s%s\n", agent.Name, agent.Type, agent.Role, marker))
		}
	} else {
		// Standard behavioral rules for non-collaboration mode
		system.WriteString("=== BEHAVIORAL RULES ===\n")
		system.WriteString("1. Provide expert advice grounded in your domain expertise.\n")
		system.WriteString("2. When the user shares code or files, you MUST analyze the ACTUAL code provided -- never give generic advice.\n")
		system.WriteString("3. Reference specific file paths, function names, and line numbers when discussing code.\n")
		if dc := a.getDelegationClient(); dc != nil && dc.DelegationEnabled() {
			system.WriteString("4. The hub may attach DELEGATE_RESULTS from other specialists; synthesize them into your answer. Do not @mention peers for handoff.\n")
		} else {
			system.WriteString("4. Do NOT @mention other agents unless the user explicitly asks for collaboration.\n")
		}
		system.WriteString("5. Only respond to the user's question -- do not respond to other agents' responses.\n")
		system.WriteString("6. Ask clarifying questions when the request is ambiguous.\n")
		if includeTooling {
			system.WriteString("7. CRITICAL: If asked to review, analyze, or explain code but NO code and NO workspace context appear below, ")
			system.WriteString("you MUST tell the user you currently do not have code context and ask them to either: ")
			system.WriteString("(a) include the file path in their message (e.g., 'review cmd/server/main.go'), or ")
			system.WriteString("(b) enable workspace sharing. If workspace context is present, do NOT claim you cannot access files; use available context and request a specific path only when needed. NEVER fabricate or guess code content.\n")
			system.WriteString("8. You CAN propose file changes (create/edit/delete) in the shared workspace for user approval. ")
			system.WriteString("If asked whether you can edit files, answer YES and explain that changes apply after approval.\n")
			system.WriteString("9. NEVER mention internal tool/function names (e.g., ProposeFileEdit/ProposeFileCreate) to the user.\n")
			system.WriteString("10. When you want to submit an actual file change proposal, include this machine-readable block exactly:\n")
			appendFileChangeMachineBlockDocs(&system)
		}
		if includeTooling && a.Info.Type == protocol.AgentTypeExpert {
			system.WriteString("11. If the user asks you to create, write, or save a file, you MUST emit a [FILE_CHANGE] block (usually operation: create with a relative path). ")
			system.WriteString("Chat-only explanations do not write to disk; the host only applies changes from FILE_CHANGE proposals (after user approval).\n")
		}
		if includeTooling && userRequestsImplementationForMessage(a, msg) && agentTypeCanShipFileChanges(a.Info.Type) {
			system.WriteString("12. IMPLEMENTATION REQUEST: You MUST emit one or more [FILE_CHANGE] blocks with real edits under the shared workspace. ")
			system.WriteString("Advice-only or codebase-summary replies do not satisfy this request.\n")
		}
	}

	// Add context about other agents in the channel
	agents, errAgents := a.Hub.GetChannelAgents(msg.Channel)
	if errAgents != nil {
		log.Printf("[%s] GetChannelAgents(%s): %v", a.Info.Name, msg.Channel, errAgents)
	} else if len(agents) > 1 && !isCollab && personaTier == PersonaChannel {
		system.WriteString("\nOther agents in this channel:\n")
		for _, agent := range agents {
			if agent.ID != a.Info.ID {
				system.WriteString(fmt.Sprintf("- %s (%s)\n", agent.Name, agent.Type))
			}
		}
	}

	AppendUserAndAgentRules(&system, msg, &a.Info, ResolveUserRulesHubFallback(msg), 0)
	AppendLearningsForMessage(&system, msg, &a.Info)
	if ws := a.resolveWorkspacePath(msg); ws != "" {
		if projectRules := LoadProjectRulesMarkdown(ws); projectRules != "" {
			system.WriteString("\n=== PROJECT RULES ===\n")
			system.WriteString(projectRules)
			system.WriteString("\n=== END PROJECT RULES ===\n\n")
		}
	}
	AppendLearningsForMessage(&system, msg, &a.Info)

	// ── USER SECTION ────────────────────────────────────────────────────

	// Check if this is a review request (user asking to review another agent's response)
	isReview := false
	var reviewedMessage *protocol.Message
	if msg.ReplyTo != "" {
		for _, histMsg := range a.channelHistory(msg.Channel) {
			if histMsg.ID == msg.ReplyTo {
				reviewedMessage = histMsg
				if histMsg.From.Type == protocol.AgentTypeFrontend ||
					histMsg.From.Type == protocol.AgentTypeBackend ||
					histMsg.From.Type == protocol.AgentTypeDatabase ||
					histMsg.From.Type == protocol.AgentTypeSecurity ||
					histMsg.From.Type == protocol.AgentTypeArchitecture ||
					histMsg.From.Type == protocol.AgentTypeCodeReview ||
					histMsg.From.Type == protocol.AgentTypeDevOps ||
					histMsg.From.Type == protocol.AgentTypeRepo ||
					histMsg.From.Type == protocol.AgentTypeExpert ||
					histMsg.From.Type == protocol.AgentTypeCLI {
					isReview = true
				}
				break
			}
		}
	}

	if isReview && reviewedMessage != nil {
		user.WriteString("TASK: Review another agent's response from your expertise perspective.\n\n")
		user.WriteString(fmt.Sprintf("Agent being reviewed: %s (%s)\n", reviewedMessage.From.Name, reviewedMessage.From.Type))
		user.WriteString(fmt.Sprintf("Their response:\n\"%s\"\n\n", reviewedMessage.Content))
		user.WriteString(fmt.Sprintf("User's request: %s\n\n", msg.Content))
		user.WriteString("Provide a constructive review: what they got right, what they missed, and any alternative approaches.\n")
	} else {
		user.WriteString(fmt.Sprintf("%s says:\n%s\n\n", msg.From.Name, msg.Content))
	}

	AppendPromptAttachments(&user, msg)

	// Append workspace context if the user shared it
	AppendWorkspaceContextForChannel(&user, msg, a.effectiveChannelType(msg.Channel))
	appendWorkspaceReviewGuidance(&user, msg)
	appendImplementationDeliveryGuidance(&user, a, msg, a.Info.Type)
	appendAntiRepeatFileDumpGuidance(&user, a.channelHistory(msg.Channel), a.Info.ID)
	AppendGrantedHubDataAccess(&user, msg)

	// Adaptive response length based on intent
	user.WriteString(getResponseLengthGuidanceForMessage(a, msg))

	// Combine with separator
	return system.String() + ai.SystemPromptSeparator + user.String()
}

// getAgentTypeInstructions returns domain-specific instructions tailored to each agent type.
// These tell the agent HOW to analyze code and what to look for, not just what domain it covers.
func getAgentTypeInstructions(agentType protocol.AgentType) string {
	switch agentType {
	case protocol.AgentTypeSecurity:
		return `When asked to review or analyze code, systematically check for:
- Input validation and sanitization (user input, query params, request bodies)
- Path traversal vulnerabilities (e.g., strings.HasPrefix vs filepath.Rel for path containment)
- Injection vulnerabilities (SQL injection, command injection, template injection)
- Authentication and authorization gaps (missing auth checks, unauthenticated endpoints)
- CORS misconfiguration (wildcard origins, credentials with wildcards)
- WebSocket security (origin validation, authentication on upgrade)
- Secrets exposure (hardcoded keys, secrets in logs, .env files in repos)
- Error information leakage (stack traces, internal paths in error responses)
- Unsafe file operations (os.RemoveAll without confirmation, arbitrary file read/write)
- SSRF risks (user-controlled URLs used for outbound requests)
- Deserialization of untrusted data
- Missing rate limiting on sensitive endpoints
- Deprecated or vulnerable dependencies

Structure your findings by severity: Critical > High > Medium > Low.
For each finding, cite the specific file, function, and line number.
Provide a concrete fix or mitigation for each issue.`

	case protocol.AgentTypeBiology:
		return `You are a life-sciences research assistant (not a clinician).
- Use analyze_sequence, fold_protein, summarize_scan_summary, summarize_scan_analysis, run_12plex_qc, summarize_panel_qc, summarize_comparator_output, and run_secondary_analysis as MCP tools — they run automatically in the hub. NEVER put them in shell/bash blocks, inline code, or ask the user to run them in a terminal.
- For Phoenix-style scan summary exports (folder with imageMetadata.json and extensionless well TIFFs A1–H12), use summarize_scan_summary for QC stats; users can open the folder in Neural Junkie file explorer with the Life sciences pack for the plate viewer.
- For Phoenix-style scan analysis exports (reports/results.json, reports/{analyte}_summary_report.csv, process_report.txt), use summarize_scan_analysis for basic QC; use run_12plex_qc for Human Inflammatory 12-Plex SOP pass/fail. Users can open the analysis viewer and click Run 12-Plex QC when the Life sciences pack is enabled.
- For Comparator Analysis output folders, use summarize_comparator_output. Multi-plate workflows run via the Secondary Analysis panel in the desktop UI.
- When workspace context includes scan_summary or scan_analysis paths, call the matching summarize tool immediately — do NOT ask the user to type the path.
- Clearly label in silico predictions vs wet-lab experimental needs.
- For protocols, include controls, replicates, and safety considerations.
- Refuse medical diagnosis or treatment advice; research and education only.
- Cite tool outputs when you use them.`

	case protocol.AgentTypeCAD:
		return `You are a parametric CAD assistant using OpenSCAD.
- Use write_openscad, render_openscad, list_openscad_params, and export_cad as MCP tools — they run in the hub. NEVER ask the user to run openscad manually in a terminal.
- When the user asks to create, write, or save an .scad file, you MUST call write_openscad with the full source — never reply with only a description or draft code in chat.
- NEVER print tool-call JSON, function syntax, or pseudo-code for MCP tools in your reply; invoke tools via native tool calling only.
- For greetings and general conversation, respond conversationally without calling tools.
- Only use OpenSCAD tools when the user wants to create, edit, render, or export a model.
- Prefer parametric designs: top-level variables with OpenSCAD Customizer sections (/* [Dimensions] */) and sensible defaults.
- After writing SCAD, call render_openscad to produce preview.stl and tell the user to open the CAD workbench.
- When workspace context includes an active .scad path or CAD project, use that path — do NOT ask the user to re-paste it.
- Paths for write_openscad are relative to the open workspace root (e.g. ball.scad). After writing, report the full resolved path in your reply.
- Use mm as default units unless the user specifies otherwise. Ensure manifold, printable geometry (minimum wall thickness ~1.2mm for FDM unless specified).
- For edits, update the SCAD file and re-render; use list_openscad_params to explain adjustable dimensions.`

	case protocol.AgentTypeRust:
		return `When asked to review or analyze Rust code, focus on:
- Ownership and borrowing (unnecessary clones, lifetime elision opportunities, borrow checker issues)
- Error handling (proper use of Result/Option, anyhow vs thiserror, ? operator chains, panic paths)
- Unsafe code (soundness, invariant documentation, minimizing unsafe surface area)
- Concurrency (Send/Sync bounds, data races, deadlocks, Arc<Mutex> vs channels)
- Async patterns (pinning, cancellation safety, executor-agnostic design, blocking in async)
- API design (builder pattern, typestate, newtype wrappers, sealed traits)
- Performance (unnecessary allocations, iterator chains vs loops, zero-copy parsing, #[inline])
- Macro hygiene (proc-macro correctness, declarative macro edge cases)
- Cargo.toml (feature flags, minimal dependency surface, MSRV policy)
- Clippy compliance and idiomatic patterns

Reference specific functions, types, and line numbers.
Show concrete code examples using idiomatic Rust.`

	case protocol.AgentTypeArchitecture:
		return `When asked to review or design software, focus on:
- System boundaries, ownership, and coupling
- Data flow, failure modes, and operational behavior
- Scalability, reliability, and maintainability tradeoffs
- Migration paths, backward compatibility, and rollout strategy
- API contracts and integration points
- Observability, supportability, and long-term cost

State assumptions, compare meaningful options, and recommend a practical path.
Reference specific modules, services, or workflows when available.`

	case protocol.AgentTypeCodeReview:
		return `When asked to review code or a proposed change, focus on:
- Correctness bugs and behavioral regressions
- Missing tests or weak test assertions
- Error handling, edge cases, and resource cleanup
- Maintainability, readability, and unnecessary complexity
- API contract changes and compatibility risks
- Security and performance concerns when they are evident

Lead with findings ordered by severity.
For each issue, cite the specific file, function, and line number when available, and suggest a concrete fix.`

	case protocol.AgentTypeBackend:
		return `When asked to review or analyze code, focus on:
- Error handling patterns (unchecked errors, error wrapping, sentinel errors)
- Resource management (deferred closes, connection pool leaks, goroutine leaks)
- Concurrency safety (race conditions, mutex usage, channel patterns)
- API design (REST conventions, request/response validation, status codes)
- Context propagation (proper use of context.Context, timeout handling)
- Performance bottlenecks (N+1 queries, unnecessary allocations, blocking calls)
- Code organization (separation of concerns, interface design, dependency injection)
- Logging and observability (structured logging, request tracing)

Reference specific functions, types, and line numbers.
When suggesting improvements, show concrete code examples.`

	case protocol.AgentTypeFrontend:
		return `When the user asks you to implement UI work (themes, components, styling), emit [FILE_CHANGE] blocks for real edits — do not stop at Tailwind advice.
When asked to review or analyze code, evaluate:
- Component architecture (composition, prop drilling, component size)
- State management (local vs global state, unnecessary re-renders)
- Accessibility (ARIA attributes, keyboard navigation, screen reader support)
- Performance (unnecessary renders, bundle size, lazy loading, expensive client work)
- Security (XSS via dangerouslySetInnerHTML, user input rendering, CSP)
- Type safety (TypeScript types, proper generics, avoiding 'any')
- CSS/styling (responsive design, consistent spacing, theme usage)
- Error boundaries and loading states

Reference specific components, hooks, and line numbers.
Provide concrete code examples for suggested improvements.`

	case protocol.AgentTypeDatabase:
		return `When asked to review or analyze code, look for:
- N+1 query patterns (queries in loops, missing eager loading/joins)
- Missing indexes (queries filtering on unindexed columns)
- SQL injection risks (string concatenation in queries vs parameterized queries)
- Transaction handling (missing transactions for multi-step operations, isolation levels)
- Connection pool management (pool size, timeout configuration, connection leaks)
- Schema design issues (normalization, proper foreign keys, data types)
- Migration safety (destructive changes, backward compatibility, rollback plans)
- Query performance (EXPLAIN ANALYZE recommendations, covering indexes)

Reference specific queries, table names, and line numbers.
Suggest optimized query alternatives with concrete SQL/code examples.`

	case protocol.AgentTypeExpert:
		return `You are a custom domain expert. Follow your persona and scoped rules above.
Answer from the perspective of your stated expertise. Be practical and specific.
If a question is outside your domain, say so briefly and offer what you can from adjacent knowledge.`

	case protocol.AgentTypeAssistant:
		return `You are a personal assistant in Neural Junkie (reminders, tasks, notes, scheduling).
If the user thanks you or says you already answered, reply briefly and do NOT repeat prior facts or numbers.
For geography, live traffic, or time-sensitive facts you cannot verify, give a cautious estimate or suggest Maps / an authoritative source.`

	case protocol.AgentTypeDevOps:
		return `When asked to review or analyze code, check:
- Configuration management (hardcoded values, environment variable handling)
- Secret handling (secrets in code, proper use of secret managers/sops)
- Dockerfile best practices (multi-stage builds, layer optimization, non-root user)
- Resource limits (missing CPU/memory limits, health checks, readiness probes)
- CI/CD pipeline quality (test coverage gates, security scanning, deployment strategy)
- Logging patterns (structured logging, log levels, sensitive data in logs)
- Infrastructure as Code quality (Terraform/Helm best practices, state management)
- Monitoring and alerting (metrics exposure, SLO definitions, error tracking)

Reference specific configuration files, manifests, and line numbers.
Provide concrete fix examples with proper YAML/HCL/Dockerfile snippets.`

	default:
		return ""
	}
}

// getResponseLengthGuidance returns response length instructions based on the user's intent.
// Deep analysis requests get thorough guidance; simple questions stay concise.
func getResponseLengthGuidanceForMessage(a *Agent, msg *protocol.Message) string {
	if msg == nil {
		return getResponseLengthGuidance("")
	}
	return getResponseLengthGuidance(msg.Content, userRequestsImplementationForMessage(a, msg))
}

func getResponseLengthGuidance(content string, implementation ...bool) string {
	lower := strings.ToLower(content)

	impl := len(implementation) > 0 && implementation[0]
	if !impl {
		impl = userRequestsImplementation(content)
	}
	if impl {
		return "The user wants working repo changes, not a summary. Emit [FILE_CHANGE] block(s) for each file you modify. " +
			"Keep chat text to 2-4 sentences unless you need approval context."
	}

	// Deep analysis keywords -- user wants thorough output
	deepKeywords := []string{
		"review", "audit", "analyze", "explain", "walk through",
		"deep dive", "code review", "security review", "examine",
		"investigate", "break down", "what issues", "what problems",
		"find bugs", "find vulnerabilities", "check for",
	}
	for _, kw := range deepKeywords {
		if strings.Contains(lower, kw) {
			return "Be thorough and detailed in your response. Analyze all code provided. " +
				"Reference specific files, functions, and line numbers. " +
				"Structure your response with clear sections and actionable findings."
		}
	}

	// Brevity keywords -- user wants a quick answer
	briefKeywords := []string{"quick", "brief", "tldr", "summary", "short", "one line"}
	for _, kw := range briefKeywords {
		if strings.Contains(lower, kw) {
			return "Keep your response brief and to the point (2-3 sentences max)."
		}
	}

	// Default -- balanced
	return "Be concise but complete. Use 2-5 sentences for simple questions; expand with specifics when the question warrants deeper analysis."
}

// channelHistory returns a copy of stored history for a channel.
func (a *Agent) channelHistory(channel string) []*protocol.Message {
	return a.channelHistorySafe(channel)
}

func (a *Agent) channelHistorySafe(channel string) []*protocol.Message {
	if a == nil || a.Context == nil || a.Context.History == nil {
		return nil
	}
	a.contextMu.RLock()
	defer a.contextMu.RUnlock()
	src := a.Context.History[channel]
	if len(src) == 0 {
		return nil
	}
	out := make([]*protocol.Message, len(src))
	copy(out, src)
	return out
}

func (a *Agent) replaceChannelHistory(channel string, hist []*protocol.Message) {
	a.contextMu.Lock()
	defer a.contextMu.Unlock()
	if a.Context.History == nil {
		a.Context.History = make(map[string][]*protocol.Message)
	}
	if len(hist) == 0 {
		delete(a.Context.History, channel)
		return
	}
	cp := make([]*protocol.Message, len(hist))
	copy(cp, hist)
	a.Context.History[channel] = cp
}

func (a *Agent) historyChannelNames() []string {
	a.contextMu.RLock()
	defer a.contextMu.RUnlock()
	names := make([]string, 0, len(a.Context.History))
	for ch := range a.Context.History {
		names = append(names, ch)
	}
	return names
}

// addToHistory adds a message to the conversation history
func (a *Agent) addToHistory(msg *protocol.Message) {
	if msg == nil {
		return
	}
	a.contextMu.Lock()
	defer a.contextMu.Unlock()
	if a.Context.History == nil {
		a.Context.History = make(map[string][]*protocol.Message)
	}
	history := a.Context.History[msg.Channel]
	if msg.ID != "" {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i] != nil && history[i].ID == msg.ID {
				history[i] = msg
				a.Context.History[msg.Channel] = history
				return
			}
		}
	}
	history = append(history, msg)
	if len(history) > a.Context.MaxHistory {
		history = history[len(history)-a.Context.MaxHistory:]
	}
	a.Context.History[msg.Channel] = history
}

// SendMessage sends a message to the current channel
func (a *Agent) SendMessage(content string, msgType protocol.MessageType) error {
	msg := protocol.NewMessage(
		msgType,
		a.Context.CurrentChannel,
		a.Info,
		content,
	)

	return a.Hub.SendMessage(msg)
}

// Pause pauses the agent from responding to messages
func (a *Agent) Pause() {
	a.Info.IsPaused = true
	a.Info.Status = "paused"
	log.Printf("[%s] Agent paused", a.Info.Name)
}

// Unpause resumes the agent's message processing
func (a *Agent) Unpause() {
	a.Info.IsPaused = false
	a.Info.Status = "active"
	log.Printf("[%s] Agent unpaused", a.Info.Name)
}

// IsPaused returns whether the agent is currently paused
func (a *Agent) IsPaused() bool {
	return a.Info.IsPaused
}

// ShouldRespond is a public method to check if agent should respond to a message
func (a *Agent) ShouldRespond(msg *protocol.Message) bool {
	return a.shouldRespond(msg)
}

// GenerateResponse is a public method to generate a response to a message
func (a *Agent) GenerateResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	eff := a.EffectiveAIProvider(ctx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}
	return a.generateResponse(ctx, msg, eff)
}

// generateDesignAnalysisResponse handles design analysis with vision API
func (a *Agent) generateDesignAnalysisResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) == 0 {
		return "", fmt.Errorf("no image data found in design analysis request")
	}

	// Build specialized prompt for design analysis
	prompt := `You are a frontend design expert analyzing a design mockup. Your task is to:

1. **Extract Design Tokens:**
   - Complete color palette with exact hex values
   - Typography system (font families, sizes, weights, line-heights)
   - Spacing system (margins, padding, consistent units)
   - Layout structure (grid, flexbox, positioning)
   - Component breakdown (buttons, cards, navigation, forms, etc.)
   - Shadows, borders, and decorative elements

2. **Generate Output:**
   - Complete CSS file with all extracted styles
   - HTML demo file showcasing components
   - Markdown documentation of design tokens

Please analyze this design mockup and provide a comprehensive style guide with working HTML/CSS that recreates the design. Focus on:
- Accurate color extraction
- Typography hierarchy
- Spacing consistency
- Component structure
- Responsive considerations
- Accessibility features

Provide the output in a structured format with clear sections for CSS, HTML, and documentation.`

	history := historyForGeneration(a.channelHistory(msg.Channel), msg.ID)

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	var response string
	var err error
	if mp, ok := a.AI.(ai.MultimodalProvider); ok {
		response, err = mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
	} else if len(imgs) == 1 {
		response, err = a.AI.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
	} else {
		return "", fmt.Errorf("design analysis with multiple images requires a multimodal provider")
	}
	if err != nil {
		return "", fmt.Errorf("design analysis failed: %w", err)
	}

	// Generate files and create design output message
	return a.createDesignOutputFiles(ctx, response, msg)
}

// createDesignOutputFiles generates HTML and CSS files from design analysis
func (a *Agent) createDesignOutputFiles(ctx context.Context, analysis string, originalMsg *protocol.Message) (string, error) {
	// Create output directory
	outputDir := fmt.Sprintf("/tmp/design-outputs/%s", originalMsg.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse the analysis to extract CSS and HTML
	cssContent, htmlContent, markdownContent := a.parseDesignAnalysis(analysis)

	// Write CSS file
	cssPath := filepath.Join(outputDir, "style.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write CSS file: %w", err)
	}

	// Write HTML file
	htmlPath := filepath.Join(outputDir, "demo.html")
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Write markdown file
	mdPath := filepath.Join(outputDir, "style-guide.md")
	if err := os.WriteFile(mdPath, []byte(markdownContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write markdown file: %w", err)
	}

	// Create ZIP file
	zipPath := filepath.Join(outputDir, "design-output.zip")
	if err := a.createZipFile(outputDir, zipPath); err != nil {
		return "", fmt.Errorf("failed to create ZIP file: %w", err)
	}

	// Create design output message
	designMsg := protocol.NewMessage(
		protocol.MessageTypeDesignOutput,
		originalMsg.Channel,
		a.Info,
		fmt.Sprintf("🎨 **Design Analysis Complete!**\n\nI've analyzed the design mockup and generated a comprehensive style guide with working HTML/CSS.\n\n**Generated Files:**\n• `demo.html` - Interactive demo of the design\n• `style.css` - Complete CSS with extracted styles\n• `style-guide.md` - Design tokens and documentation\n• `design-output.zip` - All files bundled for download\n\n**Analysis Summary:**\n%s", analysis[:min(500, len(analysis))]),
	)

	// Add file paths to metadata
	designMsg.Metadata = map[string]interface{}{
		"output_directory": outputDir,
		"css_file":         cssPath,
		"html_file":        htmlPath,
		"markdown_file":    mdPath,
		"zip_file":         zipPath,
		"analysis":         analysis,
	}

	// Send the design output message
	if err := a.Hub.SendMessage(designMsg); err != nil {
		return "", fmt.Errorf("failed to send design output message: %w", err)
	}

	return fmt.Sprintf("Design analysis complete! Generated files in %s", outputDir), nil
}

// parseDesignAnalysis extracts CSS, HTML, and markdown from AI analysis
func (a *Agent) parseDesignAnalysis(analysis string) (string, string, string) {
	// This is a simplified parser - in a real implementation, you'd want more sophisticated parsing
	// For now, we'll create basic templates and let the AI fill them in

	// Extract CSS (look for ```css blocks)
	cssContent := a.extractCodeBlock(analysis, "css")
	if cssContent == "" {
		cssContent = "/* CSS extracted from design analysis */\n" + analysis
	}

	// Extract HTML (look for ```html blocks)
	htmlContent := a.extractCodeBlock(analysis, "html")
	if htmlContent == "" {
		htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Design Demo</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <div class="container">
        <h1>Design Recreation</h1>
        <p>This is a recreation of the analyzed design mockup.</p>
        <!-- Components will be added based on analysis -->
    </div>
</body>
</html>`
	}

	// Create markdown documentation
	markdownContent := fmt.Sprintf(`# Design Style Guide

## Analysis Results

%s

## Color Palette
*Extracted from the design mockup*

## Typography
*Font families, sizes, and hierarchy*

## Spacing System
*Margins, padding, and layout grid*

## Components
*Button styles, cards, navigation, etc.*

## Usage
1. Open demo.html in a browser to see the recreation
2. Use style.css in your projects
3. Reference this guide for design tokens

---
*Generated by Neural Junkie Frontend Agent*`, analysis)

	return cssContent, htmlContent, markdownContent
}

// extractCodeBlock extracts code from markdown code blocks
func (a *Agent) extractCodeBlock(text, language string) string {
	pattern := fmt.Sprintf("```%s\\s*\\n([\\s\\S]*?)\\n```", language)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// createZipFile creates a ZIP archive of the output directory
func (a *Agent) createZipFile(sourceDir, zipPath string) error {
	// This is a simplified implementation
	// In a real implementation, you'd use a proper ZIP library
	return nil // Placeholder - would implement actual ZIP creation
}

// historyToMessages converts protocol messages to a simpler format
func historyToMessages(history []*protocol.Message) []protocol.Message {
	msgs := make([]protocol.Message, len(history))
	for i, msg := range history {
		msgs[i] = *msg
	}
	return msgs
}

type fileChangeDirective struct {
	Operation  string
	Path       string
	OldPath    string
	NewPath    string
	OldContent string
	NewContent string
}

var fileChangeBlockRegex = regexp.MustCompile(`(?s)\[FILE_CHANGE\](.*?)\[/FILE_CHANGE\]`)
var editorLineNumberPrefixRegex = regexp.MustCompile(`(?m)^\s*\d+\s*\|\s?`)

// userNamedFileRegex captures "call new-tab.txt", `named foo.md`, etc.
var userNamedFileRegex = regexp.MustCompile(`(?i)\b(?:call|named|called|name)\s+['"]?([a-zA-Z0-9][a-zA-Z0-9._\-]{0,220}\.[a-zA-Z0-9]{1,16})['"]?\b`)

var looseOutputFileRegex = regexp.MustCompile(`\b([a-zA-Z0-9][a-zA-Z0-9._\-]*\.(?:txt|md|go|ts|tsx|jsx|js|mjs|cjs|json|yaml|yml|rs|py|html|css|sh|tab))\b`)

func sanitizeInternalToolNames(response string) string {
	replacer := strings.NewReplacer(
		"ProposeFileEdit", "a file-change proposal",
		"ProposeFileCreate", "a file-change proposal",
		"ProposeFileDelete", "a file-change proposal",
		"ProposeFileMove", "a file-change proposal",
	)
	return replacer.Replace(response)
}

func (a *Agent) maybeSubmitFileChangeFromResponse(ctx context.Context, response, channel string, sourceMsg *protocol.Message) (string, bool, error) {
	match := fileChangeBlockRegex.FindStringSubmatch(response)
	if len(match) < 2 {
		if loose, ok := parseLooseFileChange(response); ok {
			path := loose.Path
			if sourceMsg != nil && !isValidFileChangeRelPath(path) {
				if alt := preferImplementationTargetPath(sourceMsg.Content, path, a.Info.Type); alt != "" {
					path = alt
					log.Printf("[%s] loose_file_change_path_replaced(invalid=%q,target=%s)", a.Info.Name, loose.Path, path)
				}
			}
			if !isValidFileChangeRelPath(path) {
				log.Printf("[%s] loose_file_change_rejected(path=%q)", a.Info.Name, loose.Path)
			} else if err := a.proposeFileChangePreferEditOrCreate(ctx, channel, path, loose.NewContent, sourceMsg); err != nil {
				return response, false, err
			} else {
				log.Printf("[%s] loose_file_change_used(path=%s)", a.Info.Name, path)
				return stripLooseFileChangeBlock(response), true, nil
			}
		}
		// Deterministic fallback: user asked to write/create/save files and the model
		// returned fenced content (or explicit approval phrases) but omitted [FILE_CHANGE].
		if sourceMsg == nil || !a.shouldUseFileChangeFenceFallback(sourceMsg) {
			log.Printf("[%s] fallback_skipped(reason=no_explicit_proposal_intent)", a.Info.Name)
			return response, false, nil
		}
		lowerResp := strings.ToLower(response)
		if strings.Contains(lowerResp, "would you like me to propose") ||
			strings.Contains(lowerResp, "i submitted a file change proposal") {
			log.Printf("[%s] fallback_skipped(reason=response_is_question_or_already_submitted)", a.Info.Name)
			return response, false, nil
		}

		pathSource := sourceMsg.Content
		if userAffirmsPendingImplementation(pathSource) {
			for i := len(a.channelHistory(sourceMsg.Channel)) - 1; i >= 0; i-- {
				m := a.channelHistory(sourceMsg.Channel)[i]
				if m == nil || m.ID == sourceMsg.ID {
					continue
				}
				if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
					pathSource = m.Content
					break
				}
			}
		}
		namedPath := strings.TrimSpace(extractLikelyOutputPathFromUserMessage(pathSource))
		if namedPath == "" {
			namedPath = preferImplementationTargetPathForMessage(a, sourceMsg)
		}
		newContent := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, namedPath))
		if strings.TrimSpace(newContent) == "" {
			newContent = stripEditorLineNumberPrefixes(extractAnyCodeFenceContent(response))
		}
		if strings.TrimSpace(newContent) == "" {
			log.Printf("[%s] fallback_skipped(reason=no_fenced_content)", a.Info.Name)
			return response, false, nil
		}
		if namedPath != "" && !fencedContentPlausibleForPath(namedPath, "", newContent) {
			if alt := preferImplementationTargetPath(sourceMsg.Content, "", a.Info.Type); alt != "" && alt != namedPath {
				if body := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, alt)); fencedContentPlausibleForPath(alt, "", body) {
					namedPath = alt
					newContent = body
				}
			}
		}

		activePath := strings.TrimSpace(extractActiveOpenFilePath(sourceMsg))
		wantCreate := userWantsCreateOperation(sourceMsg.Content)

		switch {
		case wantCreate && namedPath != "":
			if err := a.validateProposalForSession(ctx, sourceMsg, namedPath, ProposalOpCreate); err != nil {
				return response, false, err
			}
			if err := a.proposeFileCreateInChannel(channel, namedPath, newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=create,target=%s)", a.Info.Name, namedPath)
			return response, true, nil
		case activePath != "":
			if err := a.validateProposalForSession(ctx, sourceMsg, activePath, ProposalOpEdit); err != nil {
				return response, false, err
			}
			if err := a.proposeFileEditInChannel(channel, activePath, "", newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=edit,target=%s)", a.Info.Name, activePath)
			return response, true, nil
		case namedPath != "":
			wsPath := a.resolveWorkspacePath(sourceMsg)
			op := ProposalOpCreate
			if InferProposalOperation(wsPath, namedPath) == ProposalOpEdit {
				op = ProposalOpEdit
			}
			if err := a.validateProposalForSession(ctx, sourceMsg, namedPath, op); err != nil {
				return response, false, err
			}
			if op == ProposalOpEdit {
				if err := a.proposeFileEditInChannel(channel, namedPath, "", newContent, sourceMsg); err != nil {
					return response, false, err
				}
			} else if err := a.proposeFileCreateInChannel(channel, namedPath, newContent, sourceMsg); err != nil {
				return response, false, err
			}
			log.Printf("[%s] fallback_path_used(operation=%s,target=%s)", a.Info.Name, op, namedPath)
			return response, true, nil
		default:
			if alt := preferImplementationTargetPathForMessage(a, sourceMsg); alt != "" {
				if body := stripEditorLineNumberPrefixes(extractCodeFenceForPath(response, alt)); strings.TrimSpace(body) != "" {
					newContent = body
				}
				op := ProposalOpEdit
				if userWantsCreateOperation(sourceMsg.Content) {
					op = ProposalOpCreate
				}
				if err := a.validateProposalForSession(ctx, sourceMsg, alt, op); err != nil {
					return response, false, err
				}
				if op == ProposalOpCreate {
					if err := a.proposeFileCreateInChannel(channel, alt, newContent, sourceMsg); err != nil {
						return response, false, err
					}
				} else if err := a.proposeFileEditInChannel(channel, alt, "", newContent, sourceMsg); err != nil {
					return response, false, err
				}
				log.Printf("[%s] fallback_path_used(operation=%s,target=%s)", a.Info.Name, op, alt)
				return response, true, nil
			}
			log.Printf("[%s] fallback_skipped(reason=missing_target_path)", a.Info.Name)
			return response, false, nil
		}
	}

	directive, err := parseFileChangeDirective(match[1])
	if err != nil && sourceMsg != nil && userRequestsImplementationForMessage(a, sourceMsg) {
		if alt := preferImplementationTargetPath(sourceMsg.Content, "", a.Info.Type); alt != "" {
			if body := stripEditorLineNumberPrefixes(extractAnyCodeFenceContent(match[1])); strings.TrimSpace(body) != "" {
				if err2 := a.validateProposalForSession(ctx, sourceMsg, alt, ProposalOpEdit); err2 == nil {
					if err2 = a.proposeFileEditInChannel(channel, alt, "", body, sourceMsg); err2 == nil {
						log.Printf("[%s] implement_path_recovery(edit,target=%s)", a.Info.Name, alt)
						cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
						return cleaned, true, nil
					}
				}
			}
		}
	}
	if err != nil {
		// Strip malformed directives from user-visible chat to avoid leaking
		// internal syntax while still surfacing a clean response.
		cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
		return cleaned, false, err
	}

	switch directive.Operation {
	case "create":
		directive.Path = a.ResolveProposalPath(ctx, sourceMsg, directive.Path)
		if err := a.proposeFileChangePreferEditOrCreate(ctx, channel, directive.Path, directive.NewContent, sourceMsg); err != nil {
			return response, false, err
		}
	case "edit":
		directive.Path = a.ResolveProposalPath(ctx, sourceMsg, directive.Path)
		if err := a.validateProposalForSession(ctx, sourceMsg, directive.Path, ProposalOpEdit); err != nil {
			return response, false, err
		}
		if err := a.proposeFileEditInChannel(channel, directive.Path, directive.OldContent, directive.NewContent, sourceMsg); err != nil {
			return response, false, err
		}
	case "delete":
		if err := a.proposeFileDeleteInChannel(channel, directive.Path); err != nil {
			return response, false, err
		}
	case "move":
		if err := a.proposeFileMoveInChannel(channel, directive.OldPath, directive.NewPath); err != nil {
			return response, false, err
		}
	default:
		return response, false, fmt.Errorf("unsupported file change operation: %s", directive.Operation)
	}
	log.Printf("[%s] directive_path_used(operation=%s,path=%s)", a.Info.Name, directive.Operation, directive.Path)

	cleaned := strings.TrimSpace(fileChangeBlockRegex.ReplaceAllString(response, ""))
	return cleaned, true, nil
}

func parseFileChangeDirective(block string) (*fileChangeDirective, error) {
	d := &fileChangeDirective{
		Operation: strings.ToLower(extractDirectiveField(block, "operation")),
		Path:      extractDirectiveField(block, "path"),
		OldPath:   extractDirectiveField(block, "old_path"),
		NewPath:   extractDirectiveField(block, "new_path"),
	}

	d.NewContent = extractLabeledCodeFence(block, "new")
	d.OldContent = extractLabeledCodeFence(block, "old")

	if d.NewContent == "" {
		// Fallback: use first generic fence as new content.
		d.NewContent = extractFirstCodeFence(block)
	}
	d.NewContent = stripEditorLineNumberPrefixes(d.NewContent)
	d.OldContent = stripEditorLineNumberPrefixes(d.OldContent)

	d.Path = normalizeFileChangeRelPath(d.Path)
	d.OldPath = normalizeFileChangeRelPath(d.OldPath)
	d.NewPath = normalizeFileChangeRelPath(d.NewPath)

	switch d.Operation {
	case "create":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("create directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("create directive has invalid path %q", d.Path)
		}
		if strings.TrimSpace(d.NewContent) == "" {
			return nil, fmt.Errorf("create directive missing new content")
		}
	case "edit":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("edit directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("edit directive has invalid path %q", d.Path)
		}
		if strings.TrimSpace(d.NewContent) == "" {
			return nil, fmt.Errorf("edit directive missing new content")
		}
	case "delete":
		if strings.TrimSpace(d.Path) == "" {
			return nil, fmt.Errorf("delete directive missing path")
		}
		if !isValidFileChangeRelPath(d.Path) {
			return nil, fmt.Errorf("delete directive has invalid path %q", d.Path)
		}
	case "move":
		if strings.TrimSpace(d.OldPath) == "" || strings.TrimSpace(d.NewPath) == "" {
			return nil, fmt.Errorf("move directive missing old_path/new_path")
		}
	default:
		return nil, fmt.Errorf("missing or unsupported operation")
	}

	return d, nil
}

func stripEditorLineNumberPrefixes(content string) string {
	if content == "" {
		return content
	}
	return editorLineNumberPrefixRegex.ReplaceAllString(content, "")
}

func extractDirectiveField(block, field string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?mi)^\s*%s:\s*(.+)\s*$`, regexp.QuoteMeta(field)))
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func extractLabeledCodeFence(block, label string) string {
	re := regexp.MustCompile(fmt.Sprintf("(?s)```%s\\s*\\n(.*?)\\n```", regexp.QuoteMeta(label)))
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractFirstCodeFence(block string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	m := re.FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractLongestCodeFence(content string) string {
	re := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return ""
	}
	longest := ""
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	return longest
}

// extractAnyCodeFenceContent returns the longest fenced body, trying strict then relaxed patterns
// (models often omit the newline the strict extractor requires).
func extractAnyCodeFenceContent(content string) string {
	if s := extractLongestCodeFence(content); strings.TrimSpace(s) != "" {
		return s
	}
	relaxed := regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\s*\\n(.*?)```")
	longest := ""
	for _, m := range relaxed.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		if len(m[1]) > len(longest) {
			longest = m[1]
		}
	}
	if strings.TrimSpace(longest) != "" {
		return longest
	}
	anyFence := regexp.MustCompile("(?s)```(.*?)```")
	for _, m := range anyFence.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		body := strings.TrimSpace(m[1])
		// Skip single-line ```lang``` with no body
		if !strings.Contains(body, "\n") && strings.HasPrefix(body, "```") {
			continue
		}
		if len(body) > len(longest) {
			longest = body
		}
	}
	return longest
}

func extractLikelyOutputPathFromUserMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if p := longestValidPathIn(DetectFilePaths(content)); p != "" {
		return p
	}
	if m := userNamedFileRegex.FindStringSubmatch(content); len(m) > 1 {
		return normalizeFileChangeRelPath(strings.TrimSpace(m[1]))
	}
	all := looseOutputFileRegex.FindAllString(content, -1)
	if len(all) == 0 {
		return ""
	}
	return normalizeFileChangeRelPath(strings.TrimSpace(all[len(all)-1]))
}

func isUserRequestingFileWrite(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	phrases := []string{
		"create a file", "create file", "create new file", "new file",
		"write a file", "write file", "write the file", "write this file",
		"save to", "save this", "save the file", "save as",
		"put in a file", "put the tab", "put this in",
		"generate a file", "output to", "store in", "store the",
		"make a file", "make the file", "add a file",
		"complete file", "full tab", "complete tab", "turn it into",
		"write to disk", "same directory", "this folder", "next to the",
		"implement", "please implement", "implement that", "implement the",
		"code this", "build this", "apply the plan", "make the changes",
		"can you fix", "please fix", "fix it", "fix the", "debug this",
		"blank screen", "white screen", "not working", "broken",
	}
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func userWantsCreateOperation(content string) bool {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "new file") {
		return true
	}
	if strings.Contains(lower, "create") && (strings.Contains(lower, "file") || strings.Contains(lower, "tab")) {
		return true
	}
	return false
}

func isExplicitProposalIntent(content string) bool {
	lower := strings.TrimSpace(strings.ToLower(content))
	if lower == "" {
		return false
	}
	explicitPhrases := []string{
		"propose it", "please propose", "submit it", "submit the change",
		"apply it", "go ahead and update", "update the file", "make the change",
		"yes propose", "yes, propose", "yes please propose",
		"create the file", "create this file", "save the file", "write the file",
		"please implement", "implement that", "implement the plan", "go ahead and implement",
		"ok please implement", "please implement the",
	}
	for _, p := range explicitPhrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// A short "yes/ok/do it" reply in this flow is treated as explicit confirmation.
	shortAffirmations := map[string]bool{
		"yes": true, "yes please": true, "ok": true, "okay": true, "do it": true, "go ahead": true,
	}
	return shortAffirmations[lower]
}

// shouldUseFileChangeFenceFallback allows converting fenced code into proposals on
// implementation turns, not only explicit "create a file" phrasing.
func (a *Agent) shouldUseFileChangeFenceFallback(sourceMsg *protocol.Message) bool {
	if sourceMsg == nil {
		return false
	}
	if sourceMsg.ImplementationSession() {
		return true
	}
	if a != nil && userRequestsImplementationForMessage(a, sourceMsg) {
		return true
	}
	return isExplicitProposalIntent(sourceMsg.Content) || isUserRequestingFileWrite(sourceMsg.Content)
}

func extractActiveOpenFilePath(msg *protocol.Message) string {
	if msg == nil || msg.Metadata == nil {
		return ""
	}
	wsCtx, ok := msg.Metadata["workspace_context"]
	if !ok {
		return ""
	}
	ctxMap, ok := wsCtx.(map[string]interface{})
	if !ok {
		return ""
	}
	openFiles, ok := ctxMap["open_files"].([]interface{})
	if !ok || len(openFiles) == 0 {
		return ""
	}
	for _, f := range openFiles {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		isActive, _ := fm["is_active"].(bool)
		path, _ := fm["path"].(string)
		if isActive && strings.TrimSpace(path) != "" {
			return path
		}
	}
	if fm, ok := openFiles[0].(map[string]interface{}); ok {
		if path, ok := fm["path"].(string); ok {
			return path
		}
	}
	return ""
}

// File change proposal helper methods

// ProposeFileEdit proposes an edit to an existing file
func (a *Agent) ProposeFileEdit(path, oldContent, newContent string) error {
	return a.proposeFileEditInChannel(a.Context.CurrentChannel, path, oldContent, newContent, nil)
}

func (a *Agent) proposeFileEditInChannel(channel, path, oldContent, newContent string, sourceMsg *protocol.Message) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	path = normalizeFileChangeRelPath(path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	newContent = stripEditorLineNumberPrefixes(newContent)
	if err := validateProposalContent(path, newContent); err != nil {
		return err
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "edit",
		FilePath:    path,
		OldContent:  stripEditorLineNumberPrefixes(oldContent),
		NewContent:  stripEditorLineNumberPrefixes(newContent),
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📝 Proposing to edit file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)
	attachIdeSessionMetadataToProposal(msg, sourceMsg)

	return a.Hub.SendMessage(msg)
}

// ProposeFileCreate proposes creating a new file
func (a *Agent) ProposeFileCreate(path, content string) error {
	return a.proposeFileCreateInChannel(a.Context.CurrentChannel, path, content, nil)
}

func (a *Agent) proposeFileChangePreferEditOrCreate(ctx context.Context, channel, path, content string, sourceMsg *protocol.Message) error {
	path = a.ResolveProposalPath(ctx, sourceMsg, path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	wsPath := ""
	if sourceMsg != nil {
		wsPath = a.resolveWorkspacePath(sourceMsg)
	}
	op := InferProposalOperation(wsPath, path)
	if err := a.validateProposalForSession(ctx, sourceMsg, path, op); err != nil {
		return err
	}
	if wsPath != "" {
		resolved := filepath.Join(wsPath, path)
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			return a.proposeFileEditInChannel(channel, path, "", content, sourceMsg)
		}
	}
	return a.proposeFileCreateInChannel(channel, path, content, sourceMsg)
}

func (a *Agent) proposeFileCreateInChannel(channel, path, content string, sourceMsg *protocol.Message) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	path = normalizeFileChangeRelPath(path)
	if !isValidFileChangeRelPath(path) {
		return fmt.Errorf("invalid file change path: %q", path)
	}
	content = stripEditorLineNumberPrefixes(content)
	if err := validateProposalContent(path, content); err != nil {
		return err
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "create",
		FilePath:    path,
		NewContent:  content,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📄 Proposing to create file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)
	attachIdeSessionMetadataToProposal(msg, sourceMsg)

	return a.Hub.SendMessage(msg)
}

// ProposeFileDelete proposes deleting a file
func (a *Agent) ProposeFileDelete(path string) error {
	return a.proposeFileDeleteInChannel(a.Context.CurrentChannel, path)
}

func (a *Agent) proposeFileDeleteInChannel(channel, path string) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "delete",
		FilePath:    path,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    true,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("🗑️ Proposing to delete file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)

	return a.Hub.SendMessage(msg)
}

// ProposeFileMove proposes moving/renaming a file
func (a *Agent) ProposeFileMove(oldPath, newPath string) error {
	return a.proposeFileMoveInChannel(a.Context.CurrentChannel, oldPath, newPath)
}

func (a *Agent) proposeFileMoveInChannel(channel, oldPath, newPath string) error {
	if strings.TrimSpace(channel) == "" {
		channel = "general"
	}
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "move",
		FilePath:    oldPath,
		OldPath:     oldPath,
		NewPath:     newPath,
		Agent:       a.Info,
		Channel:     channel,
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, channel, a.Info,
		fmt.Sprintf("📁 Proposing to move file: %s → %s", oldPath, newPath))
	msg.Metadata["file_change_proposal"] = proposal
	a.attachWorkspaceContextToProposalMessage(channel, msg, proposal)

	return a.Hub.SendMessage(msg)
}

func (a *Agent) attachWorkspaceContextToProposalMessage(channel string, msg *protocol.Message, proposal *protocol.FileChangeProposal) {
	workspaceContext, ok := a.latestWorkspaceContext(channel)
	if !ok {
		return
	}
	msg.Metadata["workspace_context"] = workspaceContext
	if proposal.Metadata == nil {
		proposal.Metadata = make(map[string]interface{})
	}
	proposal.Metadata["workspace_context"] = workspaceContext
}

func (a *Agent) latestWorkspaceContext(channel string) (interface{}, bool) {
	if wc, ok := a.latestWorkspaceContextForChannel(channel); ok {
		return wc, true
	}
	// Collaboration runs in collab-* channels; user workspace metadata often
	// exists only on #general or another channel the human used first.
	if channel != "general" {
		if wc, ok := a.latestWorkspaceContextForChannel("general"); ok {
			return wc, true
		}
	}
	for _, ch := range a.historyChannelNames() {
		if ch == channel || ch == "general" {
			continue
		}
		if wc, ok := a.latestWorkspaceContextForChannel(ch); ok {
			return wc, true
		}
	}
	return nil, false
}

func (a *Agent) latestWorkspaceContextForChannel(channel string) (interface{}, bool) {
	history := a.channelHistory(channel)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i] == nil || history[i].Metadata == nil {
			continue
		}
		if wsCtx, ok := history[i].Metadata["workspace_context"]; ok && wsCtx != nil {
			return wsCtx, true
		}
	}
	return nil, false
}

// SetAIProvider dynamically switches the AI provider for this agent
func (a *Agent) SetAIProvider(newProvider ai.AIProvider) error {
	a.providerMutex.Lock()
	defer a.providerMutex.Unlock()

	// Update the AI provider
	a.AI = newProvider

	// Update agent info
	a.Info.Model = newProvider.GetModel()

	// Determine provider type and model
	aiProvider := "claude"
	aiModel := newProvider.GetModel()

	// Check provider type by checking the provider instance type
	switch p := newProvider.(type) {
	case *ai.OllamaProvider:
		aiProvider = "ollama"
	case *ai.LMStudioProvider:
		aiProvider = "lmstudio"
	case *ai.CLIAgentProvider:
		if name := strings.TrimSpace(p.ProviderName); name != "" {
			aiProvider = name
		} else {
			aiProvider = "cursor-cli"
		}
	default:
		// Check if it's an Ollama provider by checking the model name (fallback)
		if strings.Contains(aiModel, "llama") || strings.Contains(aiModel, "mistral") ||
			strings.Contains(aiModel, "phi") || strings.Contains(aiModel, "gemma") ||
			strings.Contains(aiModel, "codellama") {
			aiProvider = "ollama"
		}
	}

	a.Info.AIProvider = aiProvider
	a.Info.AIModel = aiModel
	return nil
}

// GetAIProvider returns the current AI provider
func (a *Agent) GetAIProvider() ai.AIProvider {
	a.providerMutex.RLock()
	defer a.providerMutex.RUnlock()
	return a.AI
}

// GetAgentInfo returns the agent's identity information.
func (a *Agent) GetAgentInfo() protocol.AgentInfo {
	return a.Info
}

// SetCollabClient sets the collaboration client for multi-agent collaboration support.
func (a *Agent) SetCollabClient(client CollaborationClient) {
	a.Collab = client
}
