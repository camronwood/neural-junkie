package hub

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// CommandHandler handles chat commands
type CommandHandler struct {
	hub               *Hub
	aiProvider        ai.AIProvider
	repoAgents        map[string]*agent.RepoAgent        // Track repo agents for management
	helperAgents      map[string]*agent.HelperAgent      // Track helper agents for management
	confluenceAgents  map[string]*agent.ConfluenceAgent  // Track confluence agents for management
	assistantAgent    *agent.AssistantAgent              // Track assistant agent for meeting notes
	exportStorage     *mcp_export.ExportStorage          // Export storage for MCP exports
	pendingReviews    map[string]*protocol.PendingReview // Track pending reviews by repo path
	pendingMutex      sync.Mutex                         // Protects pending reviews map
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(hub *Hub) (*CommandHandler, error) {
	// Create AI provider for repo agents
	var aiProvider ai.AIProvider
	ollamaProvider, err := ai.NewOllamaProvider()
	if err != nil {
		// Log the error and fall back to mock provider
		fmt.Printf("⚠️  Warning: Failed to initialize Ollama provider: %v\n", err)
		fmt.Printf("⚠️  Using mock AI provider for repo agents. Make sure Ollama is running on localhost:11434\n")
		aiProvider = ai.NewMockProvider()
	} else {
		aiProvider = ollamaProvider
		fmt.Printf("✅ Ollama provider initialized for repo agents (model: %s)\n", ollamaProvider.GetModel())
	}

	// Initialize export storage
	exportStorage, err := mcp_export.NewExportStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to create export storage: %w", err)
	}

	return &CommandHandler{
		hub:               hub,
		aiProvider:        aiProvider,
		repoAgents:        make(map[string]*agent.RepoAgent),
		helperAgents:      make(map[string]*agent.HelperAgent),
		confluenceAgents:  make(map[string]*agent.ConfluenceAgent),
		exportStorage:     exportStorage,
		pendingReviews:    make(map[string]*protocol.PendingReview),
	}, nil
}

// ProcessCommand processes a command from a message
func (ch *CommandHandler) ProcessCommand(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	content := strings.TrimSpace(msg.Content)

	// Debug logging
	fmt.Printf("DEBUG: Processing command: %s\n", content)

	// Check if it's a command (starts with /)
	if !strings.HasPrefix(content, "/") {
		return nil, nil // Not a command
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return nil, nil
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/create-repo-agent":
		return ch.handleCreateRepoAgent(ctx, msg, parts)
	case "/create-confluence-agent":
		return ch.handleCreateConfluenceAgent(ctx, msg, parts)
	case "/create-helper":
		return ch.handleCreateHelper(ctx, msg, parts)
	case "/list-helper-templates":
		return ch.handleListHelperTemplates(ctx, msg)
	case "/delete-agent":
		return ch.handleDeleteAgent(ctx, msg, parts)
	case "/reindex-agent":
		return ch.handleReindexAgent(ctx, msg, parts)
	case "/reindex-confluence-agent":
		return ch.handleReindexConfluenceAgent(ctx, msg, parts)
	case "/pause-agent":
		return ch.handlePauseAgent(ctx, msg, parts)
	case "/unpause-agent":
		return ch.handleUnpauseAgent(ctx, msg, parts)
	case "/enable-watch":
		return ch.handleEnableWatch(ctx, msg, parts)
	case "/disable-watch":
		return ch.handleDisableWatch(ctx, msg, parts)
	case "/list-agents":
		return ch.handleListAgents(ctx, msg)
	case "/list-confluence-agents":
		return ch.handleListConfluenceAgents(ctx, msg)
	case "/remove-agent":
		return ch.handleRemoveAgent(ctx, msg, parts)
	case "/recall-agent":
		return ch.handleRecallAgent(ctx, msg, parts)
	case "/list-removed-agents":
		return ch.handleListRemovedAgents(ctx, msg)
	case "/export-agent-mcp":
		return ch.handleExportAgentMCP(ctx, msg, parts)
	case "/list-exports":
		return ch.handleListExports(ctx, msg)
	case "/delete-export":
		return ch.handleDeleteExport(ctx, msg, parts)
	case "/import-agent-mcp":
		return ch.handleImportAgentMCP(ctx, msg, parts)
	case "/export-all-agents":
		return ch.handleExportAllAgents(ctx, msg)
	case "/test-anthropic-connection":
		return ch.handleTestAnthropicConnection(ctx, msg)
	case "/test-github-connection":
		return ch.handleTestGitHubConnection(ctx, msg)
	case "/test-confluence-connection":
		return ch.handleTestConfluenceConnection(ctx, msg)
	case "/switch-provider":
		return ch.handleSwitchProvider(ctx, msg, parts)
	case "/switch-all-providers":
		return ch.handleSwitchAllProviders(ctx, msg, parts)
	case "/help":
		return ch.handleHelp(ctx, msg)
	case "/migrate-agent-names":
		return ch.handleMigrateAgentNames(ctx, msg)
	case "/open-file":
		return ch.handleOpenFile(ctx, msg, parts)
	case "/add-workspace":
		return ch.handleAddWorkspace(ctx, msg, parts)
	case "/list-workspaces":
		return ch.handleListWorkspaces(ctx, msg)
	case "/remind", "/remind-recurring":
		return ch.handleReminder(ctx, msg, parts)
	case "/task-add", "/task-list", "/task-done":
		return ch.handleTask(ctx, msg, parts)
	case "/note-save", "/note-search":
		return ch.handleNote(ctx, msg, parts)
	case "/meeting-add":
		return ch.handleMeeting(ctx, msg, parts)
	case "/ingest-meetings":
		return ch.handleIngestMeetings(ctx, msg)
	case "/search-meetings":
		return ch.handleSearchMeetings(ctx, msg, parts)
	case "/meeting-summary":
		return ch.handleMeetingSummary(ctx, msg, parts)
	case "/action-items":
		return ch.handleActionItems(ctx, msg)
	case "/list-meetings":
		return ch.handleListMeetings(ctx, msg)
	case "/summarize":
		return ch.handleSummarize(ctx, msg, parts)
	case "/help-assistant":
		return ch.handleAssistantHelp(ctx, msg)
	case "/analyze-design":
		return ch.handleAnalyzeDesign(ctx, msg, parts)
	case "/approve-file":
		return ch.handleApproveFile(ctx, msg, parts)
	case "/reject-file":
		return ch.handleRejectFile(ctx, msg, parts)
	case "/approve-delete":
		return ch.handleApproveDelete(ctx, msg, parts)
	case "/list-file-changes":
		return ch.handleListFileChanges(ctx, msg)
	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unknown command: %s\nUse /help to see available commands.", command)), nil
	}
}

// handleCreateRepoAgent creates a new repository expert agent
func (ch *CommandHandler) handleCreateRepoAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /create-repo-agent <repo-path> [agent-name] [provider] [model]\nProviders: ollama (default), claude, lmstudio\nExample: /create-repo-agent /path/to/repo MyRepoExpert ollama llama3.1"), nil
	}

	repoPath := parts[1]
	agentName := ""
	provider := "ollama"  // Default to ollama
	model := ""

	// Parse arguments
	if len(parts) >= 3 {
		// Check if third argument is a provider
		if parts[2] == "claude" || parts[2] == "ollama" || parts[2] == "lmstudio" {
			provider = parts[2]
			if len(parts) >= 4 {
				model = parts[3]
			}
		} else {
			// Third argument is agent name
			agentName = protocol.NormalizeAgentName(strings.Join(parts[2:], " "))
		}
	}

	if agentName == "" {
		// Generate name from repo path
		agentName = protocol.NormalizeAgentName(filepath.Base(repoPath) + "Expert")
	}

	// Validate path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid repository path: %v", err)), nil
	}

	// Check if agent with same name already exists
	existingAgents := ch.hub.ListAgents()
	for _, existingAgent := range existingAgents {
		if strings.EqualFold(existingAgent.Name, agentName) && existingAgent.Type == protocol.AgentTypeRepo {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Repository agent '%s' already exists. Use /delete-agent to remove it first.", agentName)), nil
		}
	}

	// Create AI provider based on selection
	var aiProvider ai.AIProvider

	if provider == "ollama" {
		if model == "" {
			model = "llama3.1"
		}
		aiProvider = ai.NewOllamaProviderWithConfig("http://localhost:11434", model)
	} else if provider == "lmstudio" {
		if model == "" {
			model = "" // Will be determined from available models
		}
		// Get endpoint from metadata or use default
		endpoint := "http://localhost:1234/v1"
		if ep, ok := msg.Metadata["lm_studio_endpoint"].(string); ok && ep != "" {
			endpoint = ep
		}
		aiProvider = ai.NewLMStudioProviderWithConfig(endpoint, model)
	} else {
		// Claude provider
		if model == "" {
			model = "claude-sonnet"
		}

		// Check for custom Anthropic credentials in metadata
		if apiKey, ok := msg.Metadata["anthropic_api_key"].(string); ok && apiKey != "" {
			useAIHub, _ := msg.Metadata["use_ai_hub"].(bool)
			aiHubEndpoint, _ := msg.Metadata["ai_hub_endpoint"].(string)
			aiProvider = ai.NewClaudeProviderWithConfig(apiKey, useAIHub, aiHubEndpoint, model)
		} else {
			aiProvider = ai.NewClaudeProviderWithConfig("", false, "", model)
		}
	}

	// Create repo agent
	repoAgent, err := agent.NewRepoAgent(agentName, absPath, aiProvider, ch.hub)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create repository agent: %v", err)), nil
	}

	// Register with hub
	if err := ch.hub.RegisterAgent(&repoAgent.Info); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
	}

	// Join channel
	if err := ch.hub.JoinChannel(repoAgent.Info.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
	}

	// Start with indexing
	if err := repoAgent.StartWithIndexing(ctx, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
	}

	// Track the repo agent
	ch.repoAgents[repoAgent.Info.ID] = repoAgent

	// Auto-create workspace for the repository
	if ch.hub.GetWorkspaceManager() != nil {
		workspaceName := agentName // Use agent name as workspace name
		_, err := ch.hub.GetWorkspaceManager().AddWorkspace(workspaceName, absPath)
		if err != nil {
			// Log error but don't fail agent creation
			fmt.Printf("Warning: Failed to auto-create workspace for repo agent: %v\n", err)
		} else {
			fmt.Printf("DEBUG: Auto-created workspace '%s' for repo agent at %s\n", workspaceName, absPath)
		}
	}

	// Check if this was auto-created for a pending review
	isAutoCreated := false
	if autoCreated, ok := msg.Metadata["auto_created"].(bool); ok && autoCreated {
		isAutoCreated = true
	}

	// Check if cache exists for this repository
	storage, err := repo.NewStorage()
	cacheExists := false
	if err == nil {
		cacheKey, keyErr := storage.GetCacheKeyForPath(absPath)
		if keyErr == nil {
			cacheExists = storage.IndexExists(cacheKey)
		}
	}

	var statusMsg string
	if isAutoCreated {
		// For auto-created agents, use a more concise message
		if cacheExists {
			statusMsg = fmt.Sprintf("✅ Repository expert agent '%s' created and ready!\n"+
				"💾 Loaded from cache (instant) - repository already indexed.",
				agentName)
		} else {
			statusMsg = fmt.Sprintf("✅ Repository expert agent '%s' created!\n"+
				"📊 Indexing repository (30-60 seconds) - agent will respond when ready.",
				agentName)
		}
	} else {
		// For manual creation, use the original detailed messages
		if cacheExists {
			statusMsg = fmt.Sprintf("🤖 Creating repository expert agent '%s' for %s...\n"+
				"💾 Cache found! Loading will be instant if cache is fresh.\n"+
				"Watch for status messages from the agent.",
				agentName, filepath.Base(absPath))
		} else {
			statusMsg = fmt.Sprintf("🤖 Creating repository expert agent '%s' for %s...\n"+
				"📊 No cache found - first indexing may take 30-60 seconds.\n"+
				"Future agents for this repository will load instantly from cache!",
				agentName, filepath.Base(absPath))
		}
	}

	return ch.systemResponse(msg.Channel, statusMsg), nil
}

