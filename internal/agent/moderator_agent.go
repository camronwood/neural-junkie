package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MessageTracker tracks user messages to detect when no agent responds
type MessageTracker struct {
	MessageID   string
	Timestamp   time.Time
	HasResponse bool
	Channel     string
	FromUser    bool
}

// ModeratorAgent is a special system agent that helps users with chat features
// and steps in when no specialized agents respond
type ModeratorAgent struct {
	*Agent
	trackedMessages map[string]*MessageTracker
	trackerMutex    sync.RWMutex
	stopTracking    chan struct{}
}

// NewModeratorAgent creates a new moderator agent
func NewModeratorAgent(name string, ai ai.AIProvider, hub HubClient) *ModeratorAgent {
	expertise := []string{
		"Chat Commands",
		"Chat Features",
		"Agent System",
		"User Assistance",
		"System Help",
	}

	baseAgent := NewAgent(protocol.AgentTypeModerator, name, expertise, ai, hub)

	moderator := &ModeratorAgent{
		Agent:           baseAgent,
		trackedMessages: make(map[string]*MessageTracker),
		stopTracking:    make(chan struct{}),
	}

	return moderator
}

// Start begins the moderator's message processing loop with tracking
func (m *ModeratorAgent) Start(ctx context.Context, channel string) error {
	// Start the timeout monitoring goroutine
	go m.monitorTimeouts(ctx)

	// Call base agent's Start method
	return m.Agent.Start(ctx, channel)
}

// ProcessMessage overrides base agent to add message tracking
func (m *ModeratorAgent) ProcessMessage(ctx context.Context, msg *protocol.Message) {
	// Track user messages (not from agents)
	if m.isUserMessage(msg) {
		m.trackUserMessage(msg)
	}

	// Mark tracked messages as having responses when agents reply
	if m.isAgentMessage(msg) && msg.ReplyTo != "" {
		m.markAsResponded(msg.ReplyTo)
	}

	// Let base agent handle the rest
	m.Agent.handleMessage(ctx, msg)
}

// isUserMessage checks if a message is from a user (not an agent)
func (m *ModeratorAgent) isUserMessage(msg *protocol.Message) bool {
	// Users don't have an agent type or it's empty
	return msg.From.Type == "" || msg.From.Type == protocol.AgentTypeGeneral
}

// isAgentMessage checks if a message is from an agent
func (m *ModeratorAgent) isAgentMessage(msg *protocol.Message) bool {
	return msg.From.Type != "" && msg.From.Type != protocol.AgentTypeGeneral
}

// trackUserMessage adds a user message to the tracking system
func (m *ModeratorAgent) trackUserMessage(msg *protocol.Message) {
	// Only track questions or messages that might need responses
	// Skip commands as they get immediate responses
	if strings.HasPrefix(msg.Content, "/") {
		return
	}

	m.trackerMutex.Lock()
	defer m.trackerMutex.Unlock()

	m.trackedMessages[msg.ID] = &MessageTracker{
		MessageID:   msg.ID,
		Timestamp:   msg.Timestamp,
		HasResponse: false,
		Channel:     msg.Channel,
		FromUser:    true,
	}

	log.Printf("[%s] Tracking user message: %s (channel: %s)", m.Info.Name, msg.ID, msg.Channel)
}

// markAsResponded marks a tracked message as having received a response
func (m *ModeratorAgent) markAsResponded(messageID string) {
	m.trackerMutex.Lock()
	defer m.trackerMutex.Unlock()

	if tracker, exists := m.trackedMessages[messageID]; exists {
		tracker.HasResponse = true
		log.Printf("[%s] Message %s received response", m.Info.Name, messageID)
	}
}

// monitorTimeouts checks for messages that haven't received responses
func (m *ModeratorAgent) monitorTimeouts(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopTracking:
			return
		case <-ticker.C:
			m.checkTimeouts(ctx)
		}
	}
}

