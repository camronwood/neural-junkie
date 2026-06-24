package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	assistantmcp "github.com/camronwood/neural-junkie/internal/mcp/assistant"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/fsnotify/fsnotify"
)

// AssistantAgent is a personal assistant agent that helps with reminders, tasks, notes, and more
type AssistantAgent struct {
	*Agent
	storage              *AssistantStorage
	reminderTicker       *time.Ticker
	stopReminders        chan struct{}
	reminderMutex        sync.RWMutex
	config               *AssistantConfig
	stopGoogleSync       chan struct{}
	googleSyncStarted    bool
	googleSyncMu         sync.Mutex
	emailWatcher         *fsnotify.Watcher
	stopEmailWatcher     chan struct{}
	processedEmails      map[string]bool
	processedEmailsMutex sync.RWMutex
	pendingApprovals     map[string]*PendingApproval
	approvalMutex        sync.RWMutex
}

// PendingApproval represents a pending approval request
type PendingApproval struct {
	ID          string    `json:"id"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	RequestedBy string    `json:"requested_by"`
	RequestedAt time.Time `json:"requested_at"`
	Channel     string    `json:"channel"`
}

// NewAssistantAgent creates a new assistant agent
func NewAssistantAgent(name string, ai ai.AIProvider, hub HubClient) *AssistantAgent {
	expertise := []string{
		"Reminders",
		"Task Management",
		"Note Taking",
		"Scheduling",
		"Time Management",
		"Productivity",
		"Organization",
		"Meeting Coordination",
		"Summarization",
		"Personal Assistant",
	}

	baseAgent := NewAgent(protocol.AgentTypeAssistant, name, expertise, ai, hub)

	// Initialize storage
	storage, err := NewAssistantStorage()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize assistant storage: %v", err)
		// Continue without storage - agent will work but won't persist data
	}

	// Load config
	config := DefaultAssistantConfig()
	if storage != nil {
		if loadedConfig, err := storage.LoadConfig(); err == nil {
			config = loadedConfig
		}
	}

	assistant := &AssistantAgent{
		Agent:            baseAgent,
		storage:          storage,
		stopReminders:    make(chan struct{}),
		config:           config,
		stopGoogleSync:   make(chan struct{}),
		stopEmailWatcher: make(chan struct{}),
		processedEmails:  make(map[string]bool),
		pendingApprovals: make(map[string]*PendingApproval),
	}

	baseAgent.unansweredTracker = newUnansweredMessageTracker(baseAgent)

	if assistantMCP, err := assistantmcp.NewAssistantMCP(); err != nil {
		log.Printf("Failed to create Assistant MCP server: %v", err)
	} else {
		assistantMCP.AttachWorkspaceTools(func() string { return baseAgent.WorkspacePath })
		assistantMCP.AttachWebTools()
		attachContextCompressTools(assistantMCP)
		baseAgent.MCPServer = assistantMCP
		log.Printf("Assistant workspace MCP tools registered for agent: %s", name)
	}

	// Ensure deterministic assistant actions are handled before the shared
	// LLM response path.
	assistant.Agent.SetMessageInterceptor(assistant.handleDirectAssistantActions)
	assistant.Agent.SetPromptBuilder(assistant.buildAssistantPrompt)

	return assistant
}

var assistantPromptNow = time.Now

// Start begins the assistant's message processing loop with reminder monitoring
func (a *AssistantAgent) Start(ctx context.Context, channel string) error {
	// Start the reminder monitoring goroutine
	go a.monitorReminders(ctx)

	if a.unansweredTracker != nil {
		a.unansweredTracker.start(ctx)
	}

	a.ensureGoogleMeetNotesSync(ctx)

	// Start email watcher if enabled
	if a.config.EmailIngestEnabled {
		if info, statErr := os.Stat(a.config.EmailDir); statErr != nil || !info.IsDir() {
			log.Printf("ℹ️  [Assistant] Email auto-ingestion unavailable: directory not found (%s)", a.config.EmailDir)
		} else {
			log.Printf("📧 [Assistant] Starting email watcher for directory: %s", a.config.EmailDir)
			if err := a.startEmailWatcher(ctx); err != nil {
				log.Printf("ℹ️  [Assistant] Email watcher unavailable: %v", err)
			} else {
				log.Printf("✅ [Assistant] Email watcher started successfully")
			}
		}
	} else {
		log.Printf("ℹ️  [Assistant] Email auto-ingestion is disabled")
	}

	// Call base agent's Start method
	return a.Agent.Start(ctx, channel)
}

// ProcessMessage overrides base agent to add assistant-specific processing
func (a *AssistantAgent) ProcessMessage(ctx context.Context, msg *protocol.Message) {
	// Handle common assistant actions directly so we do not claim work happened
	// when no command was actually executed.
	if a.handleDirectAssistantActions(ctx, msg) {
		return
	}

	// Direct meeting-note questions already use the grounded meeting prompt below.
	// Do not also send proactive context, or the UI can show two contradictory replies.
	if !assistantSkipPersonalContextEnrichment(msg) && !a.detectsMeetingQuery(msg) {
		a.checkProactiveSuggestions(ctx, msg)
	}

	// Let base agent handle the rest
	a.Agent.handleMessage(ctx, msg)
}

// ShouldRespond determines if the assistant should respond to a message
func (a *AssistantAgent) ShouldRespond(msg *protocol.Message) bool {
	return a.Agent.shouldRespond(msg)
}

// GenerateResponse uses the shared agent pipeline (closure shortcut, assistant prompt, history).
func (a *AssistantAgent) GenerateResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	eff := a.EffectiveAIProvider(ctx, msg)
	if eff == nil {
		eff = a.GetAIProvider()
	}
	return a.generateResponse(ctx, msg, eff)
}

// assistantSkipPersonalContextEnrichment returns true for collaboration orchestration
// messages where meeting/email enrichment must not run (avoids recap infinite loops).
func assistantSkipPersonalContextEnrichment(msg *protocol.Message) bool {
	if msg == nil {
		return true
	}
	switch msg.Type {
	case protocol.MessageTypeCollabRecap,
		protocol.MessageTypeCollabTask,
		protocol.MessageTypeCollabDiscussion:
		return true
	}
	if msg.GetCollaborationID() != "" {
		return true
	}
	return msg.IsFromSystem()
}

// buildAssistantPrompt creates a specialized prompt for the assistant.
// Uses the system/user separator so providers can send identity as a system message.
func (a *AssistantAgent) buildAssistantPrompt(msg *protocol.Message) string {
	return a.buildAssistantPromptCore(msg, false)
}

func (a *AssistantAgent) buildAssistantPromptCore(msg *protocol.Message, skipPersonalEnrichment bool) string {
	if isCollabRecapMessage(msg) {
		return buildCollabRecapPrompt(a.Info.Name, msg)
	}
	if !skipPersonalEnrichment && !assistantSkipPersonalContextEnrichment(msg) {
		if a.detectsMeetingQuery(msg) {
			log.Printf("🔍 [Assistant] Detected meeting query, using enriched context")
			return a.buildMeetingContextPrompt(msg)
		}
		if a.detectsEmailQuery(msg) {
			log.Printf("🔍 [Assistant] Detected email query, using enriched context")
			return a.buildEmailContextPrompt(msg)
		}
		if a.detectsNJAppQuery(msg) {
			log.Printf("🔍 [Assistant] Detected Neural Junkie app query, using full product knowledge")
		}
	}

	var prompt strings.Builder

	prompt.WriteString("You are the Assistant, a helpful AI in the Neural Junkie multi-agent collaboration system.\n\n")
	prompt.WriteString("=== YOUR ROLE ===\n")
	prompt.WriteString("You help users stay organized and productive by managing reminders, tasks, notes, and schedules. ")
	prompt.WriteString("You also serve as a knowledgeable guide to the Neural Junkie system itself — you know how to create agents, use commands, and leverage the full capabilities of this platform. ")
	prompt.WriteString("You are friendly, proactive, and always ready to help.\n\n")

	// Self-knowledge: honest identity
	prompt.WriteString("=== YOUR TECHNICAL IDENTITY ===\n")
	prompt.WriteString(fmt.Sprintf("You are powered by the %q model via the %q provider.\n", a.Agent.Info.AIModel, a.Agent.Info.AIProvider))
	prompt.WriteString("If a user asks what model or LLM you are running, answer honestly with this information.\n")
	prompt.WriteString("Do NOT fabricate or guess your model architecture. Only state what is listed above.\n\n")

	prompt.WriteString("=== YOUR CAPABILITIES ===\n\n")

	prompt.WriteString("**Reminders:**\n")
	prompt.WriteString("• Set time-based reminders: 'remind me in 30 minutes to review the PR'\n")
	prompt.WriteString("• Set recurring reminders: 'remind me daily at 9am about standup'\n")
	prompt.WriteString("• Context-triggered reminders: 'remind me when someone mentions deployment'\n\n")

	prompt.WriteString("**Task Management:**\n")
	prompt.WriteString("• Create and track tasks with priorities\n")
	prompt.WriteString("• Mark tasks as complete\n")
	prompt.WriteString("• Organize tasks by channel or project\n\n")

	prompt.WriteString("**Note Taking:**\n")
	prompt.WriteString("• Save important information from conversations\n")
	prompt.WriteString("• Tag notes for easy searching\n")
	prompt.WriteString("• Link notes to specific messages\n\n")

	prompt.WriteString("**Scheduling:**\n")
	prompt.WriteString("• Track meetings and events\n")
	prompt.WriteString("• Set up recurring meetings\n")
	prompt.WriteString("• Provide meeting reminders\n\n")

	prompt.WriteString("**Meeting Notes:**\n")
	prompt.WriteString("• Access and summarize meeting notes from previous meetings\n")
	prompt.WriteString("• Search through meeting content for specific topics\n")
	prompt.WriteString("• Provide summaries of recent meetings\n")
	prompt.WriteString("• Answer questions about meeting discussions and decisions\n")
	prompt.WriteString("• Full meeting-note and email context is loaded only when the user asks about meetings or email\n\n")

	prompt.WriteString("**Web lookup (Assistant MCP tools):**\n")
	prompt.WriteString("• web_search — query the public web when the user needs current or external facts\n")
	prompt.WriteString("• fetch_url — read a public HTTPS page after search when snippets are not enough\n")
	prompt.WriteString("• Requires Settings → Integrations → Web search (Tavily recommended; Brave also supported).\n\n")

	prompt.WriteString("**Conversation Summarization:**\n")
	prompt.WriteString("• Summarize long discussions\n")
	prompt.WriteString("• Extract action items and decisions\n")
	prompt.WriteString("• Identify key points and next steps\n\n")

	prompt.WriteString("=== ASSISTANT COMMANDS (your own) ===\n")
	prompt.WriteString("• /remind <time> <message> - Set a reminder\n")
	prompt.WriteString("• /remind-recurring <schedule> <message> - Set recurring reminder\n")
	prompt.WriteString("• /task-add <title> - Add a task\n")
	prompt.WriteString("• /task-list - List all tasks\n")
	prompt.WriteString("• /task-done <id> - Mark task complete\n")
	prompt.WriteString("• /note-save <content> - Save a note\n")
	prompt.WriteString("• /note-search <query> - Search notes\n")
	prompt.WriteString("• /meeting-add <time> <title> - Add meeting\n")
	prompt.WriteString("• /summarize [last N messages] - Summarize conversation\n")
	prompt.WriteString("• /help-assistant - Show assistant help\n\n")

	prompt.WriteString("=== SYSTEM COMMANDS (Neural Junkie platform) ===\n")
	prompt.WriteString("You MUST know and be able to explain these commands. When users ask about creating agents, using features, or how the system works, refer to these.\n\n")

	prompt.WriteString("**Repository Agents:**\n")
	prompt.WriteString("• /create-repo-agent <path> [name] [provider] [model] - Create a repository expert agent that analyzes a codebase\n")
	prompt.WriteString("  Example: /create-repo-agent /path/to/my-project MyProjectExpert ollama llama3.1\n")
	prompt.WriteString("  Providers: ollama (default), claude, lmstudio\n")
	prompt.WriteString("• /reindex-agent <name> - Reindex a repository agent's knowledge\n")
	prompt.WriteString("• /enable-watch <name> - Enable automatic file watching and reindexing for a repo agent\n")
	prompt.WriteString("• /disable-watch <name> - Disable automatic file watching\n\n")

	prompt.WriteString("**Agent Management:**\n")
	prompt.WriteString("• /remove-agent <name> - Remove agent from conversation (can recall later)\n")
	prompt.WriteString("• /recall-agent <name> - Recall a removed agent back to conversation\n")
	prompt.WriteString("• /list-removed-agents - List removed agents available for recall\n")
	prompt.WriteString("• /delete-agent <name> - Delete an agent permanently\n")
	prompt.WriteString("• /pause-agent <name> - Pause an agent (stops responding)\n")
	prompt.WriteString("• /unpause-agent <name> - Resume a paused agent\n")
	prompt.WriteString("• /list-agents - List all agents in the channel\n\n")

	prompt.WriteString("**Provider Switching:**\n")
	prompt.WriteString("• /switch-provider <agent-name> <provider> [model] - Switch AI provider for an agent\n")
	prompt.WriteString("• /switch-all-providers <provider> [model] - Switch AI provider for all agents\n")
	prompt.WriteString("  Providers: claude, ollama, lmstudio\n\n")

	prompt.WriteString("**MCP Exports:**\n")
	prompt.WriteString("• /export-agent-mcp <name> - Export an agent to MCP format\n")
	prompt.WriteString("• /list-exports - List all exported agents\n")
	prompt.WriteString("• /delete-export <name> - Delete an export\n")
	prompt.WriteString("• /import-agent-mcp <path> - Import an agent from file\n")
	prompt.WriteString("• /export-all-agents - Export all agents at once\n\n")

	prompt.WriteString("**Other:**\n")
	prompt.WriteString("• /help - Show all available commands\n")
	prompt.WriteString("• /migrate-agent-names - Check and migrate agent names for @mention compatibility\n\n")

	appendNJAppKnowledgeBrief(&prompt)
	if messageAsksAboutNJApp(msg.Content) {
		appendNJAppKnowledgeFull(&prompt)
	}

	prompt.WriteString("=== AGENTS & PROVIDERS ===\n")
	prompt.WriteString("• Specialist agents include Frontend, Backend, Platform, Security, Software Architecture, Code Review, Repository Experts, and custom domain experts.\n")
	prompt.WriteString("• Users @mention agents by name (e.g. @BackendEngineer) to direct questions; /list-agents shows who is in the channel.\n")
	prompt.WriteString("• AI providers: Ollama (local), Claude (Anthropic), LM Studio — switch per agent via /switch-provider or Settings.\n\n")

	prompt.WriteString("=== GUIDELINES ===\n")
	prompt.WriteString("• Be proactive and helpful\n")
	prompt.WriteString("• Use emojis appropriately (⏰ 📝 ✅ 📅 🔔)\n")
	prompt.WriteString("• Acknowledge commands with confirmation\n")
	prompt.WriteString("• Suggest helpful actions when appropriate\n")
	prompt.WriteString("• Be conversational and friendly\n")
	prompt.WriteString("• If the user thanks you or says you already answered, reply in one short sentence only — do NOT repeat facts, numbers, or prior answers\n")
	prompt.WriteString("• For geography, dates, or live data you cannot verify, say you may be approximate and suggest Maps or an authoritative source\n")
	prompt.WriteString("• Remember user preferences and context\n")
	prompt.WriteString("• When asked about meetings, use the available meeting notes to provide accurate information\n")
	prompt.WriteString("• When asked about Neural Junkie features, UI, shortcuts, settings, or workflows, use NEURAL JUNKIE APP KNOWLEDGE and SYSTEM COMMANDS above — give concrete steps and shortcuts\n")
	prompt.WriteString("• NEVER give generic answers about external tools (like GitHub Actions) when the user is asking about THIS system's capabilities\n")
	prompt.WriteString("• Never say a file was saved or applied unless you see Applied change or file_change_approved in this thread\n\n")

	AppendUserAndAgentRules(&prompt, msg, &a.Agent.Info, ResolveUserRulesHubFallback(msg), 0)
	AppendMemoryForMessage(&prompt, msg, a.channelHistory(msg.Channel))
	AppendLearningsForMessage(&prompt, msg, &a.Agent.Info)

	// Insert system/user separator -- everything above is system context,
	// everything below is the user's actual message and workspace data.
	prompt.WriteString(ai.SystemPromptSeparator)

	// Append workspace context if the user shared it
	AppendWorkspaceContext(&prompt, msg)
	appendWorkspaceReviewGuidance(&prompt, msg)
	appendContentDeliveryGuidance(&prompt, msg)

	AppendPromptAttachments(&prompt, msg)
	AppendGrantedHubDataAccess(&prompt, msg)

	if userAsksAboutPromptContext(msg.Content) {
		prompt.WriteString("\nThe user is asking what context or metadata you received for this turn. ")
		prompt.WriteString("Summarize the WORKSPACE CONTEXT section (if any), list relevant metadata keys ")
		prompt.WriteString("(e.g. context_scope, workspace_context), and quote their exact User message below. ")
		prompt.WriteString("Do NOT claim they failed to provide a prompt.\n\n")
	}

	prompt.WriteString(fmt.Sprintf("User message from %s:\n%s\n\n", msg.From.Name, msg.Content))

	// Adaptive response length for assistant too
	prompt.WriteString(getResponseLengthGuidance(msg.Content))

	return prompt.String()
}

// monitorReminders runs a background loop to check for due reminders
func (a *AssistantAgent) monitorReminders(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopReminders:
			return
		case <-ticker.C:
			a.checkDueReminders(ctx)
		}
	}
}

// checkDueReminders checks for reminders that are due and sends them
func (a *AssistantAgent) checkDueReminders(ctx context.Context) {
	if a.storage == nil {
		return
	}

	reminders, err := a.storage.LoadReminders()
	if err != nil {
		log.Printf("[%s] Failed to load reminders: %v", a.Info.Name, err)
		return
	}

	now := time.Now()
	for _, reminder := range reminders {
		if !reminder.Active {
			continue
		}

		// Check if reminder is due
		if reminder.TriggerTime.Before(now) || reminder.TriggerTime.Equal(now) {
			err := a.sendReminder(ctx, reminder)
			deadChannel := err != nil && isReminderChannelMissingError(err)

			if reminder.Recurring != nil {
				if err == nil {
					a.scheduleNextRecurring(reminder)
				} else if deadChannel {
					reminder.Active = false
					if saveErr := a.storage.SaveReminder(reminder); saveErr != nil {
						log.Printf("[%s] Failed to save reminder after channel missing: %v", a.Info.Name, saveErr)
					} else {
						log.Printf("[%s] Reminder deactivated (channel missing): %q → %v", a.Info.Name, reminder.Channel, err)
					}
				}
			} else {
				if err == nil || deadChannel {
					reminder.Active = false
					if saveErr := a.storage.SaveReminder(reminder); saveErr != nil {
						log.Printf("[%s] Failed to save reminder after send: %v", a.Info.Name, saveErr)
					}
				}
			}
		}
	}
}

func isReminderChannelMissingError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "channel ") && strings.Contains(s, " not found")
}

// sendReminder sends a reminder message.
func (a *AssistantAgent) sendReminder(ctx context.Context, reminder *Reminder) error {
	message := fmt.Sprintf("🔔 **Reminder**: %s", reminder.Content)

	msg := protocol.NewMessage(
		protocol.MessageTypeChat,
		reminder.Channel,
		a.Info,
		message,
	)

	if err := a.Hub.SendMessage(msg); err != nil {
		log.Printf("[%s] Failed to send reminder: %v", a.Info.Name, err)
		return err
	}
	return nil
}

// scheduleNextRecurring calculates the next occurrence of a recurring reminder
func (a *AssistantAgent) scheduleNextRecurring(reminder *Reminder) {
	if reminder.Recurring == nil {
		return
	}

	var nextTime time.Time

	switch reminder.Recurring.Type {
	case "daily":
		nextTime = reminder.TriggerTime.AddDate(0, 0, reminder.Recurring.Interval)
	case "weekly":
		nextTime = reminder.TriggerTime.AddDate(0, 0, 7*reminder.Recurring.Interval)
	case "monthly":
		nextTime = reminder.TriggerTime.AddDate(0, reminder.Recurring.Interval, 0)
	default:
		// For now, default to daily
		nextTime = reminder.TriggerTime.AddDate(0, 0, 1)
	}

	reminder.TriggerTime = nextTime
	a.storage.SaveReminder(reminder)
}

// checkProactiveSuggestions monitors conversation for proactive suggestions
func (a *AssistantAgent) checkProactiveSuggestions(ctx context.Context, msg *protocol.Message) {
	if a.storage == nil {
		return
	}

	content := strings.ToLower(msg.Content)

	// Check for meeting mentions
	meetingKeywords := []string{"meeting", "standup", "retro", "planning", "review", "sync"}
	for _, keyword := range meetingKeywords {
		if strings.Contains(content, keyword) {
			a.suggestMeetingReminder(ctx, msg)
			break
		}
	}

	// Check for action items
	actionKeywords := []string{"todo", "task", "need to", "should", "must", "action item"}
	for _, keyword := range actionKeywords {
		if strings.Contains(content, keyword) {
			a.suggestTaskCreation(ctx, msg)
			break
		}
	}

	// Check for important information
	importantKeywords := []string{"important", "note", "remember", "key", "critical"}
	for _, keyword := range importantKeywords {
		if strings.Contains(content, keyword) {
			a.suggestNoteSaving(ctx, msg)
			break
		}
	}

	// Check for meeting history context
	a.checkMeetingHistoryContext(ctx, msg)
}

// checkMeetingHistoryContext provides context-aware suggestions based on meeting history
func (a *AssistantAgent) checkMeetingHistoryContext(ctx context.Context, msg *protocol.Message) {
	if a.storage == nil {
		return
	}

	content := strings.ToLower(msg.Content)

	// Search for relevant meeting notes
	notes, err := a.storage.SearchMeetingNotes(msg.Content)
	if err != nil || len(notes) == 0 {
		return
	}

	// Find the most relevant meeting note
	var relevantNote *MeetingNote
	for _, note := range notes {
		if a.isRelevantToContext(note, content) {
			relevantNote = note
			break
		}
	}

	if relevantNote != nil {
		a.provideMeetingContext(ctx, msg, relevantNote)
	}
}

// isRelevantToContext checks if a meeting note is relevant to the current context
func (a *AssistantAgent) isRelevantToContext(note *MeetingNote, content string) bool {
	// Check if any attendees are mentioned
	for _, attendee := range note.Attendees {
		if strings.Contains(strings.ToLower(content), strings.ToLower(attendee)) {
			return true
		}
	}

	// Check if any topics are mentioned
	for _, topic := range note.Topics {
		if strings.Contains(strings.ToLower(content), strings.ToLower(topic)) {
			return true
		}
	}

	// Check if any action items are mentioned
	for _, action := range note.ActionItems {
		if strings.Contains(strings.ToLower(content), strings.ToLower(action)) {
			return true
		}
	}

	return false
}

// provideMeetingContext provides relevant meeting context to the user
func (a *AssistantAgent) provideMeetingContext(ctx context.Context, msg *protocol.Message, note *MeetingNote) {
	// Check if proactive assistance is enabled
	if !a.config.ProactiveAssistance {
		return
	}

	// Create context message
	contextMessage := fmt.Sprintf("📅 **Relevant Meeting Context**\n\n**Meeting:** %s\n**Date:** %s\n**Attendees:** %s\n\n**Summary:** %s",
		note.Title,
		note.MeetingDate.Format("2006-01-02 15:04"),
		strings.Join(note.Attendees, ", "),
		note.Summary[:minInt(200, len(note.Summary))])

	if len(note.ActionItems) > 0 {
		contextMessage += "\n\n**Action Items:**\n"
		for i, item := range note.ActionItems {
			if i >= 3 { // Limit to 3 action items
				contextMessage += fmt.Sprintf("... and %d more\n", len(note.ActionItems)-3)
				break
			}
			contextMessage += fmt.Sprintf("• %s\n", item)
		}
	}

	// Check if there are pending action items that might be relevant
	if len(note.ActionItems) > 0 {
		contextMessage += "\n\n💡 **Suggestion:** I noticed there are pending action items from this meeting. Would you like me to help you track or complete any of these?"

		// Request approval to create a task or reminder
		action := "Create task from meeting action item"
		description := fmt.Sprintf("Create a task for: %s", note.ActionItems[0])
		a.RequestApproval(ctx, action, description, msg.Channel)
	}

	// Send context message
	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		a.Info,
		contextMessage,
	)
	response.ReplyTo = msg.ID

	a.Hub.SendMessage(response)
}

// suggestMeetingReminder suggests creating a meeting reminder
func (a *AssistantAgent) suggestMeetingReminder(ctx context.Context, msg *protocol.Message) {
	suggestion := "💡 I noticed you mentioned a meeting. Would you like me to help you set up a reminder or add it to your schedule? Use `/meeting-add` or `/remind` to get started!"

	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		a.Info,
		suggestion,
	)
	response.ReplyTo = msg.ID

	a.Hub.SendMessage(response)
}

// suggestTaskCreation suggests creating a task
func (a *AssistantAgent) suggestTaskCreation(ctx context.Context, msg *protocol.Message) {
	suggestion := "📝 I noticed some action items in your message. Would you like me to help you track these as tasks? Use `/task-add` to create a task!"

	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		a.Info,
		suggestion,
	)
	response.ReplyTo = msg.ID

	a.Hub.SendMessage(response)
}

// suggestNoteSaving suggests saving important information
func (a *AssistantAgent) suggestNoteSaving(ctx context.Context, msg *protocol.Message) {
	suggestion := "📋 This seems like important information. Would you like me to save this as a note for future reference? Use `/note-save` to save it!"

	response := protocol.NewMessage(
		protocol.MessageTypeChat,
		msg.Channel,
		a.Info,
		suggestion,
	)
	response.ReplyTo = msg.ID

	a.Hub.SendMessage(response)
}

// ParseTime parses various time formats for reminders
func (a *AssistantAgent) ParseTime(timeStr string) (time.Time, error) {
	now := time.Now()
	timeStr = strings.ToLower(strings.TrimSpace(timeStr))

	// Handle "in X minutes/hours"
	if strings.HasPrefix(timeStr, "in ") {
		return a.parseRelativeTime(timeStr, now)
	}

	// Handle "at X" (time today)
	if strings.HasPrefix(timeStr, "at ") {
		timePart := strings.TrimPrefix(timeStr, "at ")
		return a.parseTimeToday(timePart, now)
	}

	// Handle "tomorrow at X"
	if strings.HasPrefix(timeStr, "tomorrow at ") {
		timePart := strings.TrimPrefix(timeStr, "tomorrow at ")
		tomorrow := now.AddDate(0, 0, 1)
		return a.parseTimeToday(timePart, tomorrow)
	}

	// Handle specific date/time formats
	return a.parseAbsoluteTime(timeStr, now)
}

// parseRelativeTime parses "in X minutes/hours" format
func (a *AssistantAgent) parseRelativeTime(timeStr string, base time.Time) (time.Time, error) {
	// Remove "in " prefix
	timeStr = strings.TrimPrefix(timeStr, "in ")

	// Parse number and unit
	re := regexp.MustCompile(`(?i)^(\d+)\s*(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)$`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) != 3 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	amount, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, err
	}

	unit := matches[2]
	switch unit {
	case "s", "sec", "secs", "second", "seconds":
		return base.Add(time.Duration(amount) * time.Second), nil
	case "m", "min", "mins", "minute", "minutes":
		return base.Add(time.Duration(amount) * time.Minute), nil
	case "h", "hr", "hrs", "hour", "hours":
		return base.Add(time.Duration(amount) * time.Hour), nil
	case "d", "day", "days":
		return base.AddDate(0, 0, amount), nil
	default:
		return time.Time{}, fmt.Errorf("unknown time unit: %s", unit)
	}
}

func (a *AssistantAgent) handleDirectAssistantActions(ctx context.Context, msg *protocol.Message) bool {
	if a.storage == nil {
		return false
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(content), "/") {
		return false
	}

	if reminder, ok := a.buildReminderFromNaturalLanguage(msg, content); ok {
		if err := a.storage.SaveReminder(reminder); err != nil {
			log.Printf("[%s] Failed to save reminder from natural language: %v", a.Info.Name, err)
			errMsg := protocol.NewMessage(
				protocol.MessageTypeSystemInfo,
				msg.Channel,
				a.Info,
				fmt.Sprintf("❌ Failed to persist reminder: %v", err),
			)
			errMsg.ReplyTo = msg.ID
			a.Hub.SendMessage(errMsg)
			return true
		}
		confirm := protocol.NewMessage(
			protocol.MessageTypeSystemInfo,
			msg.Channel,
			a.Info,
			fmt.Sprintf("⏰ Reminder set: '%s' at %s", reminder.Content, reminder.TriggerTime.Format(time.RFC1123)),
		)
		confirm.ReplyTo = msg.ID
		a.Hub.SendMessage(confirm)
		return true
	}

	if task, ok := a.buildTaskFromNaturalLanguage(msg, content); ok {
		if err := a.storage.SaveTask(task); err != nil {
			log.Printf("[%s] Failed to save task from natural language: %v", a.Info.Name, err)
			errMsg := protocol.NewMessage(
				protocol.MessageTypeSystemInfo,
				msg.Channel,
				a.Info,
				fmt.Sprintf("❌ Failed to persist task: %v", err),
			)
			errMsg.ReplyTo = msg.ID
			a.Hub.SendMessage(errMsg)
			return true
		}
		confirm := protocol.NewMessage(
			protocol.MessageTypeSystemInfo,
			msg.Channel,
			a.Info,
			fmt.Sprintf("📝 Task added: [%s] %s", assistantShortID(task.ID), task.Title),
		)
		confirm.ReplyTo = msg.ID
		a.Hub.SendMessage(confirm)
		return true
	}

	// Prevent "I set it" hallucinations when a reminder intent is present but
	// the time/message parse fails.
	if isLikelyReminderIntent(content) {
		confirm := protocol.NewMessage(
			protocol.MessageTypeSystemInfo,
			msg.Channel,
			a.Info,
			"❌ I couldn't parse that reminder. Try formats like `remind me in 30s to stand up` or `/remind in 30s stand up`.",
		)
		confirm.ReplyTo = msg.ID
		a.Hub.SendMessage(confirm)
		return true
	}

	return false
}

func (a *AssistantAgent) buildReminderFromNaturalLanguage(msg *protocol.Message, content string) (*Reminder, bool) {
	normalized := strings.ToLower(strings.TrimSpace(content))
	rest := ""
	for _, prefix := range []string{
		"remind me ",
		"reminder me ",
		"reminde me ",
		"remindme ",
		"set a reminder for ",
		"set reminder for ",
		"set a reminder ",
		"set reminder ",
		"ping me ",
		"nudge me ",
		"alert me ",
	} {
		if strings.HasPrefix(normalized, prefix) {
			rest = strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
			break
		}
	}
	if rest == "" {
		return nil, false
	}

	parts := strings.Fields(rest)
	if len(parts) < 2 {
		if trigger, ok := a.tryParseReminderTime(rest); ok {
			return &Reminder{
				ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
				Content:     "Reminder",
				TriggerTime: trigger,
				Channel:     msg.Channel,
				CreatedBy:   msg.From.Name,
				Active:      true,
				CreatedAt:   time.Now(),
			}, true
		}
		return nil, false
	}

	var (
		trigger time.Time
		message string
	)
	if left, right, ok := splitReminderTimeAndMessage(rest); ok {
		if parsed, parseOK := a.tryParseReminderTime(left); parseOK {
			trigger = parsed
			message = right
		}
	}

	for i := len(parts) - 1; i >= 1; i-- {
		if message != "" {
			break
		}
		timeExpr := strings.Join(parts[:i], " ")
		parsed, ok := a.tryParseReminderTime(timeExpr)
		if !ok {
			continue
		}
		reminderText := strings.TrimSpace(strings.Join(parts[i:], " "))
		reminderText = strings.TrimSpace(strings.TrimPrefix(reminderText, "to "))
		if reminderText == "" {
			continue
		}
		trigger = parsed
		message = reminderText
		break
	}
	if message == "" {
		parsed, ok := a.tryParseReminderTime(rest)
		if !ok {
			return nil, false
		}
		trigger = parsed
		message = "Reminder"
	}

	return &Reminder{
		ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Content:     message,
		TriggerTime: trigger,
		Channel:     msg.Channel,
		CreatedBy:   msg.From.Name,
		Active:      true,
		CreatedAt:   time.Now(),
	}, true
}

func isLikelyReminderIntent(content string) bool {
	normalized := strings.ToLower(strings.TrimSpace(content))
	if !(strings.HasPrefix(normalized, "remind") ||
		strings.HasPrefix(normalized, "reminde") ||
		strings.HasPrefix(normalized, "set reminder") ||
		strings.HasPrefix(normalized, "set a reminder") ||
		strings.HasPrefix(normalized, "ping me") ||
		strings.HasPrefix(normalized, "nudge me") ||
		strings.HasPrefix(normalized, "alert me")) {
		return false
	}
	hasTimeSignal := strings.Contains(normalized, " in ") || strings.Contains(normalized, " at ")
	if !hasTimeSignal {
		timeToken := regexp.MustCompile(`\b\d+\s*(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)\b`)
		hasTimeSignal = timeToken.MatchString(normalized)
	}
	return hasTimeSignal
}

func splitReminderTimeAndMessage(rest string) (string, string, bool) {
	for _, sep := range []string{" to ", " about "} {
		parts := strings.SplitN(rest, sep, 2)
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		left = strings.TrimSpace(strings.TrimPrefix(left, "for "))
		right := strings.TrimSpace(parts[1])
		if left == "" || right == "" {
			continue
		}
		return left, right, true
	}
	return "", "", false
}

func (a *AssistantAgent) tryParseReminderTime(expr string) (time.Time, bool) {
	candidates := normalizeReminderTimeCandidates(expr)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		parsed, err := a.ParseTime(candidate)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func normalizeReminderTimeCandidates(expr string) []string {
	raw := strings.TrimSpace(strings.ToLower(expr))
	if raw == "" {
		return nil
	}

	candidates := []string{raw}
	trimmedFor := strings.TrimSpace(strings.TrimPrefix(raw, "for "))
	if trimmedFor != raw {
		candidates = append(candidates, trimmedFor)
	}

	relative := regexp.MustCompile(`^(\d+\s*(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days))(?:\s+from\s+now)?$`)
	if m := relative.FindStringSubmatch(trimmedFor); len(m) > 1 {
		candidates = append(candidates, "in "+strings.TrimSpace(m[1]))
	}

	// Deduplicate while preserving order.
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

func (a *AssistantAgent) buildTaskFromNaturalLanguage(msg *protocol.Message, content string) (*Task, bool) {
	trimmed := strings.TrimSpace(content)
	lowerTrimmed := strings.ToLower(trimmed)
	var title string
	switch {
	case strings.HasPrefix(lowerTrimmed, "add task "):
		title = strings.TrimSpace(trimmed[len("add task "):])
	case strings.HasPrefix(lowerTrimmed, "create task "):
		title = strings.TrimSpace(trimmed[len("create task "):])
	case strings.HasPrefix(lowerTrimmed, "task:"):
		title = strings.TrimSpace(trimmed[len("task:"):])
	default:
		return nil, false
	}
	if title == "" {
		return nil, false
	}

	return &Task{
		ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:     title,
		Priority:  3,
		Status:    "todo",
		CreatedAt: time.Now(),
		Channel:   msg.Channel,
		CreatedBy: msg.From.Name,
	}, true
}

func assistantShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// parseTimeToday parses time for a specific day
func (a *AssistantAgent) parseTimeToday(timeStr string, day time.Time) (time.Time, error) {
	// Try parsing as "HH:MM" or "H:MM AM/PM"
	layouts := []string{
		"15:04",
		"3:04 PM",
		"3:04pm",
		"3:04 AM",
		"3:04am",
		"3 PM",
		"3pm",
		"3 AM",
		"3am",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, timeStr); err == nil {
			// Combine the date from 'day' with the time from 'parsed'
			return time.Date(day.Year(), day.Month(), day.Day(),
				parsed.Hour(), parsed.Minute(), 0, 0, day.Location()), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// parseAbsoluteTime parses absolute date/time formats
func (a *AssistantAgent) parseAbsoluteTime(timeStr string, _ time.Time) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04",
		"2006-01-02 3:04 PM",
		"Jan 2 15:04",
		"Jan 2 3:04 PM",
		"January 2, 2006 15:04",
		"January 2, 2006 3:04 PM",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, timeStr); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse absolute time: %s", timeStr)
}

// Stop gracefully stops the assistant agent
func (a *AssistantAgent) Stop() {
	close(a.stopReminders)
	close(a.stopGoogleSync)
	if a.emailWatcher != nil {
		a.emailWatcher.Close()
	}
	close(a.stopEmailWatcher)
	if a.unansweredTracker != nil {
		a.unansweredTracker.stop()
	}
	a.Agent.Stop()
}

// analyzeMeetingContent uses AI to extract structured data from meeting content
func (a *AssistantAgent) analyzeMeetingContent(content string) (summary string, attendees []string, actionItems []string, topics []string, err error) {
	prompt := fmt.Sprintf(`Analyze this meeting transcript and extract the following information. Return ONLY valid JSON, no explanatory text:

{
  "summary": "Brief summary of the meeting",
  "attendees": ["Name1", "Name2", "Name3"],
  "action_items": ["Action item 1", "Action item 2"],
  "topics": ["Topic1", "Topic2", "Topic3"]
}

Meeting content:
%s`, content)

	response, err := a.AI.GenerateResponse(context.Background(), prompt, nil)
	if err != nil {
		log.Printf("AI analysis failed: %v", err)
		return "", nil, nil, nil, err
	}

	log.Printf("AI analysis response: %s", response)

	// Try to parse JSON response
	var result struct {
		Summary     string   `json:"summary"`
		Attendees   []string `json:"attendees"`
		ActionItems []string `json:"action_items"`
		Topics      []string `json:"topics"`
	}

	// Try to extract JSON from response (handle cases where AI adds explanatory text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStr := response[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			log.Printf("Failed to parse JSON response, using basic parsing: %v", err)
			// Fallback to basic parsing
			summary = a.extractBasicSummary(content)
			attendees = a.extractBasicAttendees(content)
			actionItems = a.extractBasicActionItems(content)
			topics = a.extractBasicTopics(content)
			return summary, attendees, actionItems, topics, nil
		}
	} else {
		log.Printf("No JSON found in response, using basic parsing")
		// Fallback to basic parsing
		summary = a.extractBasicSummary(content)
		attendees = a.extractBasicAttendees(content)
		actionItems = a.extractBasicActionItems(content)
		topics = a.extractBasicTopics(content)
		return summary, attendees, actionItems, topics, nil
	}

	return result.Summary, result.Attendees, result.ActionItems, result.Topics, nil
}

// extractBasicSummary extracts a basic summary from meeting content
func (a *AssistantAgent) extractBasicSummary(content string) string {
	lines := strings.Split(content, "\n")

	// Look for lines that contain "summary" but extract the content after it
	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "summary") {
			// Extract content after "summary:" or "summary -" etc.
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				summary := strings.TrimSpace(parts[1])
				if summary != "" {
					return summary
				}
			}
		}
	}

	// If no summary line found, return first few meaningful lines
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && len(trimmed) > 10 {
			return trimmed
		}
	}

	// Last resort: return first 200 characters
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}

// extractBasicAttendees extracts attendee names from content
func (a *AssistantAgent) extractBasicAttendees(content string) []string {
	var attendees []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for common patterns that might indicate attendees
		if strings.Contains(strings.ToLower(line), "led") ||
			strings.Contains(strings.ToLower(line), "discussed") ||
			strings.Contains(strings.ToLower(line), "confirmed") {
			// Extract potential names (simplified)
			words := strings.Fields(line)
			for _, word := range words {
				if len(word) > 2 && word[0] >= 'A' && word[0] <= 'Z' {
					attendees = append(attendees, word)
				}
			}
		}
	}

	return attendees
}

// extractBasicActionItems extracts action items from content
func (a *AssistantAgent) extractBasicActionItems(content string) []string {
	var actionItems []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "action") ||
			strings.Contains(strings.ToLower(line), "todo") ||
			strings.Contains(strings.ToLower(line), "follow up") {
			actionItems = append(actionItems, line)
		}
	}

	return actionItems
}

// extractBasicTopics extracts topics from content
func (a *AssistantAgent) extractBasicTopics(content string) []string {
	var topics []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines that might contain topics
		if len(line) > 10 && len(line) < 100 {
			topics = append(topics, line)
		}
	}

	return topics[:minInt(5, len(topics))] // Limit to 5 topics
}

// sendMeetingNoteNotification sends a notification about a new meeting note
func (a *AssistantAgent) sendMeetingNoteNotification(ctx context.Context, note *MeetingNote) {
	message := fmt.Sprintf("📝 New meeting note ingested: **%s**\n\n**Date:** %s\n**Attendees:** %s\n**Summary:** %s",
		note.Title,
		note.MeetingDate.Format("2006-01-02 15:04"),
		strings.Join(note.Attendees, ", "),
		note.Summary[:minInt(200, len(note.Summary))])

	// Send to the default channel
	msg := &protocol.Message{
		ID:        fmt.Sprintf("meeting_note_%d", time.Now().UnixNano()),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   a.config.DefaultChannel,
		From:      a.Info,
		Content:   message,
		Timestamp: time.Now(),
	}

	a.Hub.SendMessage(msg)
}

// sendMeetingNotesBatchNotification posts one summary instead of N startup messages.
func (a *AssistantAgent) sendMeetingNotesBatchNotification(ctx context.Context, notes []*MeetingNote) {
	if len(notes) == 0 {
		return
	}
	if len(notes) == 1 {
		a.sendMeetingNoteNotification(ctx, notes[0])
		return
	}

	const maxListed = 12
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📝 Ingested **%d** meeting notes on startup:\n\n", len(notes)))
	for i, note := range notes {
		if i >= maxListed {
			b.WriteString(fmt.Sprintf("\n… and **%d** more (already in assistant storage).\n", len(notes)-maxListed))
			break
		}
		summary := note.Summary
		if len(summary) > 80 {
			summary = summary[:80] + "…"
		}
		b.WriteString(fmt.Sprintf("• **%s** — %s\n", note.Title, note.MeetingDate.Format("2006-01-02")))
		if summary != "" {
			b.WriteString(fmt.Sprintf("  _%s_\n", summary))
		}
	}

	msg := &protocol.Message{
		ID:        fmt.Sprintf("meeting_notes_batch_%d", time.Now().UnixNano()),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   a.config.DefaultChannel,
		From:      a.Info,
		Content:   b.String(),
		Timestamp: time.Now(),
	}
	a.Hub.SendMessage(msg)
}

// messageAsksAboutMeetings reports whether the user message is about meeting notes.
func messageAsksAboutMeetings(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	meetingKeywords := []string{
		"meeting", "last meeting", "recent meeting",
		"summarize notes", "meeting summary", "meeting notes",
		"what happened in", "discussed in", "decided in",
		"action items from", "follow up from", "next steps from",
	}
	for _, keyword := range meetingKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	if strings.Contains(lower, "notes") && (strings.Contains(lower, "today") || strings.Contains(lower, "from")) {
		return true
	}
	return false
}

// messageAsksAboutEmail reports whether the user message is about recent email.
func messageAsksAboutEmail(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	emailKeywords := []string{
		"email", "emails", "inbox", "mail from", "recent mail",
		"my messages", "unread",
	}
	for _, keyword := range emailKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// detectsMeetingQuery determines if a message is asking about meetings
func (a *AssistantAgent) detectsMeetingQuery(msg *protocol.Message) bool {
	if msg == nil || assistantSkipPersonalContextEnrichment(msg) {
		return false
	}
	return messageAsksAboutMeetings(msg.Content)
}

// detectsEmailQuery determines if a message is asking about email.
func (a *AssistantAgent) detectsEmailQuery(msg *protocol.Message) bool {
	if msg == nil || assistantSkipPersonalContextEnrichment(msg) {
		return false
	}
	return messageAsksAboutEmail(msg.Content)
}

// detectsNJAppQuery determines if a message is asking about the Neural Junkie application.
func (a *AssistantAgent) detectsNJAppQuery(msg *protocol.Message) bool {
	if msg == nil || assistantSkipPersonalContextEnrichment(msg) {
		return false
	}
	return messageAsksAboutNJApp(msg.Content)
}

// buildMeetingContextPrompt creates an enriched prompt with full meeting content
func (a *AssistantAgent) buildMeetingContextPrompt(msg *protocol.Message) string {
	if a.storage == nil {
		return a.buildAssistantPromptCore(msg, true)
	}

	// Load meeting notes
	meetingNotes, err := a.storage.LoadMeetingNotes()
	if err != nil || len(meetingNotes) == 0 {
		log.Printf("⚠️  [Assistant] No meeting notes available for context")
		return a.buildAssistantPromptCore(msg, true)
	}

	now := assistantPromptTime(a.config)
	todayQuery := meetingQueryAsksAboutToday(msg.Content)
	todayNotes := meetingNotesOnDate(meetingNotes, now)
	selectedNotes, matched := selectMeetingNotesForPrompt(meetingNotes, msg.Content, 3)

	var prompt strings.Builder

	AppendUserAndAgentRules(&prompt, msg, &a.Agent.Info, ResolveUserRulesHubFallback(msg), 0)

	prompt.WriteString("You are the Assistant, a helpful AI in the Neural Junkie multi-agent collaboration system.\n\n")
	prompt.WriteString("=== YOUR ROLE ===\n")
	prompt.WriteString("You help users stay organized and productive by managing reminders, tasks, notes, and schedules. ")
	prompt.WriteString("You also serve as a knowledgeable guide to the Neural Junkie system itself. ")
	prompt.WriteString("You are friendly, proactive, and always ready to help.\n\n")

	prompt.WriteString("=== CURRENT DATE/TIME ===\n")
	prompt.WriteString(fmt.Sprintf("Current date: %s\n", now.Format("Monday, January 2, 2006")))
	prompt.WriteString(fmt.Sprintf("Current time: %s\n", now.Format("15:04 MST")))
	prompt.WriteString("Interpret relative date phrases such as \"today\", \"yesterday\", and \"last meeting\" using this date/time. Do not invent a different current date.\n\n")

	prompt.WriteString("=== MEETING NOTES CONTEXT ===\n")
	prompt.WriteString("The user is asking about synced meeting notes. You have access to the meeting notes listed below from Assistant storage.\n")
	prompt.WriteString("If notes are listed here, do not say you cannot access meeting notes, files, or prior meetings.\n")
	prompt.WriteString("For questions like \"do you see/have <meeting>\", answer yes/no first, then summarize what was found.\n")
	if todayQuery {
		if len(todayNotes) > 0 {
			prompt.WriteString(fmt.Sprintf("The user asked about today's meeting notes. Some synced meeting notes are dated today (%s); only call a listed note today's note when its Date matches today.\n", now.Format("January 2, 2006")))
		} else {
			prompt.WriteString(fmt.Sprintf("The user asked about today's meeting notes. No synced meeting notes are dated today (%s). The notes listed below are older context; do not describe them as today's notes.\n", now.Format("January 2, 2006")))
		}
	}
	if matched {
		prompt.WriteString("These are the best matching synced notes for the user's query:\n\n")
	} else {
		prompt.WriteString("No exact title/content match was found, so the most recent synced notes are included for context:\n\n")
	}

	for i, note := range selectedNotes {
		prompt.WriteString(fmt.Sprintf("**Meeting %d: %s**\n", i+1, note.Title))
		prompt.WriteString(fmt.Sprintf("**Date:** %s\n", note.MeetingDate.Format("Jan 2, 2006 15:04")))
		prompt.WriteString(fmt.Sprintf("**Attendees:** %s\n", strings.Join(note.Attendees, ", ")))

		if note.Summary != "" {
			prompt.WriteString(fmt.Sprintf("**Summary:** %s\n", note.Summary))
		}

		if len(note.ActionItems) > 0 {
			prompt.WriteString("**Action Items:**\n")
			for _, item := range note.ActionItems {
				prompt.WriteString(fmt.Sprintf("• %s\n", item))
			}
		}

		if len(note.Topics) > 0 {
			prompt.WriteString("**Topics:**\n")
			for _, topic := range note.Topics {
				prompt.WriteString(fmt.Sprintf("• %s\n", topic))
			}
		}

		if note.GoogleDocLink != "" {
			prompt.WriteString(fmt.Sprintf("**Google Doc:** %s\n", note.GoogleDocLink))
		}

		// Include full content for the best match/recent note. Additional notes keep
		// compact metadata to avoid drowning smaller local models.
		if i == 0 && strings.TrimSpace(note.FullContent) != "" {
			prompt.WriteString("**Full Meeting Content:**\n")
			prompt.WriteString(note.FullContent)
			prompt.WriteString("\n")
		}

		prompt.WriteString("\n")
	}

	prompt.WriteString("=== INSTRUCTIONS ===\n")
	prompt.WriteString("Use the meeting information above to provide accurate, detailed responses about the meetings. ")
	prompt.WriteString("When asked about 'last meeting' or 'recent meeting', refer to the most recent meeting by date. ")
	prompt.WriteString("Be specific about attendees, decisions made, action items, and next steps.\n\n")

	AppendPromptAttachments(&prompt, msg)
	AppendGrantedHubDataAccess(&prompt, msg)

	prompt.WriteString(fmt.Sprintf("User message from %s:\n%s\n\n", msg.From.Name, msg.Content))
	prompt.WriteString("Provide a helpful response based on the meeting information:")

	return prompt.String()
}

func selectMeetingNotesForPrompt(notes []*MeetingNote, query string, limit int) ([]*MeetingNote, bool) {
	if limit <= 0 || len(notes) == 0 {
		return nil, false
	}

	sorted := append([]*MeetingNote(nil), notes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].MeetingDate.After(sorted[j].MeetingDate)
	})

	tokens := meetingQueryTokens(query)
	type scoredNote struct {
		note      *MeetingNote
		score     int
		titleHits int
	}
	var scored []scoredNote
	for _, note := range sorted {
		score, titleHits := scoreMeetingNoteForQuery(note, query, tokens)
		if score > 0 {
			scored = append(scored, scoredNote{note: note, score: score, titleHits: titleHits})
		}
	}
	if len(scored) == 0 {
		return sorted[:minInt(limit, len(sorted))], false
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].titleHits != scored[j].titleHits {
			return scored[i].titleHits > scored[j].titleHits
		}
		if scored[i].titleHits > 0 && !scored[i].note.MeetingDate.Equal(scored[j].note.MeetingDate) {
			return scored[i].note.MeetingDate.After(scored[j].note.MeetingDate)
		}
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].note.MeetingDate.After(scored[j].note.MeetingDate)
	})

	out := make([]*MeetingNote, 0, minInt(limit, len(scored)))
	for _, item := range scored {
		out = append(out, item.note)
		if len(out) >= limit {
			break
		}
	}
	return out, true
}

func assistantPromptTime(config *AssistantConfig) time.Time {
	now := assistantPromptNow()
	location := time.Local
	if config != nil && strings.TrimSpace(config.Timezone) != "" {
		if loaded, err := time.LoadLocation(config.Timezone); err == nil {
			location = loaded
		}
	}
	return now.In(location)
}

func meetingQueryAsksAboutToday(query string) bool {
	for _, field := range strings.Fields(normalizeMeetingSearchText(query)) {
		if field == "today" {
			return true
		}
	}
	return false
}

func meetingNotesOnDate(notes []*MeetingNote, target time.Time) []*MeetingNote {
	if len(notes) == 0 {
		return nil
	}
	location := target.Location()
	var matches []*MeetingNote
	for _, note := range notes {
		if note == nil {
			continue
		}
		noteDate := note.MeetingDate.In(location)
		if noteDate.Year() == target.Year() && noteDate.YearDay() == target.YearDay() {
			matches = append(matches, note)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].MeetingDate.After(matches[j].MeetingDate)
	})
	return matches
}

func scoreMeetingNoteForQuery(note *MeetingNote, query string, tokens []string) (int, int) {
	if note == nil {
		return 0, 0
	}
	queryNorm := normalizeMeetingSearchText(query)
	titleNorm := normalizeMeetingSearchText(note.Title)
	score := 0
	titleHits := 0
	if titleNorm != "" && strings.Contains(queryNorm, titleNorm) {
		score += 200
	}

	summaryNorm := normalizeMeetingSearchText(note.Summary)
	fullNorm := normalizeMeetingSearchText(note.FullContent)
	for _, tok := range tokens {
		if containsMeetingToken(titleNorm, tok) {
			titleHits++
			score += 30
		}
		if containsMeetingToken(summaryNorm, tok) {
			score += 8
		}
		if containsMeetingToken(fullNorm, tok) {
			score += 4
		}
		for _, topic := range note.Topics {
			if containsMeetingToken(normalizeMeetingSearchText(topic), tok) {
				score += 12
			}
		}
		for _, action := range note.ActionItems {
			if containsMeetingToken(normalizeMeetingSearchText(action), tok) {
				score += 10
			}
		}
		for _, attendee := range note.Attendees {
			if containsMeetingToken(normalizeMeetingSearchText(attendee), tok) {
				score += 10
			}
		}
	}
	return score, titleHits
}

func meetingQueryTokens(query string) []string {
	normalized := normalizeMeetingSearchText(query)
	if normalized == "" {
		return nil
	}
	stop := map[string]bool{
		"a": true, "about": true, "an": true, "and": true, "any": true, "are": true,
		"can": true, "could": true, "do": true, "does": true, "for": true, "from": true,
		"have": true, "i": true, "in": true, "is": true, "it": true, "me": true,
		"meeting": true, "meetings": true, "note": true, "notes": true, "of": true,
		"on": true, "please": true, "see": true, "that": true, "the": true,
		"this": true, "today": true, "was": true, "we": true, "what": true, "with": true,
		"you": true,
	}
	seen := map[string]bool{}
	var tokens []string
	for _, tok := range strings.Fields(normalized) {
		if len(tok) < 3 || stop[tok] || seen[tok] {
			continue
		}
		seen[tok] = true
		tokens = append(tokens, tok)
	}
	return tokens
}

func normalizeMeetingSearchText(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	lastSpace := true
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func containsMeetingToken(normalized, token string) bool {
	if normalized == "" || token == "" {
		return false
	}
	for _, field := range strings.Fields(normalized) {
		if field == token {
			return true
		}
	}
	return false
}

// buildEmailContextPrompt creates a prompt with recent email content when the user asks about mail.
func (a *AssistantAgent) buildEmailContextPrompt(msg *protocol.Message) string {
	if a.storage == nil {
		return a.buildAssistantPromptCore(msg, true)
	}
	recentEmails, err := a.storage.GetRecentEmails(7)
	if err != nil || len(recentEmails) == 0 {
		log.Printf("⚠️  [Assistant] No recent emails available for context")
		return a.buildAssistantPromptCore(msg, true)
	}

	sort.Slice(recentEmails, func(i, j int) bool {
		return recentEmails[i].Date.After(recentEmails[j].Date)
	})

	var prompt strings.Builder
	prompt.WriteString("You are the Assistant in Neural Junkie.\n\n")
	prompt.WriteString("=== RECENT EMAILS (Last 7 Days) ===\n")
	for i, email := range recentEmails {
		if i >= 5 {
			break
		}
		prompt.WriteString(fmt.Sprintf("\n## Email %d: %s\n", i+1, email.Subject))
		prompt.WriteString(fmt.Sprintf("**From:** %s\n", email.From))
		prompt.WriteString(fmt.Sprintf("**Date:** %s\n", email.Date.Format("January 2, 2006 15:04")))
		body := email.Body
		if len(body) > 400 {
			body = body[:400] + "..."
		}
		prompt.WriteString(fmt.Sprintf("**Content:** %s\n", body))
	}
	prompt.WriteString("\nUse this information to answer the user's email question.\n\n")
	prompt.WriteString(fmt.Sprintf("User message from %s:\n%s\n", msg.From.Name, msg.Content))
	return prompt.String()
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetConfig returns the assistant agent's configuration
func (a *AssistantAgent) GetConfig() *AssistantConfig {
	return a.config
}

// ProcessMeetingNote triggers a Google meet notes sync (legacy name for /ingest-meetings).
func (a *AssistantAgent) ProcessMeetingNote(ctx context.Context, _ string) {
	_, _ = a.SyncGoogleMeetNotes(ctx)
}

// SearchMeetingNotes searches meeting notes by query
func (a *AssistantAgent) SearchMeetingNotes(query string) ([]*MeetingNote, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}
	return a.storage.SearchMeetingNotes(query)
}

// GetMeetingNotesByDate gets meeting notes for a specific date
func (a *AssistantAgent) GetMeetingNotesByDate(date time.Time) ([]*MeetingNote, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	// Get start and end of day
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)

	return a.storage.GetMeetingNotesByDateRange(start, end)
}

// GetPendingActionItems gets all pending action items from meeting notes
func (a *AssistantAgent) GetPendingActionItems() ([]string, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}
	return a.storage.GetPendingActionItems()
}

// GetRecentMeetingNotes gets recent meeting notes
func (a *AssistantAgent) GetRecentMeetingNotes(limit int) ([]*MeetingNote, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	notes, err := a.storage.LoadMeetingNotes()
	if err != nil {
		return nil, err
	}

	// Sort by date (most recent first) and limit
	if len(notes) > limit {
		notes = notes[:limit]
	}

	return notes, nil
}

// RequestApproval requests user approval for a proposed action
func (a *AssistantAgent) RequestApproval(ctx context.Context, action, description, channel string) string {
	approvalID := fmt.Sprintf("approval_%d", time.Now().UnixNano())

	approval := &PendingApproval{
		ID:          approvalID,
		Action:      action,
		Description: description,
		RequestedBy: a.Info.Name,
		RequestedAt: time.Now(),
		Channel:     channel,
	}

	a.approvalMutex.Lock()
	a.pendingApprovals[approvalID] = approval
	a.approvalMutex.Unlock()

	// Send approval request message
	message := fmt.Sprintf("🤔 **Approval Request**\n\n**Action:** %s\n**Description:** %s\n\n**Approve:** `/approve %s`\n**Reject:** `/reject %s`",
		action, description, approvalID, approvalID)

	msg := &protocol.Message{
		ID:        fmt.Sprintf("approval_request_%d", time.Now().UnixNano()),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   channel,
		From:      a.Info,
		Content:   message,
		Timestamp: time.Now(),
	}

	a.Hub.SendMessage(msg)

	return approvalID
}

// HandleApproval handles approval/rejection responses
func (a *AssistantAgent) HandleApproval(ctx context.Context, approvalID string, approved bool, userID string) error {
	a.approvalMutex.Lock()
	approval, exists := a.pendingApprovals[approvalID]
	a.approvalMutex.Unlock()

	if !exists {
		return fmt.Errorf("approval request not found: %s", approvalID)
	}

	// Remove from pending approvals
	a.approvalMutex.Lock()
	delete(a.pendingApprovals, approvalID)
	a.approvalMutex.Unlock()

	// Send response
	status := "❌ Rejected"
	if approved {
		status = "✅ Approved"
	}

	message := fmt.Sprintf("%s **Approval Response**\n\n**Action:** %s\n**Status:** %s\n**User:** %s",
		status, approval.Action, status, userID)

	msg := &protocol.Message{
		ID:        fmt.Sprintf("approval_response_%d", time.Now().UnixNano()),
		Type:      protocol.MessageTypeSystemInfo,
		Channel:   approval.Channel,
		From:      a.Info,
		Content:   message,
		Timestamp: time.Now(),
	}

	a.Hub.SendMessage(msg)

	// If approved, execute the action
	if approved {
		return a.executeApprovedAction(ctx, approval)
	}

	return nil
}

// executeApprovedAction executes the approved action
func (a *AssistantAgent) executeApprovedAction(ctx context.Context, approval *PendingApproval) error {
	// This is where you would implement the actual action execution
	// For now, just log the action
	log.Printf("Executing approved action: %s - %s", approval.Action, approval.Description)

	// Could integrate with external tools here to execute the actual action

	return nil
}

// GetPendingApprovals returns all pending approvals for a user
func (a *AssistantAgent) GetPendingApprovals(userID string) []*PendingApproval {
	a.approvalMutex.RLock()
	defer a.approvalMutex.RUnlock()

	var userApprovals []*PendingApproval
	for _, approval := range a.pendingApprovals {
		// In a real implementation, you'd filter by userID
		userApprovals = append(userApprovals, approval)
	}

	return userApprovals
}

// startEmailWatcher initializes and starts the file system watcher for emails
func (a *AssistantAgent) startEmailWatcher(ctx context.Context) error {
	log.Printf("🔍 [Assistant] Creating email file system watcher...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create email watcher: %w", err)
	}

	a.emailWatcher = watcher

	// Add the email directory to the watcher
	log.Printf("📂 [Assistant] Adding email directory to watcher: %s", a.config.EmailDir)
	if err := watcher.Add(a.config.EmailDir); err != nil {
		return fmt.Errorf("failed to add email directory to watcher: %w", err)
	}

	// Start the watcher goroutine
	go a.watchEmails(ctx)

	// Load existing emails on startup
	log.Printf("📥 [Assistant] Starting ingestion of existing emails...")
	go a.ingestExistingEmails(ctx)

	return nil
}

// watchEmails monitors the email directory for changes
func (a *AssistantAgent) watchEmails(ctx context.Context) {
	defer a.emailWatcher.Close()

	for {
		select {
		case event, ok := <-a.emailWatcher.Events:
			if !ok {
				return
			}
			a.handleEmailEvent(ctx, event)
		case err, ok := <-a.emailWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Email watcher error: %v", err)
		case <-a.stopEmailWatcher:
			return
		case <-ctx.Done():
			return
		}
	}
}

// handleEmailEvent processes file system events for emails
func (a *AssistantAgent) handleEmailEvent(ctx context.Context, event fsnotify.Event) {
	// Only process email files (common extensions: .eml, .msg, .txt)
	ext := strings.ToLower(filepath.Ext(event.Name))
	if ext != ".eml" && ext != ".msg" && ext != ".txt" {
		return
	}

	// Check if this is a write or create event
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		// Check if we've already processed this file
		a.processedEmailsMutex.RLock()
		alreadyProcessed := a.processedEmails[event.Name]
		a.processedEmailsMutex.RUnlock()

		if !alreadyProcessed {
			// Mark as processed
			a.processedEmailsMutex.Lock()
			a.processedEmails[event.Name] = true
			a.processedEmailsMutex.Unlock()

			// Process the email file
			go a.processEmail(ctx, event.Name)
		}
	}
}

// ingestExistingEmails loads all existing emails on startup
func (a *AssistantAgent) ingestExistingEmails(ctx context.Context) {
	// Wait a bit for the watcher to be fully initialized
	time.Sleep(1 * time.Second)

	log.Printf("🔍 [Assistant] Searching for emails in: %s", a.config.EmailDir)

	// Look for common email file extensions
	patterns := []string{"*.eml", "*.msg", "*.txt"}
	var allFiles []string

	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(a.config.EmailDir, pattern))
		if err != nil {
			log.Printf("❌ [Assistant] Failed to list email files with pattern %s: %v", pattern, err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	log.Printf("📄 [Assistant] Found %d email files to process", len(allFiles))
	for i, file := range allFiles {
		log.Printf("📧 [Assistant] Processing email %d/%d: %s", i+1, len(allFiles), filepath.Base(file))

		// Check if already processed
		a.processedEmailsMutex.RLock()
		alreadyProcessed := a.processedEmails[file]
		a.processedEmailsMutex.RUnlock()

		if !alreadyProcessed {
			// Mark as processed
			a.processedEmailsMutex.Lock()
			a.processedEmails[file] = true
			a.processedEmailsMutex.Unlock()

			a.processEmail(ctx, file)
		} else {
			log.Printf("⏭️  [Assistant] Email already processed: %s", filepath.Base(file))
		}
	}
	log.Printf("✅ [Assistant] Completed ingestion of existing emails")
}

// processEmail parses and stores an email file
func (a *AssistantAgent) processEmail(ctx context.Context, filePath string) {
	log.Printf("🔄 [Assistant] Processing email: %s", filepath.Base(filePath))

	// Read the email file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("❌ [Assistant] Failed to read email file %s: %v", filePath, err)
		return
	}

	// Parse the email
	email, err := a.parseEmail(filePath, string(content))
	if err != nil {
		log.Printf("❌ [Assistant] Failed to parse email %s: %v", filePath, err)
		return
	}

	// Save to storage
	if err := a.storage.SaveEmail(email); err != nil {
		log.Printf("❌ [Assistant] Failed to save email %s: %v", filePath, err)
		return
	}

	log.Printf("✅ [Assistant] Successfully processed email: %s", email.Subject)
}

// parseEmail extracts structured data from an email file
func (a *AssistantAgent) parseEmail(filePath, content string) (*Email, error) {
	// For now, create a basic email structure
	// In a real implementation, you'd parse the actual email format (MIME, etc.)

	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Try to extract date from filename if it follows a pattern
	var emailDate time.Time
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	matches := dateRegex.FindStringSubmatch(nameWithoutExt)
	if len(matches) > 1 {
		if parsed, err := time.Parse("2006-01-02", matches[1]); err == nil {
			emailDate = parsed
		}
	}

	// If no date found in filename, use file modification time
	if emailDate.IsZero() {
		if info, err := os.Stat(filePath); err == nil {
			emailDate = info.ModTime()
		} else {
			emailDate = time.Now()
		}
	}

	// Create a basic email structure
	emailID := fmt.Sprintf("email_%d", time.Now().UnixNano())
	email := &Email{
		ID:         emailID,
		Subject:    nameWithoutExt, // Use filename as subject for now
		From:       "unknown@example.com",
		To:         []string{"user@example.com"},
		Body:       content,
		Date:       emailDate,
		ReceivedAt: time.Now(),
		MessageID:  fmt.Sprintf("<%s@assistant>", emailID),
		FilePath:   filePath,
		IsRead:     false,
	}

	// Try to extract more information from content if it's a structured email
	if strings.Contains(content, "Subject:") {
		// Basic email header parsing
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Subject:") {
				email.Subject = strings.TrimSpace(strings.TrimPrefix(line, "Subject:"))
			} else if strings.HasPrefix(line, "From:") {
				email.From = strings.TrimSpace(strings.TrimPrefix(line, "From:"))
			} else if strings.HasPrefix(line, "To:") {
				email.To = []string{strings.TrimSpace(strings.TrimPrefix(line, "To:"))}
			}
		}
	}

	return email, nil
}

// filterAssistantHistory trims noisy rows and excludes the current message (it is
// appended again in buildAssistantPrompt) so small models are not confused by duplicates.
func filterAssistantHistory(history []*protocol.Message, current *protocol.Message) []*protocol.Message {
	if len(history) == 0 {
		return nil
	}
	currentID := ""
	if current != nil {
		currentID = current.ID
	}
	seen := make(map[string]struct{})
	out := make([]*protocol.Message, 0, len(history))
	for _, m := range history {
		if m == nil {
			continue
		}
		switch m.Type {
		case protocol.MessageTypeAgentJoin, protocol.MessageTypeAgentLeave, protocol.MessageTypeAgentStatus:
			continue
		}
		if currentID != "" && m.ID == currentID {
			continue
		}
		if m.ID != "" {
			if _, ok := seen[m.ID]; ok {
				continue
			}
			seen[m.ID] = struct{}{}
		}
		out = append(out, m)
	}
	const maxAssistantHistory = 8
	if len(out) > maxAssistantHistory {
		out = out[len(out)-maxAssistantHistory:]
	}
	return out
}

func userAsksAboutPromptContext(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}
	markers := []string{
		"what information", "what infomation", "what context", "what data",
		"exact information", "metadata", "in your context", "current context",
		"when i send you a prompt", "what you get when",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