// handleDeleteAgent deletes an agent
func (ch *CommandHandler) handleDeleteAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /delete-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name
	var agentID string
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			agentID = a.ID
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
	}

	// If it's a repo agent, clean up stored data
	if repoAgent, ok := ch.repoAgents[agentID]; ok {
		if err := repoAgent.Cleanup(); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("⚠️  Warning: Failed to cleanup agent data: %v", err)), nil
		}
		delete(ch.repoAgents, agentID)
	}

	// If it's a helper agent, stop it
	if helperAgent, ok := ch.helperAgents[agentID]; ok {
		helperAgent.Stop()
		delete(ch.helperAgents, agentID)
	}

	// If it's a confluence agent, clean up by name
	for name, confluenceAgent := range ch.confluenceAgents {
		if confluenceAgent.Info.ID == agentID {
			confluenceAgent.Stop()
			delete(ch.confluenceAgents, name)
			break
		}
	}

	// Leave channel
	ch.hub.LeaveChannel(agentID, msg.Channel)

	// Unregister agent
	if err := ch.hub.UnregisterAgent(agentID); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to delete agent: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Agent '%s' has been deleted", agentName)), nil
}

// handleReindexAgent triggers a reindex of a repository agent
func (ch *CommandHandler) handleReindexAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /reindex-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name
	var agentID string
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) && a.Type == protocol.AgentTypeRepo {
			agentID = a.ID
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Repository agent '%s' not found", agentName)), nil
	}

	// Get repo agent
	repoAgent, ok := ch.repoAgents[agentID]
	if !ok {
		return ch.systemResponse(msg.Channel, "❌ Agent is not a repository agent"), nil
	}

	// Trigger reindex
	if err := repoAgent.Reindex(ctx); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start reindex: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🔄 Reindexing repository for '%s'...\n"+
			"The agent will be temporarily unavailable during reindexing.",
			agentName)), nil
}

// handlePauseAgent pauses an agent
func (ch *CommandHandler) handlePauseAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /pause-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find and pause agent
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			// Update agent status in hub
			ch.hub.mu.Lock()
			if agent, ok := ch.hub.agents[a.ID]; ok {
				agent.IsPaused = true
				agent.Status = "paused"
			}
			ch.hub.mu.Unlock()

			// If it's a repo agent we manage, pause it
			if repoAgent, ok := ch.repoAgents[a.ID]; ok {
				repoAgent.Pause()
			}

			// If it's a helper agent we manage, pause it
			if helperAgent, ok := ch.helperAgents[a.ID]; ok {
				helperAgent.Pause()
			}

			// If it's a confluence agent we manage, pause it
			for _, confluenceAgent := range ch.confluenceAgents {
				if confluenceAgent.Info.ID == a.ID {
					confluenceAgent.Pause()
					break
				}
			}

			return ch.systemResponse(msg.Channel, fmt.Sprintf("⏸️  Agent '%s' has been paused", agentName)), nil
		}
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
}

// handleUnpauseAgent unpauses an agent
func (ch *CommandHandler) handleUnpauseAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /unpause-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find and unpause agent
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			// Update agent status in hub
			ch.hub.mu.Lock()
			if agent, ok := ch.hub.agents[a.ID]; ok {
				agent.IsPaused = false
				agent.Status = "active"
			}
			ch.hub.mu.Unlock()

			// If it's a repo agent we manage, unpause it
			if repoAgent, ok := ch.repoAgents[a.ID]; ok {
				repoAgent.Unpause()
			}

			// If it's a helper agent we manage, unpause it
			if helperAgent, ok := ch.helperAgents[a.ID]; ok {
				helperAgent.Unpause()
			}

			// If it's a confluence agent we manage, unpause it
			for _, confluenceAgent := range ch.confluenceAgents {
				if confluenceAgent.Info.ID == a.ID {
					confluenceAgent.Unpause()
					break
				}
			}

			return ch.systemResponse(msg.Channel, fmt.Sprintf("▶️  Agent '%s' has been unpaused", agentName)), nil
		}
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
}

// handleListAgents lists all agents in the channel
func (ch *CommandHandler) handleListAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// Get agents in current channel
	channelAgents, err := ch.hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Failed to list agents"), nil
	}

	// Get all registered agents to show user-created agents that might not be in channel
	allAgents := ch.hub.ListAgents()

	var response strings.Builder
	response.WriteString("🤖 **My Agents:**\n\n")

	// Show agents in current channel first
	if len(channelAgents) > 0 {
		response.WriteString("**In this channel:**\n")
		for _, a := range channelAgents {
			status := "✅"
			if a.IsPaused {
				status = "⏸️"
			} else if a.IndexingStatus == string(protocol.IndexingStatusIndexing) {
				status = "🔄"
			} else if a.IndexingStatus == string(protocol.IndexingStatusReindexing) {
				status = "🔄"
			}

			response.WriteString(fmt.Sprintf("%s **%s** (%s)", status, a.Name, a.Type))

			if a.Type == protocol.AgentTypeRepo {
				if a.IndexingStatus == string(protocol.IndexingStatusIndexing) ||
					a.IndexingStatus == string(protocol.IndexingStatusReindexing) {
					response.WriteString(fmt.Sprintf(" - Indexing: %d%%", a.IndexProgress))
				}
				if a.RepositoryPath != "" {
					response.WriteString(fmt.Sprintf("\n  📁 %s", filepath.Base(a.RepositoryPath)))
				}
			}

			if a.Type == protocol.AgentTypeHelper {
				if a.KnowledgePath != "" {
					response.WriteString(fmt.Sprintf("\n  📚 %s", filepath.Base(a.KnowledgePath)))
				}
			}

			response.WriteString("\n")
		}
		response.WriteString("\n")
	}

	// Show user-created agents that are not in any channel
	userCreatedAgents := []protocol.AgentInfo{}
	for _, agent := range allAgents {
		if protocol.IsUserCreatedAgent(string(agent.Type)) {
			// Check if agent is in current channel
			inCurrentChannel := false
			for _, channelAgent := range channelAgents {
				if channelAgent.ID == agent.ID {
					inCurrentChannel = true
					break
				}
			}

			// If not in current channel, check if it's in any channel
			if !inCurrentChannel && !ch.hub.IsAgentInAnyChannel(agent.ID) {
				userCreatedAgents = append(userCreatedAgents, *agent)
			}
		}
	}

	if len(userCreatedAgents) > 0 {
		response.WriteString("**Available (not in any channel):**\n")
		for _, a := range userCreatedAgents {
			status := "📋"
			if a.IsPaused {
				status = "⏸️"
			} else if a.IndexingStatus == string(protocol.IndexingStatusIndexing) {
				status = "🔄"
			} else if a.IndexingStatus == string(protocol.IndexingStatusReindexing) {
				status = "🔄"
			}

			response.WriteString(fmt.Sprintf("%s **%s** (%s)", status, a.Name, a.Type))

			if a.Type == protocol.AgentTypeRepo {
				if a.IndexingStatus == string(protocol.IndexingStatusIndexing) ||
					a.IndexingStatus == string(protocol.IndexingStatusReindexing) {
					response.WriteString(fmt.Sprintf(" - Indexing: %d%%", a.IndexProgress))
				}
				if a.RepositoryPath != "" {
					response.WriteString(fmt.Sprintf("\n  📁 %s", filepath.Base(a.RepositoryPath)))
				}
			}

			if a.Type == protocol.AgentTypeHelper {
				if a.KnowledgePath != "" {
					response.WriteString(fmt.Sprintf("\n  📚 %s", filepath.Base(a.KnowledgePath)))
				}
			}

			response.WriteString("\n")
		}
		response.WriteString("\n")
	}

	// Show removed agents
	removedAgents := ch.hub.GetRemovedAgents()
	if len(removedAgents) > 0 {
		response.WriteString("**Removed agents:**\n")
		for _, a := range removedAgents {
			response.WriteString(fmt.Sprintf("🚪 **%s** (%s)\n", a.Name, a.Type))
		}
		response.WriteString("\n")
	}

	if len(channelAgents) == 0 && len(userCreatedAgents) == 0 && len(removedAgents) == 0 {
		response.WriteString("No agents available.\n\n")
		response.WriteString("**Create agents:**\n")
		response.WriteString("• `/create-repo-agent <path>` - Repository expert\n")
		response.WriteString("• `/create-helper <template>` - Helper agent\n")
		response.WriteString("• `/create-confluence-agent <space>` - Confluence expert\n")
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleEnableWatch enables automatic file watching for a repo agent
func (ch *CommandHandler) handleEnableWatch(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /enable-watch <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name
	var agentID string
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) && a.Type == protocol.AgentTypeRepo {
			agentID = a.ID
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Repository agent '%s' not found", agentName)), nil
	}

	// Get repo agent
	repoAgent, ok := ch.repoAgents[agentID]
	if !ok {
		return ch.systemResponse(msg.Channel, "❌ Agent is not a repository agent"), nil
	}

	// Enable auto-watch
	repoAgent.EnableAutoWatch(ctx)

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("👁️  Auto-watch enabled for '%s'\n"+
			"The agent will now automatically detect file changes and reindex.",
			agentName)), nil
}

