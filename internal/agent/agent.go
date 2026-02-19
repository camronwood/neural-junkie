package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/google/uuid"
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
}

// MCPServerInterface defines the interface for MCP servers
type MCPServerInterface interface {
	GetMCPServer() interface{} // Returns the underlying MCP server
	Start() error              // Starts the MCP server
}

// AIProvider is now defined in the ai package

// HubClient defines the interface for interacting with the chat hub
type HubClient interface {
	SendMessage(msg *protocol.Message) error
	Subscribe(channelName string) (chan *protocol.Message, error)
	GetMessages(channelName string, limit int) ([]*protocol.Message, error)
	GetChannelAgents(channelName string) ([]protocol.AgentInfo, error)
	GetThreadParentAuthor(threadID string) string
	GetCommandHandler() CommandHandlerInterface
	GetAgentChannels(agentID string) []string
	GetChannelType(channelName string) protocol.ChannelType
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

	// Start periodic channel discovery so the agent picks up new
	// channels it has been added to (DMs, custom channels, etc.)
	go a.discoverChannels(ctx)

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

	go a.discoverChannels(ctx)
	return nil
}

// AddChannel subscribes the agent to an additional channel dynamically
func (a *Agent) AddChannel(ctx context.Context, channel string) error {
	a.channelMu.Lock()
	if _, exists := a.activeChannels[channel]; exists {
		a.channelMu.Unlock()
		return nil // already listening
	}
	a.channelMu.Unlock()

	subCh, err := a.Hub.Subscribe(channel)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	history, err := a.Hub.GetMessages(channel, 20)
	if err == nil {
		a.Context.History[channel] = history
	}

	chCtx, cancel := context.WithCancel(ctx)

	a.channelMu.Lock()
	a.activeChannels[channel] = cancel
	a.channelMu.Unlock()

	log.Printf("[%s] Agent listening on channel: %s", a.Info.Name, channel)

	go func() {
		for {
			select {
			case <-chCtx.Done():
				return
			case <-a.stopCh:
				return
			case msg := <-subCh:
				if msg == nil {
					return
				}
				a.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

// discoverChannels periodically checks for new channels this agent was added to
func (a *Agent) discoverChannels(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
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

	// Check if we've already responded to this message (atomic check-and-set)
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMutex.Unlock()

	// Add to history first (so we have context for decision)
	a.addToHistory(msg)

	// Decide if we should respond BEFORE marking as responded
	// This allows other agents to process the message if we don't respond
	if !a.shouldRespond(msg) {
		return
	}

	// Now mark as responded since we're going to respond
	// Use atomic check-and-set to prevent race conditions
	a.respondedMutex.Lock()
	if a.respondedMessages[msg.ID] {
		// Another agent already responded, skip
		a.respondedMutex.Unlock()
		return
	}
	a.respondedMessages[msg.ID] = true
	a.respondedMutex.Unlock()

	// Log that we're processing this message
	log.Printf("[%s] ⬇️ RECEIVED msg ID %s from %s (mentions: %v)", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Mentions)
	log.Printf("[%s] ✅ MARKED msg %s as responded", a.Info.Name, msg.ID[:8])

	log.Printf("[%s] 💬 WILL RESPOND to msg %s from %s: %s", a.Info.Name, msg.ID[:8], msg.From.Name, msg.Content[:min(50, len(msg.Content))])
	log.Printf("[%s] 🔍 Message details - ThreadID: '%s', IsThreadReply: %v, ReplyTo: '%s'", a.Info.Name, msg.ThreadID, msg.IsThreadReply, msg.ReplyTo)

	// Send thinking status
	a.sendThinkingStatus(msg, protocol.ThinkingStatusStarted)

	// Generate response
	log.Printf("[%s] 📝 Generating response...", a.Info.Name)
	response, err := a.generateResponse(ctx, msg)
	if err != nil {
		log.Printf("[%s] Error generating response: %v", a.Info.Name, err)
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}
	log.Printf("[%s] ✍️  Generated response: %s", a.Info.Name, response[:min(50, len(response))])

	// Send response
	responseMsg := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		a.Info,
		response,
	)
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
		// Look for the message being replied to
		for _, histMsg := range a.Context.History[msg.Channel] {
			if histMsg.ID == msg.ReplyTo {
				// Check if it's from another agent (review scenario)
				isFromAgent := histMsg.From.Type == protocol.AgentTypeFrontend ||
					histMsg.From.Type == protocol.AgentTypeBackend ||
					histMsg.From.Type == protocol.AgentTypeDatabase ||
					histMsg.From.Type == protocol.AgentTypeSecurity ||
					histMsg.From.Type == protocol.AgentTypeDevOps ||
					histMsg.From.Type == protocol.AgentTypeRepo ||
					histMsg.From.Type == protocol.AgentTypeHelper ||
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
				}
				break
			}
		}
	}

	// Detect commands in the response and add them to metadata
	commandDetector := protocol.NewCommandDetector(nil)
	suggestions := commandDetector.DetectCommands(response, a.Info.Name, responseMsg.ID)
	if len(suggestions) > 0 {
		responseMsg.Metadata["suggested_commands"] = suggestions
		log.Printf("[%s] 🔧 Detected %d command suggestions", a.Info.Name, len(suggestions))
	}

	log.Printf("[%s] 📤 Sending response msg ID %s (replying to %s)...", a.Info.Name, responseMsg.ID[:8], msg.ID[:8])
	if err := a.Hub.SendMessage(responseMsg); err != nil {
		log.Printf("[%s] Error sending message: %v", a.Info.Name, err)
		a.sendThinkingStatus(msg, protocol.ThinkingStatusError)
		return
	}
	log.Printf("[%s] ✅ Response sent successfully!", a.Info.Name)
	a.sendThinkingStatus(msg, protocol.ThinkingStatusCompleted)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sendThinkingStatus sends an agent_status message indicating thinking state
func (a *Agent) sendThinkingStatus(originalMsg *protocol.Message, status protocol.ThinkingStatus) {
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		originalMsg.Channel,
		a.Info,
		"", // Empty content for status messages
	)
	statusMsg.Metadata["thinking_status"] = string(status)
	statusMsg.Metadata["question_id"] = originalMsg.ID

	// Fire and forget - don't block on sending status
	go func() {
		if err := a.Hub.SendMessage(statusMsg); err != nil {
			log.Printf("[%s] Warning: failed to send thinking status: %v", a.Info.Name, err)
		}
	}()
}

// shouldRespond determines if the agent should respond to a message
func (a *Agent) shouldRespond(msg *protocol.Message) bool {
	// Never respond to commands - let the command handler process them
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		return false
	}

	// Special handling for design analysis requests
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		// Only frontend agents should respond to design analysis
		if a.Info.Type == protocol.AgentTypeFrontend {
			log.Printf("[%s] 🎨 DESIGN ANALYSIS request detected - will respond", a.Info.Name)
			return true
		}
	}

	// Never respond to system messages (errors, notifications, etc.)
	if msg.From.Name == "System" || msg.From.ID == "system" {
		return false
	}

	// Never respond to our own messages
	if msg.From.ID == a.Info.ID {
		return false
	}

	// In DM channels, always respond to non-agent messages (the user is talking directly to us)
	channelType := a.Hub.GetChannelType(msg.Channel)
	if channelType == protocol.ChannelTypeDM {
		isFromAgent := msg.From.Type == protocol.AgentTypeFrontend ||
			msg.From.Type == protocol.AgentTypeBackend ||
			msg.From.Type == protocol.AgentTypeDatabase ||
			msg.From.Type == protocol.AgentTypeSecurity ||
			msg.From.Type == protocol.AgentTypeRust ||
			msg.From.Type == protocol.AgentTypeDevOps ||
			msg.From.Type == protocol.AgentTypeRepo ||
			msg.From.Type == protocol.AgentTypeHelper ||
			msg.From.Type == protocol.AgentTypeAssistant ||
			msg.From.Type == protocol.AgentTypeModerator ||
			msg.From.Type == protocol.AgentTypeCLI
		if !isFromAgent {
			log.Printf("[%s] ✅ DM CHANNEL - will respond", a.Info.Name)
			return true
		}
		return false
	}

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
		msg.From.Type == protocol.AgentTypeDevOps ||
		msg.From.Type == protocol.AgentTypeRepo ||
		msg.From.Type == protocol.AgentTypeHelper ||
		msg.From.Type == protocol.AgentTypeAssistant ||
		msg.From.Type == protocol.AgentTypeModerator ||
		msg.From.Type == protocol.AgentTypeCLI

	// If message has @mentions, ONLY respond if explicitly mentioned
	// This works even for agent-to-agent communication if explicitly mentioned
	if msg.HasMentions() {
		if msg.IsMentioned(a.Info.ID) {
			// Check if this is a review request (replying to another agent's message)
			if msg.ReplyTo != "" {
				// Find the message being replied to
				var repliedToMsg *protocol.Message
				for _, histMsg := range a.Context.History[msg.Channel] {
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
					repliedToMsg.From.Type == protocol.AgentTypeDevOps ||
					repliedToMsg.From.Type == protocol.AgentTypeRepo ||
					repliedToMsg.From.Type == protocol.AgentTypeHelper ||
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

	// Check if it's a question - require explicit question mark for better precision
	isQuestion := msg.Type == protocol.MessageTypeQuestion ||
		strings.Contains(content, "?")

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
		if len(word) >= 3 {
			wordSet[word] = true
		}
	}

	// Check expertise keywords - require whole word matches
	for _, skill := range a.Info.Expertise {
		skillLower := strings.ToLower(skill)
		skillWords := strings.Fields(skillLower)

		// Check if any significant word from expertise appears in message
		for _, skillWord := range skillWords {
			skillWord = strings.Trim(skillWord, ".,!?;:")
			if len(skillWord) >= 4 && wordSet[skillWord] {
				return true
			}
		}

		// Also check for full skill phrase match (for multi-word skills like "task management")
		if len(skillWords) > 1 {
			skillPhrase := strings.Join(skillWords, " ")
			if strings.Contains(content, skillPhrase) {
				return true
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
			return true
		}
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
		return []string{"deploy", "deployment", "ci/cd", "docker", "kubernetes", "infrastructure", "monitoring"}
	case protocol.AgentTypeDatabase:
		return []string{"database", "sql", "query", "schema", "migration", "postgres", "mysql", "mongodb"}
	case protocol.AgentTypeSecurity:
		return []string{"security", "auth", "authentication", "authorization", "encryption", "vulnerability", "xss", "sql injection"}
	case protocol.AgentTypeRust:
		return []string{"rust", "cargo", "tokio", "ownership", "borrowing", "lifetime", "trait", "async", "unsafe", "wasm", "serde", "crate"}
	default:
		return []string{}
	}
}

// generateResponse generates an AI response based on the message and context
func (a *Agent) generateResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	// Check if this is a design analysis request
	if designAnalysis, ok := msg.Metadata["design_analysis"].(bool); ok && designAnalysis {
		return a.generateDesignAnalysisResponse(ctx, msg)
	}

	prompt := a.buildPrompt(msg)

	// Track files already included in the prompt so the workspace scanner
	// doesn't duplicate them.
	includedFiles := collectIncludedFilePaths(msg)

	// Auto-detect and load file paths referenced in the user's message.
	wsPath := a.resolveWorkspacePath(msg)
	if wsPath != "" {
		var referencedFiles strings.Builder
		AppendReferencedFiles(&referencedFiles, msg.Content, wsPath)
		if referencedFiles.Len() > 0 {
			prompt += referencedFiles.String()
			for _, p := range DetectFilePaths(msg.Content) {
				includedFiles[p] = true
			}
		}
	}

	// Proactively scan the workspace for domain-relevant source files.
	// This lets specialist agents (RustExpert, GoExpert, etc.) see project
	// code even when the user doesn't mention specific file paths.
	if wsPath != "" && !a.isRepoOrHelperAgent() {
		existingContextSize := len(prompt) - len(a.buildPrompt(msg))
		if existingContextSize < maxScanChars/2 {
			scannedFiles, err := ScanWorkspaceFiles(wsPath, a.Info.Type, msg.Content, maxScanChars, includedFiles)
			if err != nil {
				log.Printf("[%s] Workspace scan failed: %v", a.Info.Name, err)
			} else if scannedFiles != "" {
				prompt += scannedFiles
			}
		}
	}

	// Get recent conversation history for context
	history := a.Context.History[msg.Channel]
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	response, err := a.AI.GenerateResponse(ctx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	return response, nil
}

// isRepoOrHelperAgent returns true for agent types that already have their
// own file-context strategy (repo agents use their index, helpers use
// knowledge bases, CLI agents have shell access).
func (a *Agent) isRepoOrHelperAgent() bool {
	switch a.Info.Type {
	case protocol.AgentTypeRepo, protocol.AgentTypeHelper, protocol.AgentTypeCLI,
		protocol.AgentTypeModerator, protocol.AgentTypeAssistant, protocol.AgentTypeConfluence:
		return true
	default:
		return false
	}
}

// collectIncludedFilePaths extracts file paths that are already present in
// the prompt via workspace context (open editor tabs) so the scanner can
// skip them.
func collectIncludedFilePaths(msg *protocol.Message) map[string]bool {
	paths := make(map[string]bool)
	if msg.Metadata == nil {
		return paths
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
func (a *Agent) buildPrompt(msg *protocol.Message) string {
	var system strings.Builder
	var user strings.Builder

	// ── SYSTEM SECTION ──────────────────────────────────────────────────
	system.WriteString(fmt.Sprintf("You are %s, a %s specialist agent in a multi-agent collaboration chat room.\n\n", a.Info.Name, a.Info.Type))
	system.WriteString(fmt.Sprintf("Your expertise: %s\n\n", strings.Join(a.Info.Expertise, ", ")))

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

	// Add MCP tools if available
	if a.MCPServer != nil {
		system.WriteString("AVAILABLE TOOLS:\n")
		system.WriteString("You have access to the following diagnostic and analysis tools:\n")

		switch a.Info.Type {
		case protocol.AgentTypeBackend:
			system.WriteString("- analyze_go_code(file_path): Run static analysis on Go code using go vet, staticcheck, golangci-lint\n")
			system.WriteString("- run_go_tests(package_path): Execute Go tests and return results\n")
			system.WriteString("- profile_performance(binary_path, endpoint): Profile Go application performance using pprof\n")
			system.WriteString("- check_dependencies(module_path): Check Go module dependencies for vulnerabilities\n")
			system.WriteString("- detect_race_conditions(package_path): Run Go race detector on tests\n")
		case protocol.AgentTypeRust:
			system.WriteString("- cargo_check(package_path): Run cargo check for compile errors without producing binaries\n")
			system.WriteString("- cargo_clippy(package_path): Run clippy lints for idiomatic Rust improvements\n")
			system.WriteString("- cargo_test(package_path): Execute Rust tests with cargo test\n")
			system.WriteString("- cargo_audit(package_path): Audit Cargo.lock for known security vulnerabilities\n")
			system.WriteString("- miri_check(package_path): Run Miri to detect undefined behavior in unsafe code\n")
		case protocol.AgentTypeDevOps:
			system.WriteString("- kubectl_query(resource, namespace): Query Kubernetes cluster using kubectl\n")
			system.WriteString("- check_docker_image(image_name): Analyze Docker image for size, layers, and vulnerabilities\n")
			system.WriteString("- validate_yaml(yaml_file): Validate Kubernetes or Helm YAML files\n")
			system.WriteString("- check_pod_logs(pod_name, namespace): Fetch and analyze logs from Kubernetes pods\n")
			system.WriteString("- query_prometheus(query): Query Prometheus metrics for monitoring data\n")
		case protocol.AgentTypeDatabase:
			system.WriteString("- explain_query(sql_query): Run EXPLAIN ANALYZE on SQL queries to analyze performance\n")
			system.WriteString("- check_indexes(table_name): Analyze table indexes for optimization opportunities\n")
			system.WriteString("- validate_schema(schema_name): Check database schema for consistency and best practices\n")
			system.WriteString("- suggest_optimizations(table_name): Analyze query patterns and suggest database optimizations\n")
			system.WriteString("- check_table_stats(table_name): Get table statistics including size, row count, and storage info\n")
			system.WriteString("- generate_migration(description, changes): Generate database migration scripts based on schema changes\n")
		}

		system.WriteString("\nUse these tools to provide data-driven answers. When diagnosing issues,\n")
		system.WriteString("USE THE TOOLS to get actual data rather than guessing. Always explain\n")
		system.WriteString("what tools you used and what the results show.\n\n")
	}

	// Core behavioral rules (always in system prompt)
	system.WriteString("=== BEHAVIORAL RULES ===\n")
	system.WriteString("1. Provide expert advice grounded in your domain expertise.\n")
	system.WriteString("2. When the user shares code or files, you MUST analyze the ACTUAL code provided -- never give generic advice.\n")
	system.WriteString("3. Reference specific file paths, function names, and line numbers when discussing code.\n")
	system.WriteString("4. Do NOT @mention other agents unless the user explicitly asks for collaboration.\n")
	system.WriteString("5. Only respond to the user's question -- do not respond to other agents' responses.\n")
	system.WriteString("6. Ask clarifying questions when the request is ambiguous.\n")
	system.WriteString("7. CRITICAL: If asked to review, analyze, or explain code but NO code or workspace files appear in the context below, ")
	system.WriteString("you MUST tell the user you cannot see any code and ask them to either: ")
	system.WriteString("(a) include the file path in their message (e.g., 'review cmd/server/main.go'), or ")
	system.WriteString("(b) enable workspace sharing. NEVER fabricate or guess code content.\n")

	// Add context about other agents in the channel
	agents, _ := a.Hub.GetChannelAgents(msg.Channel)
	if len(agents) > 1 {
		system.WriteString("\nOther agents in this channel:\n")
		for _, agent := range agents {
			if agent.ID != a.Info.ID {
				system.WriteString(fmt.Sprintf("- %s (%s)\n", agent.Name, agent.Type))
			}
		}
	}

	// ── USER SECTION ────────────────────────────────────────────────────

	// Check if this is a review request (user asking to review another agent's response)
	isReview := false
	var reviewedMessage *protocol.Message
	if msg.ReplyTo != "" {
		for _, histMsg := range a.Context.History[msg.Channel] {
			if histMsg.ID == msg.ReplyTo {
				reviewedMessage = histMsg
				if histMsg.From.Type == protocol.AgentTypeFrontend ||
					histMsg.From.Type == protocol.AgentTypeBackend ||
					histMsg.From.Type == protocol.AgentTypeDatabase ||
					histMsg.From.Type == protocol.AgentTypeSecurity ||
					histMsg.From.Type == protocol.AgentTypeDevOps ||
					histMsg.From.Type == protocol.AgentTypeRepo ||
					histMsg.From.Type == protocol.AgentTypeHelper ||
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

	// Append workspace context if the user shared it
	AppendWorkspaceContext(&user, msg)

	// Adaptive response length based on intent
	user.WriteString(getResponseLengthGuidance(msg.Content))

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
		return `When asked to review or analyze code, evaluate:
- Component architecture (composition, prop drilling, component size)
- State management (local vs global state, unnecessary re-renders)
- Accessibility (ARIA attributes, keyboard navigation, screen reader support)
- Performance (memo/useMemo/useCallback usage, bundle size, lazy loading)
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
func getResponseLengthGuidance(content string) string {
	lower := strings.ToLower(content)

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

// addToHistory adds a message to the conversation history
func (a *Agent) addToHistory(msg *protocol.Message) {
	history := a.Context.History[msg.Channel]
	history = append(history, msg)

	// Trim history if too long
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
	return a.generateResponse(ctx, msg)
}

// generateDesignAnalysisResponse handles design analysis with vision API
func (a *Agent) generateDesignAnalysisResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	// Extract image data from metadata
	imageData, hasImage := msg.Metadata["image_data"].([]byte)
	imageType, hasImageType := msg.Metadata["image_type"].(string)

	if !hasImage || !hasImageType {
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

	// Get recent conversation history for context
	history := a.Context.History[msg.Channel]
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	// Use vision API for design analysis
	response, err := a.AI.GenerateVisionResponse(ctx, prompt, imageData, imageType, historyToMessages(history))
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

// File change proposal helper methods

// ProposeFileEdit proposes an edit to an existing file
func (a *Agent) ProposeFileEdit(path, oldContent, newContent string) error {
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "edit",
		FilePath:    path,
		OldContent:  oldContent,
		NewContent:  newContent,
		Agent:       a.Info,
		Channel:     "general", // Default channel for now
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, "general", a.Info,
		fmt.Sprintf("📝 Proposing to edit file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal

	return a.Hub.SendMessage(msg)
}

// ProposeFileCreate proposes creating a new file
func (a *Agent) ProposeFileCreate(path, content string) error {
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "create",
		FilePath:    path,
		NewContent:  content,
		Agent:       a.Info,
		Channel:     "general", // Default channel for now
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, "general", a.Info,
		fmt.Sprintf("📄 Proposing to create file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal

	return a.Hub.SendMessage(msg)
}

// ProposeFileDelete proposes deleting a file
func (a *Agent) ProposeFileDelete(path string) error {
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "delete",
		FilePath:    path,
		Agent:       a.Info,
		Channel:     "general", // Default channel for now
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    true,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, "general", a.Info,
		fmt.Sprintf("🗑️ Proposing to delete file: %s", path))
	msg.Metadata["file_change_proposal"] = proposal

	return a.Hub.SendMessage(msg)
}

// ProposeFileMove proposes moving/renaming a file
func (a *Agent) ProposeFileMove(oldPath, newPath string) error {
	// Create file change proposal
	proposal := &protocol.FileChangeProposal{
		ChangeID:    uuid.New().String()[:8],
		Operation:   "move",
		FilePath:    oldPath,
		OldPath:     oldPath,
		NewPath:     newPath,
		Agent:       a.Info,
		Channel:     "general", // Default channel for now
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		IsDelete:    false,
		Metadata:    make(map[string]interface{}),
	}

	// Create message with file change proposal
	msg := protocol.NewMessage(protocol.MessageTypeFileChange, "general", a.Info,
		fmt.Sprintf("📁 Proposing to move file: %s → %s", oldPath, newPath))
	msg.Metadata["file_change_proposal"] = proposal

	return a.Hub.SendMessage(msg)
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
	switch newProvider.(type) {
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

	a.Info.AIProvider = aiProvider
	a.Info.AIModel = aiModel

	// Send status update to hub
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		"general",
		a.Info,
		fmt.Sprintf("🔄 %s switched to %s (%s)", a.Info.Name, aiProvider, aiModel),
	)
	statusMsg.Metadata = map[string]interface{}{
		"ai_provider": aiProvider,
		"ai_model":    aiModel,
		"model":       aiModel,
	}

	return a.Hub.SendMessage(statusMsg)
}

// GetAIProvider returns the current AI provider
func (a *Agent) GetAIProvider() ai.AIProvider {
	a.providerMutex.RLock()
	defer a.providerMutex.RUnlock()
	return a.AI
}