// checkTimeouts looks for messages that need moderator intervention
func (m *ModeratorAgent) checkTimeouts(ctx context.Context) {
	m.trackerMutex.Lock()
	defer m.trackerMutex.Unlock()

	now := time.Now()
	toDelete := []string{}

	for msgID, tracker := range m.trackedMessages {
		elapsed := now.Sub(tracker.Timestamp)

		// If 20 seconds have passed and no response, step in
		if !tracker.HasResponse && elapsed >= 20*time.Second {
			log.Printf("[%s] No response for message %s after 20s, stepping in", m.Info.Name, msgID)
			go m.respondToUnanswered(ctx, tracker)
			toDelete = append(toDelete, msgID)
		}

		// Clean up old messages (older than 5 minutes)
		if elapsed > 5*time.Minute {
			toDelete = append(toDelete, msgID)
		}
	}

	// Remove processed/old messages
	for _, msgID := range toDelete {
		delete(m.trackedMessages, msgID)
	}
}

// respondToUnanswered sends a helpful message when no agent responded
func (m *ModeratorAgent) respondToUnanswered(ctx context.Context, tracker *MessageTracker) {
	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		tracker.Channel,
		m.Info,
		"👋 I noticed no agents responded to your question. This chat is designed for development and technical discussions. "+
			"If you're looking for help with the chat system itself, feel free to ask me about:\n"+
			"• Available commands (type /help)\n"+
			"• How to mention agents (@name or @type)\n"+
			"• Creating repo or helper agents\n\n"+
			"For technical questions, try mentioning specific agent types like @backend, @frontend, @devops, etc.",
	)
	response.ReplyTo = tracker.MessageID

	if err := m.Hub.SendMessage(response); err != nil {
		log.Printf("[%s] Failed to send unanswered message response: %v", m.Info.Name, err)
	}
}