// handleDisableWatch disables automatic file watching for a repo agent
func (ch *CommandHandler) handleDisableWatch(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /disable-watch <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name
	var agentID string
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) && a.Type == protocol.AgentTypeRepo {
			agentID = a.ID
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Repository agent '%s' not found", agentName)), nil
	}

	// Get repo agent
	repoAgent, ok := ch.repoAgents[agentID]
	if !ok {
		return ch.systemResponse(msg.Channel, "❌ Agent is not a repository agent"), nil
	}

	// Disable auto-watch
	repoAgent.DisableAutoWatch()

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🚫 Auto-watch disabled for '%s'", agentName)), nil
}

// handleCreateHelper creates a new helper agent
func (ch *CommandHandler) handleCreateHelper(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"Usage: /create-helper <template-name>\n\n"+
				"**Available templates:**\n"+
				"• `day-one` or `dayoneexpert` - Day One Expert (onboarding help)\n"+
				"• `testing-expert` or `testingexpert` - Testing Expert (testing practices)\n"+
				"• `docs-expert` or `docsexpert` - Docs Expert (documentation help)\n\n"+
				"Use `/list-helper-templates` for detailed information."), nil
	}

	templateName := parts[1]

	// Get template configuration
	templates := agent.DefaultHelperAgentConfigs()

	// Try exact match first
	config, ok := templates[templateName]

	// If not found, try case-insensitive match and common variations
	if !ok {
		// Normalize the input (lowercase, replace underscores with hyphens)
		normalized := strings.ToLower(strings.ReplaceAll(templateName, "_", "-"))

		// Try normalized match
		config, ok = templates[normalized]

		// If still not found, try common variations
		if !ok {
			variations := map[string]string{
				"dayoneexpert":   "day-one",
				"dayone":         "day-one",
				"day-one-expert": "day-one",
				"testingexpert":  "testing-expert",
				"testing":        "testing-expert",
				"docsexpert":     "docs-expert",
				"docs":           "docs-expert",
				"docs-expert":    "docs-expert",
			}

			if variation, exists := variations[normalized]; exists {
				config, ok = templates[variation]
			}
		}
	}

	if !ok {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Unknown template '%s'\n"+
				"Use `/list-helper-templates` to see available templates.\n\n"+
				"**Common variations:**\n"+
				"• `day-one` or `dayoneexpert` for Day One Expert\n"+
				"• `testing-expert` or `testingexpert` for Testing Expert\n"+
				"• `docs-expert` or `docsexpert` for Docs Expert", templateName)), nil
	}

	// Initialize storage and save template if needed
	storage, err := agent.NewHelperAgentStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to initialize storage: %v", err)), nil
	}

	// Check if config already exists
	_, err = storage.LoadConfig(templateName)
	if err != nil {
		// Save the template config
		if err := storage.EnsureKnowledgePath(templateName); err != nil {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Failed to create knowledge directory: %v", err)), nil
		}

		config.KnowledgePath = storage.GetKnowledgePath(templateName)
		if err := storage.SaveConfig(templateName, config); err != nil {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Failed to save configuration: %v", err)), nil
		}

		// Create example knowledge
		if err := storage.CreateDefaultTemplates(); err != nil {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("⚠️  Warning: Failed to create example knowledge: %v", err)), nil
		}
	}

	// Normalize the agent name for @mention compatibility
	config.Name = protocol.NormalizeAgentName(config.Name)

	// Create helper agent
	helperAgent, err := agent.NewHelperAgent(config, ch.aiProvider, ch.hub)
	if err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to create helper agent: %v", err)), nil
	}

	// Check if agent with same name already exists
	existingAgents := ch.hub.ListAgents()
	for _, existingAgent := range existingAgents {
		if strings.EqualFold(existingAgent.Name, config.Name) && existingAgent.Type == protocol.AgentTypeHelper {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Helper agent '%s' already exists. Use /delete-agent to remove it first.", config.Name)), nil
		}
	}

	// Register with hub
	if err := ch.hub.RegisterAgent(&helperAgent.Info); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
	}

	// Join channel
	if err := ch.hub.JoinChannel(helperAgent.Info.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
	}

	// Start agent
	if err := helperAgent.Start(ctx, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
	}

	// Track the helper agent
	ch.helperAgents[helperAgent.Info.ID] = helperAgent

	statusMsg := fmt.Sprintf("🤖 Created helper agent: **%s**\n\n"+
		"**Description:** %s\n\n"+
		"**Expertise:** %s\n\n"+
		"**Knowledge Base:** %s\n\n"+
		"You can customize the knowledge base by adding .md or .txt files to:\n"+
		"`%s`",
		config.Name,
		config.Description,
		strings.Join(config.Expertise, ", "),
		filepath.Base(config.KnowledgePath),
		config.KnowledgePath)

	return ch.systemResponse(msg.Channel, statusMsg), nil
}

// handleListHelperTemplates lists available helper agent templates
func (ch *CommandHandler) handleListHelperTemplates(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	templates := agent.DefaultHelperAgentConfigs()

	var response strings.Builder
	response.WriteString("📋 **Available Helper Agent Templates:**\n\n")

	// Sort templates for consistent output
	templateNames := []string{"day-one", "testing-expert", "docs-expert"}

	for _, name := range templateNames {
		if config, ok := templates[name]; ok {
			response.WriteString(fmt.Sprintf("**`%s`** - %s\n", name, config.Name))
			response.WriteString(fmt.Sprintf("  %s\n", config.Description))
			response.WriteString(fmt.Sprintf("  Keywords: %s\n\n", strings.Join(config.Keywords[:3], ", ")))
		}
	}

	response.WriteString("\n**Usage:**\n")
	response.WriteString("```\n/create-helper <template-name>\n```\n\n")
	response.WriteString("**Example:**\n")
	response.WriteString("```\n/create-helper day-one\n```\n")

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleHelp shows available commands
func (ch *CommandHandler) handleHelp(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	help := "**Available Commands:**\n\n" +
		"**Repository Agents:**\n" +
		"• `/create-repo-agent <path> [name]` - Create a repository expert agent\n" +
		"• `/reindex-agent <name>` - Reindex a repository agent\n" +
		"• `/enable-watch <name>` - Enable automatic file watching and reindexing\n" +
		"• `/disable-watch <name>` - Disable automatic file watching\n\n" +
		"**Helper Agents:**\n" +
		"• `/create-helper <template>` - Create a helper agent (e.g., day-one, testing-expert)\n" +
		"• `/list-helper-templates` - Show available helper agent templates\n\n" +
		"**Agent Management:**\n" +
		"• `/remove-agent <name>` - Remove agent from conversation (can recall later)\n" +
		"• `/recall-agent <name>` - Recall a removed agent back to conversation\n" +
		"• `/list-removed-agents` - List removed agents available for recall\n" +
		"• `/delete-agent <name>` - Delete an agent permanently\n" +
		"• `/pause-agent <name>` - Pause an agent (stops responding)\n" +
		"• `/unpause-agent <name>` - Resume a paused agent\n" +
		"• `/list-agents` - List all agents in the channel\n\n" +
		"**MCP Exports:**\n" +
		"• `/export-agent-mcp <name>` - Export an agent to MCP format\n" +
		"• `/list-exports` - List all exported agents\n" +
		"• `/delete-export <name>` - Delete an export\n" +
		"• `/import-agent-mcp <path>` - Import an agent from file\n" +
		"• `/export-all-agents` - Export all agents at once\n\n" +
		"**Migration:**\n" +
		"• `/migrate-agent-names` - Check and migrate agent names for @mention compatibility\n\n" +
		"**Help:**\n" +
		"• `/help` - Show this help message\n\n" +
		"**Examples:**\n" +
		"```\n" +
		"/create-repo-agent /path/to/my-project MyProjectExpert\n" +
		"/create-helper day-one\n" +
		"/enable-watch MyProjectExpert\n" +
		"```\n"

	return ch.systemResponse(msg.Channel, help), nil
}

// systemResponse creates a system message response
func (ch *CommandHandler) systemResponse(channel, content string) *protocol.Message {
	return protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{
			ID:   "system",
			Name: "System",
			Type: protocol.AgentTypeGeneral,
		},
		content,
	)
}

// handleCreateConfluenceAgent creates a new Confluence space expert agent
func (ch *CommandHandler) handleCreateConfluenceAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /create-confluence-agent <space-key> [agent-name]"), nil
	}

	spaceKey := parts[1]
	agentName := ""
	if len(parts) >= 3 {
		agentName = protocol.NormalizeAgentName(strings.Join(parts[2:], " "))
	} else {
		// Generate name from space key
		agentName = protocol.NormalizeAgentName(spaceKey + "Expert")
	}

	// Check for custom credentials in metadata
	var aiProvider ai.AIProvider = ch.aiProvider
	var confluenceCredentials map[string]string

	// Check if custom Anthropic credentials are provided
	if apiKey, ok := msg.Metadata["anthropic_api_key"].(string); ok && apiKey != "" {
		useAIHub, _ := msg.Metadata["use_ai_hub"].(bool)
		aiHubEndpoint, _ := msg.Metadata["ai_hub_endpoint"].(string)

		// Create custom AI provider with provided credentials
		customProvider := ai.NewClaudeProviderWithConfig(apiKey, useAIHub, aiHubEndpoint, "")
		aiProvider = customProvider
	}

	// Check if custom Confluence credentials are provided
	if credentials, ok := msg.Metadata["confluence_credentials"].(map[string]interface{}); ok {
		confluenceCredentials = make(map[string]string)
		if domain, ok := credentials["domain"].(string); ok {
			confluenceCredentials["domain"] = domain
		}
		if email, ok := credentials["email"].(string); ok {
			confluenceCredentials["email"] = email
		}
		if apiToken, ok := credentials["api_token"].(string); ok {
			confluenceCredentials["api_token"] = apiToken
		}
	}

	// Create Confluence agent
	confluenceAgent, err := agent.NewConfluenceAgent(agentName, spaceKey, aiProvider, ch.hub)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create Confluence agent: %v", err)), nil
	}

	// Set custom credentials if provided
	if len(confluenceCredentials) > 0 {
		confluenceAgent.SetCredentials(confluenceCredentials)
	}

	// Register with hub
	if err := ch.hub.RegisterAgent(&confluenceAgent.Info); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
	}

	// Join channel
	if err := ch.hub.JoinChannel(confluenceAgent.Info.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
	}

	// Store agent reference
	ch.confluenceAgents[agentName] = confluenceAgent

	// Start indexing
	if err := confluenceAgent.StartWithIndexing(ctx, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("✅ Confluence agent '%s' created for space '%s'.\nIndexing in progress...", agentName, spaceKey)), nil
}

