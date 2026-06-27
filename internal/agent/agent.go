package agent

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
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

	// workspaceBackendLookup resolves nj-remote backends by workspace id (hub keychain token).
	workspaceBackendLookup func(workspaceID string) workspacebackend.Backend

	// collabTaskReplyAt rate-limits responses to collaboration_task prompts.
	collabTaskReplyMu sync.Mutex
	collabTaskReplyAt map[string]time.Time

	// activeGens tracks in-flight generation cancel funcs per channel (interject/stop).
	activeGenMu sync.Mutex
	activeGens  map[string]map[string]context.CancelFunc // channel -> genID -> cancel

	// lastDelegationConsulted holds specialist names consulted on the previous turn (for metadata).
	delegationMu            sync.Mutex
	lastDelegationConsulted []string
	lastRepoConsulted       string

	// routingSnap records provider/model used for the current turn (observability metadata).
	routingSnap routingSnapshotHolder

	// compressSnap records context compression stats for the current turn.
	compressSnap compressSnapshotHolder

	// cadWrittenPaths tracks workspace-relative .scad paths written during the current turn.
	cadWrittenMu    sync.Mutex
	cadWrittenPaths []string

	// Optional pre-processing hook for specialized agents. When set and it
	// returns true, the message is considered fully handled and the base
	// response pipeline is skipped.
	messageInterceptor func(context.Context, *protocol.Message) bool

	// Optional full prompt builder (Assistant uses buildAssistantPrompt).
	customPromptBuilder func(*protocol.Message) string

	// unansweredTracker monitors public-channel user messages for the Assistant safety net.
	unansweredTracker *unansweredMessageTracker
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
	// Music generation (music-creation pack + ACE-Step sidecar).
	MusicGenerationEnabled() bool
	GenerateAndPostMusic(ctx context.Context, channel string, from protocol.AgentInfo, req MusicGenerateRequest) error
}

// MusicGenerateRequest is input for hub music generation.
type MusicGenerateRequest struct {
	StyleTags    string
	Lyrics       string
	DurationSec  int
	Instrumental bool
	Seed         int
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
	// PlanningSpeakerCooldownBlocked is true when this agent already spoke in the
	// current planning round and another participant has not spoken yet.
	PlanningSpeakerCooldownBlocked(collabID, agentID string) bool
}

// CollaborationInfo carries the subset of collaboration state an agent
// needs when building prompts and deciding whether to respond.
type CollaborationInfo struct {
	ID                     string
	Description            string
	Phase                  string
	PlanContent            string
	PlanVersion            int
	AgentRole              string
	Agents                 []CollaborationAgentSummary
	Channel                string
	ExecutionMode          string // sandbox | worktree
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
func (a *Agent) SetMessageInterceptor(interceptor func(context.Context, *protocol.Message) bool) {
	a.messageInterceptor = interceptor
}

// SetPromptBuilder replaces the default buildPrompt for this agent instance.
func (a *Agent) SetPromptBuilder(builder func(*protocol.Message) string) {
	a.customPromptBuilder = builder
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

// historyToMessages converts protocol messages to a simpler format
func historyToMessages(history []*protocol.Message) []protocol.Message {
	msgs := make([]protocol.Message, len(history))
	for i, msg := range history {
		msgs[i] = *msg
	}
	return msgs
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