// shouldRespond determines if the moderator should respond to a message
func (m *ModeratorAgent) shouldRespond(msg *protocol.Message) bool {
	// Don't respond to our own messages
	if msg.From.ID == m.Info.ID {
		return false
	}

	// Don't respond to other agents' messages
	if m.isAgentMessage(msg) {
		return false
	}

	// Always respond if mentioned
	if msg.IsMentioned(m.Info.ID) {
		return true
	}

	// If message has mentions but we're not mentioned, don't respond
	// This prevents moderator from responding when other agents are mentioned
	if msg.HasMentions() {
		return false
	}

	content := strings.ToLower(msg.Content)

	// Respond to chat feature questions (only when no mentions)
	chatKeywords := []string{
		"how do i", "how to",
		"command", "/",
		"mention", "@",
		"thread", "channel",
		"agent", "help",
		"create repo", "repo agent",
		"helper agent",
		"moderator",
		"chat feature", "chat room",
	}

	for _, keyword := range chatKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// buildModeratorPrompt creates a specialized prompt with chat system knowledge.
// Uses the system/user separator so providers can send identity as a system message.
func (m *ModeratorAgent) buildModeratorPrompt(msg *protocol.Message) string {
	var system strings.Builder
	var user strings.Builder

	// ── SYSTEM SECTION ──
	system.WriteString("You are the ChatModerator, a helpful system assistant for the Neural Junkie. Users can mention you with @ChatModerator.\n\n")
	system.WriteString("=== YOUR ROLE ===\n")
	system.WriteString("You help users understand and use the chat system effectively. You are friendly, concise, and helpful.\n\n")

	// Self-knowledge: honest identity
	system.WriteString("=== YOUR TECHNICAL IDENTITY ===\n")
	system.WriteString(fmt.Sprintf("You are powered by the %q model via the %q provider.\n", m.Agent.Info.AIModel, m.Agent.Info.AIProvider))
	system.WriteString("If a user asks what model or LLM you are running, answer honestly with this information.\n")
	system.WriteString("Do NOT fabricate or guess your model architecture. Only state what is listed above.\n\n")

	system.WriteString("=== CHAT SYSTEM KNOWLEDGE ===\n\n")

	system.WriteString("**Available Slash Commands:**\n")
	system.WriteString("• /help - Show help information\n")
	system.WriteString("• /list-agents - List all active agents\n")
	system.WriteString("• /create-repo-agent <path> [name] - Create a repository expert agent\n")
	system.WriteString("• /create-helper <template> [name] - Create a helper agent (templates: day-one, testing-expert, docs-expert)\n")
	system.WriteString("• /list-helper-templates - List available helper agent templates\n")
	system.WriteString("• /delete-agent <name> - Delete an agent\n")
	system.WriteString("• /pause-agent <name> - Pause an agent\n")
	system.WriteString("• /unpause-agent <name> - Unpause an agent\n")
	system.WriteString("• /reindex-agent <name> - Reindex a repo agent\n")
	system.WriteString("• /enable-watch <name> - Enable file watching for repo agent\n")
	system.WriteString("• /disable-watch <name> - Disable file watching\n")
	system.WriteString("**Agent Types:**\n")
	system.WriteString("• @frontend - React, Vue, Angular, UI/UX, TypeScript, CSS\n")
	system.WriteString("• @backend - Go, Node.js, APIs, microservices, architecture\n")
	system.WriteString("• @devops - Docker, Kubernetes, CI/CD, infrastructure\n")
	system.WriteString("• @database - PostgreSQL, schema design, queries, migrations\n")
	system.WriteString("• @security - Authentication, authorization, encryption, best practices\n")
	system.WriteString("• @rust - Rust, ownership, lifetimes, async, traits, cargo, unsafe, WASM\n")
	system.WriteString("• @repo - Repository-specific experts with deep code knowledge\n")
	system.WriteString("• @helper - Custom expert agents with specialized knowledge bases\n")
	system.WriteString("• @assistant - Reminders, tasks, notes, scheduling, summarization\n")
	system.WriteString("• @Cursor - CLI agent: code generation, refactoring, shell commands (Cursor)\n")
	system.WriteString("• @Gemini - CLI agent: code generation, code review, multimodal analysis (Google Gemini)\n\n")

	system.WriteString("**Mention System:**\n")
	system.WriteString("• Mention by name: @AgentName (e.g., @GoExpert)\n")
	system.WriteString("• Mention by type: @backend, @frontend, etc.\n")
	system.WriteString("• Agents always respond when mentioned\n\n")

	system.WriteString("**Thread Feature:**\n")
	system.WriteString("• Reply to messages to create threads\n")
	system.WriteString("• Keeps conversations organized\n")
	system.WriteString("• Click thread icon to view thread details\n\n")

	system.WriteString("**Repository Agents:**\n")
	system.WriteString("• Deep code analysis and understanding\n")
	system.WriteString("• Persistent caching for fast responses\n")
	system.WriteString("• Create with `/create-repo-agent /path/to/repo [name]`\n")
	system.WriteString("• Expert on specific codebase structure and patterns\n\n")

	system.WriteString("=== GUIDELINES ===\n")
	system.WriteString("• Be concise (2-4 sentences unless more detail is needed)\n")
	system.WriteString("• Provide specific command examples when relevant\n")
	system.WriteString("• Direct users to appropriate agent types for technical questions\n")
	system.WriteString("• Be encouraging and patient\n")

	AppendUserAndAgentRules(&system, msg, &m.Agent.Info)

	// ── USER SECTION ──
	user.WriteString(fmt.Sprintf("User question from %s:\n%s\n\n", msg.From.Name, msg.Content))
	AppendPromptAttachments(&user, msg)
	user.WriteString("Provide a helpful response:")

	return system.String() + ai.SystemPromptSeparator + user.String()
}

// GenerateResponse overrides base agent to use moderator-specific prompt
func (m *ModeratorAgent) GenerateResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	prompt := m.buildModeratorPrompt(msg)

	// Get recent conversation history for context
	history := m.Context.History[msg.Channel]
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	response, err := m.AI.GenerateResponse(ctx, prompt, historyToMessages(history))
	if err != nil {
		return "", err
	}

	return response, nil
}

// Stop gracefully stops the moderator agent
func (m *ModeratorAgent) Stop() {
	close(m.stopTracking)
	m.Agent.Stop()
}