// handleReindexConfluenceAgent triggers a manual reindex of a Confluence space
func (ch *CommandHandler) handleReindexConfluenceAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /reindex-confluence-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")
	confluenceAgent, exists := ch.confluenceAgents[agentName]
	if !exists {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Confluence agent '%s' not found", agentName)), nil
	}

	if err := confluenceAgent.Reindex(ctx); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to reindex: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Reindexing '%s' space...", agentName)), nil
}

// handleListConfluenceAgents lists all Confluence agents
func (ch *CommandHandler) handleListConfluenceAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	if len(ch.confluenceAgents) == 0 {
		return ch.systemResponse(msg.Channel, "No Confluence agents currently active."), nil
	}

	var agentList strings.Builder
	agentList.WriteString("**Confluence Agents:**\n\n")

	for name, agent := range ch.confluenceAgents {
		status := "✅ Ready"
		if agent.Info.IsPaused {
			status = "⏸️  Paused"
		} else if agent.Info.IndexingStatus == string(protocol.IndexingStatusIndexing) {
			status = fmt.Sprintf("🔄 Indexing (%d%%)", agent.Info.IndexProgress)
		} else if agent.Info.IndexingStatus == string(protocol.IndexingStatusReindexing) {
			status = fmt.Sprintf("🔄 Reindexing (%d%%)", agent.Info.IndexProgress)
		}

		index := agent.GetIndex()
		var stats string
		if index != nil {
			stats = fmt.Sprintf("%d pages", index.PageCount)
		} else {
			stats = "Not indexed yet"
		}

		agentList.WriteString(fmt.Sprintf("• **%s**\n", name))
		agentList.WriteString(fmt.Sprintf("  Space: %s\n", agent.Info.ConfluenceSpaceKey))
		agentList.WriteString(fmt.Sprintf("  Status: %s\n", status))
		agentList.WriteString(fmt.Sprintf("  Stats: %s\n", stats))
		agentList.WriteString("\n")
	}

	agentList.WriteString("\n**Commands:**\n")
	agentList.WriteString("• `/reindex-confluence-agent <name>` - Reindex a space\n")
	agentList.WriteString("• `/pause-agent <name>` - Pause an agent\n")
	agentList.WriteString("• `/unpause-agent <name>` - Unpause an agent\n")
	agentList.WriteString("• `/delete-agent <name>` - Delete an agent\n")

	return ch.systemResponse(msg.Channel, agentList.String()), nil
}

// handleRemoveAgent removes an agent from the conversation (temporary removal)
func (ch *CommandHandler) handleRemoveAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /remove-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name in current channel
	var agentID string
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			agentID = a.ID
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
	}

	// Check if agent is in the current channel
	channelAgents, err := ch.hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Failed to get channel agents"), nil
	}

	agentInChannel := false
	for _, a := range channelAgents {
		if a.ID == agentID {
			agentInChannel = true
			break
		}
	}

	if !agentInChannel {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' is not in this channel", agentName)), nil
	}

	// Leave channel
	if err := ch.hub.LeaveChannel(agentID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to remove agent: %v", err)), nil
	}

	// Check if agent is now in any other channels
	if !ch.hub.IsAgentInAnyChannel(agentID) {
		// Get agent info to check type
		agent, err := ch.hub.GetAgent(agentID)
		if err == nil {
			// Check if this is a user-created agent (repo, helper, confluence)
			if protocol.IsUserCreatedAgent(string(agent.Type)) {
				// User-created agents don't go to removed list, they stay available in "My Agents"
				// Just update the status to indicate they're not in any channel
				agent.Status = "available"
				agent.LastActiveTime = time.Now()
			} else {
				// System agents (frontend, backend, etc.) go to removed agents list
				agent.Status = "removed"
				agent.LastActiveTime = time.Now()
				agent.RemovedFrom = append(agent.RemovedFrom, msg.Channel)
				ch.hub.AddRemovedAgent(agent)
			}
		}
	}

	// Get agent info to provide appropriate message
	agent, err := ch.hub.GetAgent(agentID)
	if err == nil && protocol.IsUserCreatedAgent(string(agent.Type)) {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("🚪 Agent '%s' removed from conversation (available in My Agents)", agentName)), nil
	} else {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("🚪 Agent '%s' removed from conversation (use /recall-agent to bring back)", agentName)), nil
	}
}

// handleRecallAgent recalls a removed agent back to the conversation
func (ch *CommandHandler) handleRecallAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /recall-agent <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// First, try to find agent in removed agents
	removedAgents := ch.hub.GetRemovedAgents()
	var agentToRecall *protocol.AgentInfo
	for _, agent := range removedAgents {
		if strings.EqualFold(agent.Name, agentName) {
			agentToRecall = agent
			break
		}
	}

	// If not found in removed agents, check if it's a user-created agent that's just not in any channel
	if agentToRecall == nil {
		allAgents := ch.hub.ListAgents()
		for _, agent := range allAgents {
			if strings.EqualFold(agent.Name, agentName) && protocol.IsUserCreatedAgent(string(agent.Type)) {
				// Check if agent is not in any channel
				if !ch.hub.IsAgentInAnyChannel(agent.ID) {
					agentToRecall = agent
					break
				}
			}
		}
	}

	if agentToRecall == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found. Use /list-agents to see available agents.", agentName)), nil
	}

	// Join channel
	if err := ch.hub.JoinChannel(agentToRecall.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to recall agent: %v", err)), nil
	}

	// Remove from removed agents list if it was there
	ch.hub.RemoveFromRemovedAgents(agentToRecall.ID)

	// Update agent status
	agentToRecall.Status = "active"
	ch.hub.mu.Lock()
	if agent, ok := ch.hub.agents[agentToRecall.ID]; ok {
		agent.Status = "active"
		agent.LastActiveTime = time.Now()
	}
	ch.hub.mu.Unlock()

	return ch.systemResponse(msg.Channel, fmt.Sprintf("👋 Agent '%s' recalled to conversation", agentName)), nil
}

// handleListRemovedAgents lists all removed agents available for recall
func (ch *CommandHandler) handleListRemovedAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	removedAgents := ch.hub.GetRemovedAgents()

	if len(removedAgents) == 0 {
		return ch.systemResponse(msg.Channel, "No removed agents available for recall."), nil
	}

	var response strings.Builder
	response.WriteString("🚪 **Removed Agents Available for Recall:**\n\n")

	for _, agent := range removedAgents {
		response.WriteString(fmt.Sprintf("• **%s** (%s)\n", agent.Name, agent.Type))
		if len(agent.RemovedFrom) > 0 {
			response.WriteString(fmt.Sprintf("  Removed from: %s\n", strings.Join(agent.RemovedFrom, ", ")))
		}
		if !agent.LastActiveTime.IsZero() {
			response.WriteString(fmt.Sprintf("  Last active: %s\n", agent.LastActiveTime.Format("2006-01-02 15:04:05")))
		}
		response.WriteString("\n")
	}

	response.WriteString("**Usage:**\n")
	response.WriteString("```\n/recall-agent <agent-name>\n```\n")

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleExportAgentMCP exports an agent to MCP format
func (ch *CommandHandler) handleExportAgentMCP(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /export-agent-mcp <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Find agent by name first, then look up by ID
	var agentID string
	var agentType protocol.AgentType
	agents := ch.hub.ListAgents()
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			agentID = a.ID
			agentType = a.Type
			break
		}
	}

	if agentID == "" {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found. Use /list-agents to see available agents.", agentName)), nil
	}

	// Find agent in repo agents
	if agentType == protocol.AgentTypeRepo {
		if repoAgent, exists := ch.repoAgents[agentID]; exists {
			export, err := repoAgent.ExportToMCP()
			if err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to export repo agent: %v", err)), nil
			}

			if err := ch.exportStorage.SaveExport(export); err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save export: %v", err)), nil
			}

			metadata := repoAgent.GetExportMetadata()
			return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Exported repo agent '%s'\n📄 Resources: %d\n💬 Prompts: %d",
				agentName, metadata.ResourceCount, metadata.PromptCount)), nil
		}
	}

	// Find agent in helper agents
	if agentType == protocol.AgentTypeHelper {
		if helperAgent, exists := ch.helperAgents[agentID]; exists {
			export, err := helperAgent.ExportToMCP()
			if err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to export helper agent: %v", err)), nil
			}

			if err := ch.exportStorage.SaveExport(export); err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save export: %v", err)), nil
			}

			metadata := helperAgent.GetExportMetadata()
			return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Exported helper agent '%s'\n📄 Resources: %d\n💬 Prompts: %d",
				agentName, metadata.ResourceCount, metadata.PromptCount)), nil
		}
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found. Use /list-agents to see available agents.", agentName)), nil
}

// handleListExports lists all exported agents
func (ch *CommandHandler) handleListExports(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	exports, err := ch.exportStorage.ListExports()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to list exports: %v", err)), nil
	}

	if len(exports) == 0 {
		return ch.systemResponse(msg.Channel, "📦 No exports found. Use /export-agent-mcp to create exports."), nil
	}

	var response strings.Builder
	response.WriteString("📦 **Available Exports:**\n\n")

	for _, export := range exports {
		response.WriteString(fmt.Sprintf("**%s** (%s)\n", export.Name, export.Type))
		response.WriteString(fmt.Sprintf("  📄 Resources: %d | 💬 Prompts: %d | 📏 Size: %.1f KB\n",
			export.ResourceCount, export.PromptCount, float64(export.FileSize)/1024))
		if export.Description != "" {
			response.WriteString(fmt.Sprintf("  📝 %s\n", export.Description))
		}
		response.WriteString(fmt.Sprintf("  📁 %s\n\n", export.ExportPath))
	}

	stats, err := ch.exportStorage.GetExportStats()
	if err == nil {
		response.WriteString(fmt.Sprintf("📊 **Total:** %d exports (%d repo, %d helper) | %.1f KB total",
			stats.TotalExports, stats.RepoExports, stats.HelperExports, float64(stats.TotalSize)/1024))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleDeleteExport deletes an exported agent
func (ch *CommandHandler) handleDeleteExport(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /delete-export <agent-name>"), nil
	}

	agentName := strings.Join(parts[1:], " ")

	// Try to find the export first
	exports, err := ch.exportStorage.ListExports()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to list exports: %v", err)), nil
	}

	var foundExport *mcp_export.ExportInfo
	for _, export := range exports {
		if export.Name == agentName {
			foundExport = &export
			break
		}
	}

	if foundExport == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Export '%s' not found. Use /list-exports to see available exports.", agentName)), nil
	}

	// Delete the export
	if err := ch.exportStorage.DeleteExport(agentName, foundExport.Type); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to delete export: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Deleted export '%s' (%s)", agentName, foundExport.Type)), nil
}

// handleImportAgentMCP imports an agent from MCP export file
func (ch *CommandHandler) handleImportAgentMCP(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /import-agent-mcp <file-path>"), nil
	}

	filePath := parts[1]

	// Load export from file
	export, err := ch.exportStorage.LoadExportFromPath(filePath)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to load export: %v", err)), nil
	}

	// Check if agent already exists
	if _, exists := ch.repoAgents[export.Agent.Name]; exists {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Repo agent '%s' already exists", export.Agent.Name)), nil
	}
	if _, exists := ch.helperAgents[export.Agent.Name]; exists {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Helper agent '%s' already exists", export.Agent.Name)), nil
	}

	// Create agent based on type
	switch export.Agent.Type {
	case "repo":
		// For repo agents, we need the repository path
		if export.Agent.Repository == "" {
			return ch.systemResponse(msg.Channel, "❌ Repository path not found in export. Cannot recreate repo agent."), nil
		}

		// Create repo agent
		repoAgent, err := agent.NewRepoAgent(export.Agent.Name, export.Agent.Repository, ch.aiProvider, ch.hub)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create repo agent: %v", err)), nil
		}

		// Register with hub
		if err := ch.hub.RegisterAgent(&repoAgent.Info); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
		}

		// Join channel
		if err := ch.hub.JoinChannel(repoAgent.Info.ID, msg.Channel); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
		}

		// Start with indexing
		if err := repoAgent.StartWithIndexing(ctx, msg.Channel); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
		}

		// Track the repo agent
		ch.repoAgents[repoAgent.Info.ID] = repoAgent

		return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Imported repo agent '%s' from %s\n📄 Resources: %d | 💬 Prompts: %d",
			export.Agent.Name, filePath, export.GetResourceCount(), export.GetPromptCount())), nil

	case "helper":
		// For helper agents, we need the knowledge path
		if export.Agent.KnowledgePath == "" {
			return ch.systemResponse(msg.Channel, "❌ Knowledge path not found in export. Cannot recreate helper agent."), nil
		}

		// Create helper agent config
		config := &agent.HelperAgentConfig{
			Name:          export.Agent.Name,
			Description:   export.Agent.Description,
			Expertise:     export.Agent.Expertise,
			Keywords:      export.Agent.Keywords,
			KnowledgePath: export.Agent.KnowledgePath,
		}

		// Create helper agent
		helperAgent, err := agent.NewHelperAgent(config, ch.aiProvider, ch.hub)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create helper agent: %v", err)), nil
		}

		// Register with hub
		if err := ch.hub.RegisterAgent(&helperAgent.Info); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
		}

		// Join channel
		if err := ch.hub.JoinChannel(helperAgent.Info.ID, msg.Channel); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
		}

		// Start the agent
		if err := helperAgent.Start(ctx, msg.Channel); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
		}

		// Track the helper agent
		ch.helperAgents[helperAgent.Info.ID] = helperAgent

		return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Imported helper agent '%s' from %s\n📄 Resources: %d | 💬 Prompts: %d",
			export.Agent.Name, filePath, export.GetResourceCount(), export.GetPromptCount())), nil

	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unsupported agent type: %s", export.Agent.Type)), nil
	}
}

// handleExportAllAgents exports all available agents to MCP format
func (ch *CommandHandler) handleExportAllAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	var exported []string
	var errors []string

	// Export all repo agents
	for name, repoAgent := range ch.repoAgents {
		export, err := repoAgent.ExportToMCP()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Repo agent '%s': %v", name, err))
			continue
		}

		if err := ch.exportStorage.SaveExport(export); err != nil {
			errors = append(errors, fmt.Sprintf("Repo agent '%s': %v", name, err))
			continue
		}

		exported = append(exported, fmt.Sprintf("%s (repo)", name))
	}

	// Export all helper agents
	for name, helperAgent := range ch.helperAgents {
		export, err := helperAgent.ExportToMCP()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Helper agent '%s': %v", name, err))
			continue
		}

		if err := ch.exportStorage.SaveExport(export); err != nil {
			errors = append(errors, fmt.Sprintf("Helper agent '%s': %v", name, err))
			continue
		}

		exported = append(exported, fmt.Sprintf("%s (helper)", name))
	}

	// Build response
	var response strings.Builder
	if len(exported) > 0 {
		response.WriteString(fmt.Sprintf("✅ Exported %d agents:\n", len(exported)))
		for _, name := range exported {
			response.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	} else {
		response.WriteString("📦 No agents available to export. Create agents first with /create-repo-agent or /create-helper.\n")
	}

	if len(errors) > 0 {
		response.WriteString(fmt.Sprintf("\n❌ %d errors:\n", len(errors)))
		for _, err := range errors {
			response.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleTestAnthropicConnection tests Anthropic API connection
func (ch *CommandHandler) handleTestAnthropicConnection(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// Extract credentials from message metadata
	apiKey, ok := msg.Metadata["anthropic_api_key"].(string)
	if !ok || apiKey == "" {
		return ch.systemResponse(msg.Channel, "❌ No API key provided in request"), nil
	}

	useAIHub, _ := msg.Metadata["use_ai_hub"].(bool)
	aiHubEndpoint, _ := msg.Metadata["ai_hub_endpoint"].(string)

	// Create a test AI provider with the provided credentials
	var testProvider ai.AIProvider
	if useAIHub && aiHubEndpoint != "" {
		// Test AI Hub connection
		testProvider = ai.NewClaudeProviderWithConfig(apiKey, true, aiHubEndpoint, "claude-sonnet")
	} else {
		// Test direct Anthropic API connection
		testProvider = ai.NewClaudeProviderWithConfig(apiKey, false, "", "claude-3-5-sonnet-20241022")
	}

	// Test the connection with a simple request
	testMessage := "Hello, this is a connection test."
	response, err := testProvider.GenerateResponse(ctx, testMessage, []protocol.Message{})
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Connection failed: %v", err)), nil
	}

	if response == "" {
		return ch.systemResponse(msg.Channel, "❌ Connection failed: Empty response"), nil
	}

	return ch.systemResponse(msg.Channel, "✅ Anthropic connection successful!"), nil
}

// handleTestGitHubConnection tests GitHub API connection
func (ch *CommandHandler) handleTestGitHubConnection(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// Extract GitHub token from message metadata
	token, ok := msg.Metadata["github_token"].(string)
	if !ok || token == "" {
		return ch.systemResponse(msg.Channel, "❌ No GitHub token provided in request"), nil
	}

	// Test GitHub API connection by making a simple request
	// This would typically make a request to GitHub's API to verify the token
	// For now, we'll do a basic validation of the token format
	if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		return ch.systemResponse(msg.Channel, "❌ Invalid GitHub token format"), nil
	}

	// TODO: Implement actual GitHub API test
	// This would involve making a request to https://api.github.com/user with the token
	// For now, we'll just validate the format
	return ch.systemResponse(msg.Channel, "✅ GitHub token format is valid (connection test not implemented yet)"), nil
}

// handleTestConfluenceConnection tests Confluence API connection
func (ch *CommandHandler) handleTestConfluenceConnection(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// Extract Confluence credentials from message metadata
	credentials, ok := msg.Metadata["confluence_credentials"].(map[string]interface{})
	if !ok {
		return ch.systemResponse(msg.Channel, "❌ No Confluence credentials provided in request"), nil
	}

	domain, _ := credentials["domain"].(string)
	email, _ := credentials["email"].(string)
	apiToken, _ := credentials["api_token"].(string)

	if domain == "" || email == "" || apiToken == "" {
		return ch.systemResponse(msg.Channel, "❌ Missing required Confluence credentials (domain, email, or api_token)"), nil
	}

	// TODO: Implement actual Confluence API test
	// This would involve making a request to the Confluence REST API
	// For now, we'll just validate the credentials format
	if !strings.Contains(domain, ".") {
		return ch.systemResponse(msg.Channel, "❌ Invalid Confluence domain format"), nil
	}

	if !strings.Contains(email, "@") {
		return ch.systemResponse(msg.Channel, "❌ Invalid email format"), nil
	}

	return ch.systemResponse(msg.Channel, "✅ Confluence credentials format is valid (connection test not implemented yet)"), nil
}

// handleMigrateAgentNames migrates existing agents with problematic names to @mention-compatible format
func (ch *CommandHandler) handleMigrateAgentNames(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	var response strings.Builder
	response.WriteString("🔄 Migrating agent names for @mention compatibility...\n\n")

	// Track migrations
	migrated := []string{}
	errors := []string{}

	// 1. Migrate repository agents
	response.WriteString("**Repository Agents:**\n")
	repoStorage, err := repo.NewStorage()
	if err == nil {
		cachedRepos, err := repoStorage.GetAllCachedRepos()
		if err == nil {
			for _, repoData := range cachedRepos {
				if name, ok := repoData["name"].(string); ok {
					normalized := protocol.NormalizeAgentName(name)
					if name != normalized {
						response.WriteString(fmt.Sprintf("  • %s → %s\n", name, normalized))
						migrated = append(migrated, fmt.Sprintf("%s (repo)", name))
					}
				}
			}
		} else {
			errors = append(errors, fmt.Sprintf("Failed to load cached repos: %v", err))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Failed to initialize repo storage: %v", err))
	}

	// 2. Migrate helper agents
	response.WriteString("\n**Helper Agents:**\n")
	helperStorage, err := agent.NewHelperAgentStorage()
	if err == nil {
		configs, err := helperStorage.ListConfigsWithMetadata()
		if err == nil {
			for _, configData := range configs {
				if name, ok := configData["name"].(string); ok {
					normalized := protocol.NormalizeAgentName(name)
					if name != normalized {
						response.WriteString(fmt.Sprintf("  • %s → %s\n", name, normalized))
						migrated = append(migrated, fmt.Sprintf("%s (helper)", name))
					}
				}
			}
		} else {
			errors = append(errors, fmt.Sprintf("Failed to load helper configs: %v", err))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Failed to initialize helper storage: %v", err))
	}

	// 3. Migrate Confluence agents (check exports)
	response.WriteString("\n**Confluence Agents:**\n")
	exports, err := ch.exportStorage.ListExports()
	if err == nil {
		for _, export := range exports {
			if export.Type == "confluence" {
				normalized := protocol.NormalizeAgentName(export.Name)
				if export.Name != normalized {
					response.WriteString(fmt.Sprintf("  • %s → %s\n", export.Name, normalized))
					migrated = append(migrated, fmt.Sprintf("%s (confluence)", export.Name))
				}
			}
		}
	} else {
		errors = append(errors, fmt.Sprintf("Failed to load exports: %v", err))
	}

	// Summary
	response.WriteString("\n**Summary:**\n")
	if len(migrated) > 0 {
		response.WriteString(fmt.Sprintf("✅ Found %d agents with names that need migration:\n", len(migrated)))
		for _, agent := range migrated {
			response.WriteString(fmt.Sprintf("  • %s\n", agent))
		}
		response.WriteString("\n⚠️  **Note:** This is a read-only check. Agent names will be automatically normalized when agents are recreated.\n")
		response.WriteString("To apply migrations, restart the agents or recreate them.\n")
	} else {
		response.WriteString("✅ All agent names are already @mention-compatible!\n")
	}

	if len(errors) > 0 {
		response.WriteString(fmt.Sprintf("\n❌ %d errors during migration check:\n", len(errors)))
		for _, err := range errors {
			response.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// AddPendingReview adds a pending review to track
func (ch *CommandHandler) AddPendingReview(repoPath string, originalMsg *protocol.Message, agentName string) {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	ch.pendingReviews[repoPath] = &protocol.PendingReview{
		OriginalMessage: originalMsg,
		RepoPath:        repoPath,
		RepoAgentName:   agentName,
		CreatedAt:       time.Now(),
	}
}

// GetPendingReview retrieves a pending review by repo path
func (ch *CommandHandler) GetPendingReview(repoPath string) *protocol.PendingReview {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	return ch.pendingReviews[repoPath]
}

// RemovePendingReview removes a pending review
func (ch *CommandHandler) RemovePendingReview(repoPath string) {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	delete(ch.pendingReviews, repoPath)
}

// HasPendingReview checks if there's already a pending review for a path
func (ch *CommandHandler) HasPendingReview(repoPath string) bool {
	ch.pendingMutex.Lock()
	defer ch.pendingMutex.Unlock()

	_, exists := ch.pendingReviews[repoPath]
	return exists
}

// handleOpenFile opens a file in the editor (placeholder for now)
func (ch *CommandHandler) handleOpenFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /open-file <file-path>"), nil
	}

	filePath := strings.Join(parts[1:], " ")

	// For now, just return a message indicating the file should be opened
	// The frontend will handle the actual opening
	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📂 Opening file: %s\n(File will be opened in the editor panel)", filePath)), nil
}

// handleAddWorkspace adds a new workspace
func (ch *CommandHandler) handleAddWorkspace(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /add-workspace <path> [name]"), nil
	}

	path := parts[1]
	name := path
	if len(parts) > 2 {
		name = strings.Join(parts[2:], " ")
	}

	// Get workspace manager from hub (we'll need to add this to the hub)
	// For now, return a placeholder message
	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📁 Adding workspace: %s\nPath: %s\n(Workspace management will be available in the file explorer panel)", name, path)), nil
}

// handleListWorkspaces lists all workspaces
func (ch *CommandHandler) handleListWorkspaces(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// For now, return a placeholder message
	// The actual implementation will use the workspace manager
	return ch.systemResponse(msg.Channel,
		"📁 Workspaces:\n(Workspace list will be available in the file explorer panel)"), nil
}

// Assistant command handlers

// handleReminder handles reminder-related commands
func (ch *CommandHandler) handleReminder(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/remind <time> <message>` or `/remind-recurring <schedule> <message>`\n"+
				"Examples:\n"+
				"• `/remind in 30 minutes Review the PR`\n"+
				"• `/remind at 3pm Standup meeting`\n"+
				"• `/remind-recurring daily 9am Daily standup`"), nil
	}

	command := parts[0]
	timeStr := parts[1]
	message := strings.Join(parts[2:], " ")

	if message == "" {
		return ch.systemResponse(msg.Channel, "❌ Please provide a reminder message"), nil
	}

	// For now, return a placeholder response
	// The actual implementation would use the assistant agent's storage
	if command == "/remind-recurring" {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("⏰ Recurring reminder set: '%s' at %s\n(Recurring reminders will be implemented in the assistant agent)", message, timeStr)), nil
	} else {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("⏰ Reminder set: '%s' at %s\n(Reminders will be implemented in the assistant agent)", message, timeStr)), nil
	}
}

// handleTask handles task-related commands
func (ch *CommandHandler) handleTask(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]

	switch command {
	case "/task-add":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-add <title>`"), nil
		}
		title := strings.Join(parts[1:], " ")
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Task added: '%s'\n(Task management will be implemented in the assistant agent)", title)), nil

	case "/task-list":
		return ch.systemResponse(msg.Channel,
			"📋 Task List:\n(Task list will be implemented in the assistant agent)"), nil

	case "/task-done":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-done <task-id>`"), nil
		}
		taskID := parts[1]
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("✅ Task '%s' marked as done\n(Task completion will be implemented in the assistant agent)", taskID)), nil

	default:
		return ch.systemResponse(msg.Channel, "❌ Unknown task command"), nil
	}
}

// handleNote handles note-related commands
func (ch *CommandHandler) handleNote(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]

	switch command {
	case "/note-save":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-save <content>`"), nil
		}
		content := strings.Join(parts[1:], " ")
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Note saved: '%s'\n(Note saving will be implemented in the assistant agent)", content)), nil

	case "/note-search":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-search <query>`"), nil
		}
		query := strings.Join(parts[1:], " ")
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("🔍 Searching notes for: '%s'\n(Note search will be implemented in the assistant agent)", query)), nil

	default:
		return ch.systemResponse(msg.Channel, "❌ Unknown note command"), nil
	}
}

// handleMeeting handles meeting-related commands
func (ch *CommandHandler) handleMeeting(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/meeting-add <time> <title>`\n"+
				"Example: `/meeting-add tomorrow 2pm Team standup`"), nil
	}

	timeStr := parts[1]
	title := strings.Join(parts[2:], " ")

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📅 Meeting added: '%s' at %s\n(Meeting management will be implemented in the assistant agent)", title, timeStr)), nil
}

// handleSummarize handles conversation summarization
func (ch *CommandHandler) handleSummarize(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	// Get recent messages from the channel
	messages, err := ch.hub.GetMessages(msg.Channel, 20)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get messages: %v", err)), nil
	}

	if len(messages) == 0 {
		return ch.systemResponse(msg.Channel, "❌ No messages to summarize"), nil
	}

	// For now, return a placeholder response
	// The actual implementation would use the assistant agent's AI capabilities
	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📄 Summarizing last %d messages...\n(Conversation summarization will be implemented in the assistant agent)", len(messages))), nil
}

// handleAssistantHelp shows help for assistant commands
func (ch *CommandHandler) handleAssistantHelp(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	help := "🤖 **Assistant Commands**\n\n" +
		"**Reminders:**\n" +
		"• `/remind <time> <message>` - Set a reminder\n" +
		"• `/remind-recurring <schedule> <message>` - Set recurring reminder\n\n" +
		"**Tasks:**\n" +
		"• `/task-add <title>` - Add a task\n" +
		"• `/task-list` - List all tasks\n" +
		"• `/task-done <id>` - Mark task complete\n\n" +
		"**Notes:**\n" +
		"• `/note-save <content>` - Save a note\n" +
		"• `/note-search <query>` - Search notes\n\n" +
		"**Meetings:**\n" +
		"• `/meeting-add <time> <title>` - Add meeting\n\n" +
		"**Other:**\n" +
		"• `/summarize [last N messages]` - Summarize conversation\n\n" +
		"**Examples:**\n" +
		"• `/remind in 30 minutes Review the PR`\n" +
		"• `/remind at 3pm Standup meeting`\n" +
		"• `/task-add Fix the bug in login`\n" +
		"• `/note-save Important: API key is abc123`\n" +
		"• `/meeting-add tomorrow 2pm Team standup`"

	return ch.systemResponse(msg.Channel, help), nil
}

// handleAnalyzeDesign handles design analysis requests
func (ch *CommandHandler) handleAnalyzeDesign(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	// Check if there's an image in the message metadata
	imageData, hasImage := msg.Metadata["image_data"].([]byte)
	imageType, hasImageType := msg.Metadata["image_type"].(string)

	if !hasImage || !hasImageType {
		return ch.systemResponse(msg.Channel,
			"❌ No image found for design analysis.\n\n"+
				"**Usage:**\n"+
				"1. Upload an image using the file picker\n"+
				"2. Type your message with @mentions to target specific agents\n"+
				"3. Send `/analyze-design` command\n\n"+
				"**Supported formats:** PNG, JPEG, WebP, GIF\n"+
				"**Max size:** 5MB"), nil
	}

	// Validate image size (5MB limit)
	if len(imageData) > 5*1024*1024 {
		return ch.systemResponse(msg.Channel,
			"❌ Image too large. Maximum size is 5MB. Please compress the image and try again."), nil
	}

	// Get channel agents
	channelAgents, err := ch.hub.GetChannelAgents(msg.Channel)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Failed to get channel agents"), nil
	}

	// Parse mentions from the message content
	mentionedAgentNames := protocol.ParseMentions(msg.Content)
	if len(mentionedAgentNames) == 0 {
		return ch.systemResponse(msg.Channel,
			"❌ No agents mentioned for design analysis.\n\n"+
				"**Usage:**\n"+
				"1. Upload an image\n"+
				"2. Type your message with @mentions (e.g., \"@FrontendAgent please analyze this design\")\n"+
				"3. Send `/analyze-design` command\n\n"+
				"**Available vision-capable agents:**\n"+
				ch.getVisionCapableAgentsList(channelAgents)), nil
	}

	// Find mentioned agents that support vision
	var targetAgents []protocol.AgentInfo
	for _, agent := range channelAgents {
		for _, mentionedName := range mentionedAgentNames {
			if strings.EqualFold(agent.Name, mentionedName) && agent.SupportsVision {
				targetAgents = append(targetAgents, agent)
				break
			}
		}
	}

	if len(targetAgents) == 0 {
		return ch.systemResponse(msg.Channel,
			"❌ No vision-capable agents found among mentioned agents.\n\n"+
				"**Mentioned agents:** "+strings.Join(mentionedAgentNames, ", ")+"\n"+
				"**Available vision-capable agents:**\n"+
				ch.getVisionCapableAgentsList(channelAgents)), nil
	}

	// Create analysis message for each target agent
	var agentNames []string
	for _, agent := range targetAgents {
		agentNames = append(agentNames, agent.Name)

		// Create a special message for the agent with design analysis flag
		designMsg := &protocol.Message{
			ID:        protocol.NewMessage(protocol.MessageTypeChat, msg.Channel, msg.From, "").ID,
			Type:      protocol.MessageTypeChat,
			Channel:   msg.Channel,
			From:      msg.From,
			Content:   msg.Content, // Use the original message content with mentions
			Timestamp: msg.Timestamp,
			Metadata: map[string]interface{}{
				"design_analysis": true,
				"image_data":      imageData,
				"image_type":      imageType,
			},
		}

		// Send the message to trigger agent analysis
		if err := ch.hub.SendMessage(designMsg); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to send design analysis request to %s: %v", agent.Name, err)), nil
		}
	}

	return ch.systemResponse(msg.Channel,
		"🎨 Design analysis started!\n\n"+
			"**Processing:** Analyzing design mockup...\n"+
			"**Target Agents:** "+strings.Join(agentNames, ", ")+"\n"+
			"**Output:** CSS style guide + HTML demo\n\n"+
			"The mentioned agents will analyze the design and generate:\n"+
			"• Complete CSS file with extracted styles\n"+
			"• HTML demo showcasing the design\n"+
			"• Markdown style guide with design tokens\n\n"+
			"Please wait for the analysis to complete..."), nil
}

// getVisionCapableAgentsList returns a formatted list of vision-capable agents
func (ch *CommandHandler) getVisionCapableAgentsList(agents []protocol.AgentInfo) string {
	var visionAgents []string
	for _, agent := range agents {
		if agent.SupportsVision {
			visionAgents = append(visionAgents, "• @"+agent.Name)
		}
	}

	if len(visionAgents) == 0 {
		return "No vision-capable agents available in this channel."
	}

	return strings.Join(visionAgents, "\n")
}

// handleApproveFile approves a file change
func (ch *CommandHandler) handleApproveFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-file <change-id>"), nil
	}

	changeID := parts[1]
	fileChangeManager := ch.hub.GetFileChangeManager()

	// Approve the file change
	change, err := fileChangeManager.ApproveFileChange(changeID, msg.From.ID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to approve file change: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("✅ File change approved and executed!\n\n"+
			"**Change ID:** %s\n"+
			"**Operation:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s",
			change.ID, change.Operation, change.GetDisplayPath(), change.Agent.Name)), nil
}

// handleRejectFile rejects a file change
func (ch *CommandHandler) handleRejectFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /reject-file <change-id> [reason]"), nil
	}

	changeID := parts[1]
	reason := "No reason provided"
	if len(parts) > 2 {
		reason = strings.Join(parts[2:], " ")
	}

	fileChangeManager := ch.hub.GetFileChangeManager()

	// Reject the file change
	change, err := fileChangeManager.RejectFileChange(changeID, msg.From.ID, reason)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to reject file change: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("❌ File change rejected!\n\n"+
			"**Change ID:** %s\n"+
			"**Operation:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s\n"+
			"**Reason:** %s",
			change.ID, change.Operation, change.GetDisplayPath(), change.Agent.Name, reason)), nil
}

// handleApproveDelete approves a delete operation (requires explicit confirmation)
func (ch *CommandHandler) handleApproveDelete(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-delete <change-id>"), nil
	}

	changeID := parts[1]
	fileChangeManager := ch.hub.GetFileChangeManager()

	// Get the change first to verify it's a delete operation
	change, err := fileChangeManager.GetFileChange(changeID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get file change: %v", err)), nil
	}

	if !change.IsDeleteOperation() {
		return ch.systemResponse(msg.Channel, "❌ This is not a delete operation. Use /approve-file instead."), nil
	}

	// Approve the delete operation
	change, err = fileChangeManager.ApproveFileChange(changeID, msg.From.ID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to approve delete operation: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🗑️ Delete operation approved and executed!\n\n"+
			"**Change ID:** %s\n"+
			"**File:** %s\n"+
			"**Agent:** %s\n\n"+
			"⚠️ File has been deleted and backed up.",
			change.ID, change.FilePath, change.Agent.Name)), nil
}

// handleListFileChanges lists all pending file changes
func (ch *CommandHandler) handleListFileChanges(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	fileChangeManager := ch.hub.GetFileChangeManager()
	pendingChanges := fileChangeManager.ListPendingFileChanges(msg.From.ID)

	if len(pendingChanges) == 0 {
		return ch.systemResponse(msg.Channel, "📝 No pending file changes found."), nil
	}

	var response strings.Builder
	response.WriteString("📝 **Pending File Changes:**\n\n")

	for i, change := range pendingChanges {
		timeRemaining := change.GetTimeRemaining()
		timeStr := "expired"
		if timeRemaining > 0 {
			timeStr = fmt.Sprintf("%.0f minutes", timeRemaining.Minutes())
		}

		response.WriteString(fmt.Sprintf("**%d.** `%s`\n", i+1, change.ID))
		response.WriteString(fmt.Sprintf("   • **Operation:** %s\n", change.Operation))
		response.WriteString(fmt.Sprintf("   • **File:** %s\n", change.GetDisplayPath()))
		response.WriteString(fmt.Sprintf("   • **Agent:** %s\n", change.Agent.Name))
		response.WriteString(fmt.Sprintf("   • **Time remaining:** %s\n", timeStr))

		if change.IsDeleteOperation() {
			response.WriteString(fmt.Sprintf("   • **⚠️ DELETE OPERATION** - Use `/approve-delete %s`\n", change.ID))
		} else {
			response.WriteString(fmt.Sprintf("   • **Approve:** `/approve-file %s`\n", change.ID))
		}
		response.WriteString(fmt.Sprintf("   • **Reject:** `/reject-file %s [reason]`\n\n", change.ID))
	}

	response.WriteString("💡 **Commands:**\n")
	response.WriteString("• `/approve-file <id>` - Approve a file change\n")
	response.WriteString("• `/reject-file <id> [reason]` - Reject a file change\n")
	response.WriteString("• `/approve-delete <id>` - Approve a delete operation\n")

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleIngestMeetings manually triggers ingestion of all meeting notes
func (ch *CommandHandler) handleIngestMeetings(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	// Find the assistant agent
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	// Trigger manual ingestion
	go func() {
		// Get all markdown files in the meeting notes directory
		config := assistantAgent.GetConfig()
		if config == nil {
			return
		}

		files, err := filepath.Glob(filepath.Join(config.MeetingNotesDir, "*.md"))
		if err != nil {
			ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to list meeting notes directory: %v", err))
			return
		}

		processed := 0
		for _, file := range files {
			// Process each file
			assistantAgent.ProcessMeetingNote(ctx, file)
			processed++
		}

		ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Ingested %d meeting notes successfully!", processed))
	}()

	return ch.systemResponse(msg.Channel, "🔄 Starting manual ingestion of meeting notes..."), nil
}

// handleSearchMeetings searches meeting notes by query
func (ch *CommandHandler) handleSearchMeetings(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/search-meetings <query>`"), nil
	}

	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	query := strings.Join(parts[1:], " ")
	notes, err := assistantAgent.SearchMeetingNotes(query)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to search meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("🔍 No meeting notes found for query: '%s'", query)), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Found %d meeting notes for '%s':**\n\n", len(notes), query))

	for i, note := range notes {
		if i >= 5 { // Limit to 5 results
			response.WriteString(fmt.Sprintf("... and %d more results\n", len(notes)-5))
			break
		}

		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("📅 %s\n", note.MeetingDate.Format("2006-01-02 15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n\n", note.Summary[:minInt(100, len(note.Summary))]))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleMeetingSummary gets a summary of a specific meeting
func (ch *CommandHandler) handleMeetingSummary(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/meeting-summary <date>` (e.g., 2025-10-21)"), nil
	}

	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	dateStr := parts[1]
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Invalid date format. Use YYYY-MM-DD"), nil
	}

	// Search for meetings on that date
	notes, err := assistantAgent.GetMeetingNotesByDate(date)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("📅 No meetings found for %s", dateStr)), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📅 **Meetings on %s:**\n\n", dateStr))

	for _, note := range notes {
		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("🕐 %s\n", note.MeetingDate.Format("15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n", note.Summary))
		if len(note.ActionItems) > 0 {
			response.WriteString("✅ **Action Items:**\n")
			for _, item := range note.ActionItems {
				response.WriteString(fmt.Sprintf("   • %s\n", item))
			}
		}
		if note.GoogleDocLink != "" {
			response.WriteString(fmt.Sprintf("🔗 [View Full Notes](%s)\n", note.GoogleDocLink))
		}
		response.WriteString("\n")
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleActionItems lists all pending action items from meeting notes
func (ch *CommandHandler) handleActionItems(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	actionItems, err := assistantAgent.GetPendingActionItems()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get action items: %v", err)), nil
	}

	if len(actionItems) == 0 {
		return ch.systemResponse(msg.Channel, "✅ No pending action items found in meeting notes."), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📋 **Pending Action Items (%d):**\n\n", len(actionItems)))

	for i, item := range actionItems {
		if i >= 10 { // Limit to 10 items
			response.WriteString(fmt.Sprintf("... and %d more items\n", len(actionItems)-10))
			break
		}
		response.WriteString(fmt.Sprintf("• %s\n", item))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleListMeetings lists recent meeting notes
func (ch *CommandHandler) handleListMeetings(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	assistantAgent := ch.findAssistantAgent()
	if assistantAgent == nil {
		return ch.systemResponse(msg.Channel, "❌ Assistant agent not found. Please ensure the assistant agent is running."), nil
	}

	notes, err := assistantAgent.GetRecentMeetingNotes(10)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get meeting notes: %v", err)), nil
	}

	if len(notes) == 0 {
		return ch.systemResponse(msg.Channel, "📅 No meeting notes found."), nil
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📅 **Recent Meeting Notes (%d):**\n\n", len(notes)))

	for _, note := range notes {
		response.WriteString(fmt.Sprintf("**%s**\n", note.Title))
		response.WriteString(fmt.Sprintf("📅 %s\n", note.MeetingDate.Format("2006-01-02 15:04")))
		if len(note.Attendees) > 0 {
			response.WriteString(fmt.Sprintf("👥 %s\n", strings.Join(note.Attendees, ", ")))
		}
		response.WriteString(fmt.Sprintf("📝 %s\n\n", note.Summary[:minInt(100, len(note.Summary))]))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// findAssistantAgent finds the assistant agent in the hub
func (ch *CommandHandler) findAssistantAgent() *agent.AssistantAgent {
	return ch.assistantAgent
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleSwitchProvider handles /switch-provider command
func (ch *CommandHandler) handleSwitchProvider(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "Usage: /switch-provider <agent-name> <provider> [model]\nProviders: claude, ollama, lmstudio\nExample: /switch-provider BackendExpert ollama llama3.1"), nil
	}

	agentName := parts[1]
	provider := strings.ToLower(parts[2])
	model := ""
	if len(parts) > 3 {
		model = parts[3]
	}

	// Validate provider
	if provider != "claude" && provider != "ollama" && provider != "lmstudio" {
		return ch.systemResponse(msg.Channel, "❌ Invalid provider. Use 'claude', 'ollama', or 'lmstudio'"), nil
	}

	// Find the agent
	var targetAgent *protocol.AgentInfo
	for _, agent := range ch.hub.ListAgents() {
		if agent.Name == agentName {
			targetAgent = agent
			break
		}
	}

	if targetAgent == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
	}

	// Note: Provider switching is handled at the agent level
	// This command updates the agent info but doesn't actually switch the running agent's provider

	// Find the actual agent instance and switch provider
	// Note: This is a simplified implementation - in practice you'd need to access the running agent
	// For now, we'll just update the agent info and broadcast the change
	targetAgent.AIProvider = provider
	targetAgent.AIModel = model
	targetAgent.Model = model

	// Broadcast the change
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		msg.Channel,
		*targetAgent,
		fmt.Sprintf("🔄 %s switched to %s (%s)", targetAgent.Name, provider, model),
	)
	statusMsg.Metadata = map[string]interface{}{
		"ai_provider": provider,
		"ai_model":    model,
		"model":       model,
	}

	ch.hub.SendMessage(statusMsg)

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ %s switched to %s (%s)", agentName, provider, model)), nil
}

// handleSwitchAllProviders handles /switch-all-providers command
func (ch *CommandHandler) handleSwitchAllProviders(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /switch-all-providers <provider> [model]\nProviders: claude, ollama, lmstudio\nExample: /switch-all-providers ollama llama3.1"), nil
	}

	provider := strings.ToLower(parts[1])
	model := ""
	if len(parts) > 2 {
		model = parts[2]
	}

	// Validate provider
	if provider != "claude" && provider != "ollama" && provider != "lmstudio" {
		return ch.systemResponse(msg.Channel, "❌ Invalid provider. Use 'claude', 'ollama', or 'lmstudio'"), nil
	}

	// Set default models
	if model == "" {
		if provider == "ollama" {
			model = "llama3.1"
		} else if provider == "lmstudio" {
			model = "" // Will be determined from available models
		} else {
			model = "claude-sonnet"
		}
	}

	// Switch all agents
	agents := ch.hub.ListAgents()
	switchedCount := 0

	for _, agent := range agents {
		// Update agent info
		agent.AIProvider = provider
		agent.AIModel = model
		agent.Model = model

		// Broadcast the change
		statusMsg := protocol.NewMessage(
			protocol.MessageTypeAgentStatus,
			msg.Channel,
			*agent,
			fmt.Sprintf("🔄 %s switched to %s (%s)", agent.Name, provider, model),
		)
		statusMsg.Metadata = map[string]interface{}{
			"ai_provider": provider,
			"ai_model":    model,
			"model":       model,
		}

		ch.hub.SendMessage(statusMsg)
		switchedCount++
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Switched %d agents to %s (%s)", switchedCount, provider, model)), nil
}


// SetAssistantAgent sets the assistant agent reference for meeting notes functionality
func (ch *CommandHandler) SetAssistantAgent(assistant *agent.AssistantAgent) {
	ch.assistantAgent = assistant
}

// Ensure CommandHandler implements CommandHandlerInterface
var _ agent.CommandHandlerInterface = (*CommandHandler)(nil)

// GetCommandDefinitions returns metadata for every registered slash command.
func (ch *CommandHandler) GetCommandDefinitions() []protocol.CommandDefinition {
	providerOpts := []string{"ollama", "claude", "lmstudio"}

	return []protocol.CommandDefinition{
		// ── Repository Agents ──────────────────────────────────────────
		{
			Name:        "/create-repo-agent",
			Description: "Create a new repository expert agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "repo-path", Description: "Path to the repository", Type: "path", Required: true},
				{Name: "agent-name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/reindex-agent",
			Description: "Re-index a repository agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/enable-watch",
			Description: "Enable file watching for a repo agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/disable-watch",
			Description: "Disable file watching for a repo agent",
			Category:    "Repository Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the repo agent", Type: "agent-name", Required: true},
			},
		},

		// ── Confluence ─────────────────────────────────────────────────
		{
			Name:        "/create-confluence-agent",
			Description: "Create a Confluence documentation agent",
			Category:    "Confluence",
			Arguments: []protocol.CommandArgument{
				{Name: "space-key", Description: "Confluence space key", Type: "string", Required: true},
				{Name: "agent-name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/reindex-confluence-agent",
			Description: "Re-index a Confluence agent",
			Category:    "Confluence",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the Confluence agent", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-confluence-agents",
			Description: "List all Confluence agents",
			Category:    "Confluence",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Helper Agents ──────────────────────────────────────────────
		{
			Name:        "/create-helper",
			Description: "Create a custom helper/expert agent from a template",
			Category:    "Helper Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "template", Description: "Template name (e.g. day-one, testing-expert)", Type: "string", Required: true},
				{Name: "agent-name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/list-helper-templates",
			Description: "List available helper agent templates",
			Category:    "Helper Agents",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Agent Management ───────────────────────────────────────────
		{
			Name:        "/delete-agent",
			Description: "Permanently delete an agent and its data",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to delete", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/pause-agent",
			Description: "Pause an agent so it stops responding",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to pause", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/unpause-agent",
			Description: "Unpause a paused agent",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to unpause", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-agents",
			Description: "List all agents (active, available, removed)",
			Category:    "Agent Management",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/remove-agent",
			Description: "Remove an agent from the current conversation",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to remove", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/recall-agent",
			Description: "Recall a previously removed agent",
			Category:    "Agent Management",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to recall", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-removed-agents",
			Description: "List agents that have been removed from channels",
			Category:    "Agent Management",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── MCP Export/Import ──────────────────────────────────────────
		{
			Name:        "/export-agent-mcp",
			Description: "Export an agent to MCP format",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent to export", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-exports",
			Description: "List all MCP exports",
			Category:    "MCP Export/Import",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/delete-export",
			Description: "Delete an MCP export",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "export-name", Description: "Name of the export to delete", Type: "string", Required: true},
			},
		},
		{
			Name:        "/import-agent-mcp",
			Description: "Import an agent from an MCP file",
			Category:    "MCP Export/Import",
			Arguments: []protocol.CommandArgument{
				{Name: "file-path", Description: "Path to the MCP export file", Type: "path", Required: true},
			},
		},
		{
			Name:        "/export-all-agents",
			Description: "Export all agents to MCP format",
			Category:    "MCP Export/Import",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Connection Tests ───────────────────────────────────────────
		{
			Name:        "/test-anthropic-connection",
			Description: "Test the Anthropic API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/test-github-connection",
			Description: "Test the GitHub API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/test-confluence-connection",
			Description: "Test the Confluence API connection",
			Category:    "Connection Tests",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Provider ───────────────────────────────────────────────────
		{
			Name:        "/switch-provider",
			Description: "Switch one agent's AI provider",
			Category:    "Provider",
			Arguments: []protocol.CommandArgument{
				{Name: "agent-name", Description: "Name of the agent", Type: "agent-name", Required: true},
				{Name: "provider", Description: "New AI provider", Type: "provider", Required: true, Options: providerOpts},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},
		{
			Name:        "/switch-all-providers",
			Description: "Switch all agents to the same AI provider",
			Category:    "Provider",
			Arguments: []protocol.CommandArgument{
				{Name: "provider", Description: "New AI provider", Type: "provider", Required: true, Options: providerOpts},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
		},

		// ── Meetings ───────────────────────────────────────────────────
		{
			Name:        "/ingest-meetings",
			Description: "Ingest meeting notes from configured source",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/search-meetings",
			Description: "Search through ingested meeting notes",
			Category:    "Meetings",
			Arguments: []protocol.CommandArgument{
				{Name: "query", Description: "Search query", Type: "string", Required: true},
			},
		},
		{
			Name:        "/meeting-summary",
			Description: "Get a summary of meetings for a specific date",
			Category:    "Meetings",
			Arguments: []protocol.CommandArgument{
				{Name: "date", Description: "Date (e.g. 2025-01-15 or today)", Type: "string", Required: false, Default: "today"},
			},
		},
		{
			Name:        "/action-items",
			Description: "List pending action items from meetings",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/list-meetings",
			Description: "List recent meeting notes",
			Category:    "Meetings",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Files & Workspace ──────────────────────────────────────────
		{
			Name:        "/open-file",
			Description: "Open a file in the code editor",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "file-path", Description: "Path to the file to open", Type: "path", Required: true},
			},
		},
		{
			Name:        "/add-workspace",
			Description: "Add a workspace directory",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "path", Description: "Path to the workspace directory", Type: "path", Required: true},
			},
		},
		{
			Name:        "/list-workspaces",
			Description: "List configured workspaces",
			Category:    "Files & Workspace",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/approve-file",
			Description: "Approve a pending file change",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the file change to approve", Type: "string", Required: true},
			},
		},
		{
			Name:        "/reject-file",
			Description: "Reject a pending file change",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the file change to reject", Type: "string", Required: true},
			},
		},
		{
			Name:        "/approve-delete",
			Description: "Approve a pending file delete operation",
			Category:    "Files & Workspace",
			Arguments: []protocol.CommandArgument{
				{Name: "change-id", Description: "ID of the delete operation to approve", Type: "string", Required: true},
			},
		},
		{
			Name:        "/list-file-changes",
			Description: "List all pending file changes",
			Category:    "Files & Workspace",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Design ─────────────────────────────────────────────────────
		{
			Name:        "/analyze-design",
			Description: "Analyze an uploaded design image with vision agents",
			Category:    "Design",
			Arguments: []protocol.CommandArgument{
				{Name: "image-url", Description: "URL or path to the design image", Type: "string", Required: true},
			},
		},

		// ── Migration ──────────────────────────────────────────────────
		{
			Name:        "/migrate-agent-names",
			Description: "Check and migrate agent names for @mention compatibility",
			Category:    "Migration",
			Arguments:   []protocol.CommandArgument{},
		},

		// ── Help ───────────────────────────────────────────────────────
		{
			Name:        "/help",
			Description: "Show all available commands",
			Category:    "Help",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/help-assistant",
			Description: "Show assistant-specific commands and features",
			Category:    "Help",
			Arguments:   []protocol.CommandArgument{},
		},
	}
}
