package hub

import (
	"context"
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

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// CommandHandler handles chat commands
type CommandHandler struct {
	hub              *Hub
	aiProvider       ai.AIProvider
	repoAgents       map[string]*agent.RepoAgent        // Track repo agents for management
	helperAgents     map[string]*agent.HelperAgent      // Track helper agents for management
	confluenceAgents map[string]*agent.ConfluenceAgent  // Track confluence agents for management
	cliAgents        map[string]*agent.Agent            // Track CLI proxy agents
	runtimeAgents    map[string]*agent.Agent            // Track runtime specialist/moderator/assistant/CLI agents
	assistantAgent   *agent.AssistantAgent              // Track assistant agent for meeting notes
	exportStorage    *mcp_export.ExportStorage          // Export storage for MCP exports
	pendingReviews   map[string]*protocol.PendingReview // Track pending reviews by repo path
	pendingMutex     sync.Mutex                         // Protects pending reviews map
}

type commandExecutor func(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error)

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

	ch := &CommandHandler{
		hub:              hub,
		aiProvider:       aiProvider,
		repoAgents:       make(map[string]*agent.RepoAgent),
		helperAgents:     make(map[string]*agent.HelperAgent),
		confluenceAgents: make(map[string]*agent.ConfluenceAgent),
		cliAgents:        make(map[string]*agent.Agent),
		runtimeAgents:    make(map[string]*agent.Agent),
		exportStorage:    exportStorage,
		pendingReviews:   make(map[string]*protocol.PendingReview),
	}
	ch.validateCommandDefinitions()
	return ch, nil
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
	executor, ok := ch.commandExecutors()[command]
	if !ok {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unknown command: %s\nUse /help to see available commands.", command)), nil
	}
	return executor(ctx, msg, parts)
}

func (ch *CommandHandler) commandExecutors() map[string]commandExecutor {
	return map[string]commandExecutor{
		"/create-repo-agent":       ch.handleCreateRepoAgent,
		"/create-confluence-agent": ch.handleCreateConfluenceAgent,
		"/create-helper":           ch.handleCreateHelper,
		"/create-expert":           ch.handleCreateExpert,
		"/list-helper-templates": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListHelperTemplates(ctx, msg)
		},
		"/delete-agent":             ch.handleDeleteAgent,
		"/reindex-agent":            ch.handleReindexAgent,
		"/reindex-confluence-agent": ch.handleReindexConfluenceAgent,
		"/pause-agent":              ch.handlePauseAgent,
		"/unpause-agent":            ch.handleUnpauseAgent,
		"/enable-watch":             ch.handleEnableWatch,
		"/disable-watch":            ch.handleDisableWatch,
		"/list-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListAgents(ctx, msg)
		},
		"/list-confluence-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListConfluenceAgents(ctx, msg)
		},
		"/remove-agent": ch.handleRemoveAgent,
		"/recall-agent": ch.handleRecallAgent,
		"/list-removed-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListRemovedAgents(ctx, msg)
		},
		"/export-agent-mcp": ch.handleExportAgentMCP,
		"/list-exports": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListExports(ctx, msg)
		},
		"/delete-export":    ch.handleDeleteExport,
		"/import-agent-mcp": ch.handleImportAgentMCP,
		"/export-all-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleExportAllAgents(ctx, msg)
		},
		"/test-anthropic-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestAnthropicConnection(ctx, msg)
		},
		"/test-github-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestGitHubConnection(ctx, msg)
		},
		"/test-confluence-connection": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleTestConfluenceConnection(ctx, msg)
		},
		"/switch-provider":      ch.handleSwitchProvider,
		"/switch-all-providers": ch.handleSwitchAllProviders,
		"/create-channel":       ch.handleCreateChannelCmd,
		"/add-to-channel":       ch.handleAddToChannel,
		"/remove-from-channel":  ch.handleRemoveFromChannel,
		"/list-channels": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListChannels(ctx, msg)
		},
		"/delete-channel":   ch.handleDeleteChannelCmd,
		"/open-terminal":    ch.handleOpenTerminal,
		"/create-cli-agent": ch.handleCreateCLIAgent,
		"/list-cli-agents": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListCLIAgents(ctx, msg)
		},
		"/help": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleHelp(ctx, msg)
		},
		"/migrate-agent-names": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleMigrateAgentNames(ctx, msg)
		},
		"/open-file":     ch.handleOpenFile,
		"/add-workspace": ch.handleAddWorkspace,
		"/list-workspaces": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListWorkspaces(ctx, msg)
		},
		"/remind":           ch.handleReminder,
		"/remind-recurring": ch.handleReminder,
		"/task-add":         ch.handleTask,
		"/task-list":        ch.handleTask,
		"/task-done":        ch.handleTask,
		"/note-save":        ch.handleNote,
		"/note-search":      ch.handleNote,
		"/meeting-add":      ch.handleMeeting,
		"/ingest-meetings": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleIngestMeetings(ctx, msg)
		},
		"/search-meetings": ch.handleSearchMeetings,
		"/meeting-summary": ch.handleMeetingSummary,
		"/action-items": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleActionItems(ctx, msg)
		},
		"/list-meetings": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListMeetings(ctx, msg)
		},
		"/summarize": ch.handleSummarize,
		"/help-assistant": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleAssistantHelp(ctx, msg)
		},
		"/analyze-design": ch.handleAnalyzeDesign,
		"/approve-file":   ch.handleApproveFile,
		"/reject-file":    ch.handleRejectFile,
		"/approve-delete": ch.handleApproveDelete,
		"/list-file-changes": func(ctx context.Context, msg *protocol.Message, _ []string) (*protocol.Message, error) {
			return ch.handleListFileChanges(ctx, msg)
		},
		"/collaborate":   ch.handleCollaborate,
		"/approve-plan":  ch.handleApprovePlan,
		"/revise-plan":   ch.handleRevisePlan,
		"/cancel-plan":   ch.handleCancelPlan,
		"/collab-status": ch.handleCollabStatus,
	}
}

// handleCreateRepoAgent creates a new repository expert agent
func (ch *CommandHandler) handleCreateRepoAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /create-repo-agent <repo-path> [agent-name] [provider] [model]\nProviders: ollama (default), claude, lmstudio\nExample: /create-repo-agent /path/to/repo MyRepoExpert ollama llama3.1"), nil
	}

	repoPath := parts[1]
	agentName := ""
	provider := "ollama" // Default to ollama
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
		response.WriteString("• `/create-expert <type> [name]` - Specialist agent (rust, backend, frontend, ...)\n")
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

// handleCreateExpert creates a specialist agent from a known type (backend, frontend, etc.)
func (ch *CommandHandler) handleCreateExpert(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"Usage: `/create-expert <type> [name] [provider] [model]`\n\n"+
				"**Available types:**\n"+
				"• `rust` - Rust, ownership, lifetimes, async, traits, cargo, WASM\n"+
				"• `backend` - Go, Node.js, Python, REST/GraphQL/gRPC, microservices\n"+
				"• `frontend` - React, Vue, Angular, TypeScript, CSS, UI/UX\n"+
				"• `devops` - Docker, Kubernetes, CI/CD, AWS/GCP/Azure, Terraform\n"+
				"• `database` - PostgreSQL, MySQL, MongoDB, schema, query optimization\n"+
				"• `security` - Authentication, authorization, encryption, OWASP\n\n"+
				"**Examples:**\n"+
				"```\n"+
				"/create-expert rust\n"+
				"/create-expert rust RustGuru\n"+
				"/create-expert backend GoExpert ollama qwen2.5-coder:14b\n"+
				"```"), nil
	}

	expertType := strings.ToLower(parts[1])

	validTypes := map[string]protocol.AgentType{
		"rust":     protocol.AgentTypeRust,
		"backend":  protocol.AgentTypeBackend,
		"frontend": protocol.AgentTypeFrontend,
		"devops":   protocol.AgentTypeDevOps,
		"database": protocol.AgentTypeDatabase,
		"security": protocol.AgentTypeSecurity,
	}

	agentType, ok := validTypes[expertType]
	if !ok {
		typeList := []string{}
		for k := range validTypes {
			typeList = append(typeList, k)
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Unknown expert type '%s'.\n\nValid types: %s",
				expertType, strings.Join(typeList, ", "))), nil
	}

	// Determine name
	name := ""
	if len(parts) >= 3 {
		name = parts[2]
	}
	if name == "" {
		defaults := map[protocol.AgentType]string{
			protocol.AgentTypeRust:     "RustExpert",
			protocol.AgentTypeBackend:  "GoExpert",
			protocol.AgentTypeFrontend: "ReactExpert",
			protocol.AgentTypeDevOps:   "DevOpsPro",
			protocol.AgentTypeDatabase: "SQLMaster",
			protocol.AgentTypeSecurity: "SecurityExpert",
		}
		name = defaults[agentType]
	}

	name = protocol.NormalizeAgentName(name)

	// Check for duplicate
	existingAgents := ch.hub.ListAgents()
	for _, existing := range existingAgents {
		if strings.EqualFold(existing.Name, name) {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Agent '%s' already exists. Use a different name or `/delete-agent %s` first.", name, name)), nil
		}
	}

	// Determine AI provider
	var aiProvider ai.AIProvider
	providerName := ""
	modelOverride := ""

	if len(parts) >= 4 {
		providerName = strings.ToLower(parts[3])
	}
	if len(parts) >= 5 {
		modelOverride = parts[4]
	}

	switch providerName {
	case "claude":
		claudeProvider, err := ai.NewClaudeProvider()
		if err != nil {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("❌ Failed to create Claude provider: %v", err)), nil
		}
		aiProvider = claudeProvider
	case "lmstudio":
		if modelOverride != "" {
			aiProvider = ai.NewLMStudioProviderWithConfig("", modelOverride)
		} else {
			lmProvider, err := ai.NewLMStudioProvider()
			if err != nil {
				return ch.systemResponse(msg.Channel,
					fmt.Sprintf("❌ Failed to create LM Studio provider: %v", err)), nil
			}
			aiProvider = lmProvider
		}
	case "ollama", "":
		if modelOverride != "" {
			aiProvider = ai.NewOllamaProviderWithConfig("", modelOverride)
		} else {
			ollamaProvider, err := ai.NewOllamaProvider()
			if err != nil {
				return ch.systemResponse(msg.Channel,
					fmt.Sprintf("❌ Failed to create Ollama provider: %v", err)), nil
			}
			aiProvider = ollamaProvider
		}
	default:
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Unknown provider '%s'. Use: ollama, claude, lmstudio", providerName)), nil
	}

	// Create agent via factory
	agentInstance, err := agent.AgentFactory(agentType, name, aiProvider, ch.hub)
	if err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to create %s agent: %v", expertType, err)), nil
	}
	agentInstance.SetCollabClient(ch.hub.NewCollaborationClientAdapter())
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance

	// Register with hub
	if err := ch.hub.RegisterAgent(&agentInstance.Info); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
	}

	// Join channel
	if err := ch.hub.JoinChannel(agentInstance.Info.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
	}

	// Start agent
	if err := agentInstance.Start(ctx, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
	}

	expertiseStr := strings.Join(agentInstance.Info.Expertise, ", ")
	if len(agentInstance.Info.Expertise) > 5 {
		expertiseStr = strings.Join(agentInstance.Info.Expertise[:5], ", ") +
			fmt.Sprintf(" and %d more", len(agentInstance.Info.Expertise)-5)
	}

	providerDisplay := aiProvider.GetModel()
	if providerName != "" {
		providerDisplay = providerName + " / " + aiProvider.GetModel()
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🤖 Created **%s** expert agent: **%s**\n\n"+
			"**Type:** %s\n"+
			"**Provider:** %s\n"+
			"**Expertise:** %s\n\n"+
			"Mention with `@%s` to ask questions.",
			expertType, name, agentType, providerDisplay, expertiseStr, name)), nil
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
		"**Expert Agents:**\n" +
		"• `/create-expert <type> [name] [provider] [model]` - Create a specialist agent (rust, backend, frontend, devops, database, security)\n\n" +
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
	msg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{
			ID:   "system",
			Name: "System",
			Type: protocol.AgentTypeGeneral,
		},
		content,
	)
	msg.Mentions = []string{}
	return msg
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

// handleOpenFile validates and resolves a file path for opening in the editor.
func (ch *CommandHandler) handleOpenFile(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /open-file <file-path>"), nil
	}

	filePath := strings.Join(parts[1:], " ")
	resolved := filePath

	if !filepath.IsAbs(resolved) {
		if wm := ch.hub.GetWorkspaceManager(); wm != nil {
			workspaces := wm.ListWorkspaces()
			if len(workspaces) > 0 {
				resolved = filepath.Join(workspaces[0].Path, resolved)
			}
		}
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to resolve file path: %v", err)), nil
	}
	if _, err := os.Stat(absPath); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ File not found: %s", absPath)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📂 Opening file: %s", absPath)), nil
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

	wm := ch.hub.GetWorkspaceManager()
	if wm == nil {
		return ch.systemResponse(msg.Channel, "❌ Workspace manager is not available"), nil
	}
	ws, err := wm.AddWorkspace(name, path)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to add workspace: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("📁 Added workspace: %s\nPath: %s", ws.Name, ws.Path)), nil
}

// handleListWorkspaces lists all workspaces
func (ch *CommandHandler) handleListWorkspaces(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	wm := ch.hub.GetWorkspaceManager()
	if wm == nil {
		return ch.systemResponse(msg.Channel, "❌ Workspace manager is not available"), nil
	}
	workspaces := wm.ListWorkspaces()
	if len(workspaces) == 0 {
		return ch.systemResponse(msg.Channel, "📁 No workspaces configured"), nil
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return strings.ToLower(workspaces[i].Name) < strings.ToLower(workspaces[j].Name)
	})

	var b strings.Builder
	b.WriteString("📁 Workspaces:\n")
	for _, ws := range workspaces {
		b.WriteString(fmt.Sprintf("• %s (%s)\n", ws.Name, ws.Path))
	}
	return ch.systemResponse(msg.Channel, strings.TrimSpace(b.String())), nil
}

// Assistant command handlers

// handleReminder handles reminder-related commands
func (ch *CommandHandler) handleReminder(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel,
			"❌ Usage: `/remind <time> <message>` or `/remind-recurring <schedule> <message>`\n"+
				"Examples:\n"+
				"• `/remind in 30 minutes Review the PR`\n"+
				"• `/remind in 30s check the deploy`\n"+
				"• `/remind at 3pm Standup meeting`\n"+
				"• `/remind-recurring daily 9am Daily standup`"), nil
	}

	command := parts[0]
	rest := strings.TrimSpace(strings.TrimPrefix(msg.Content, command))

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	if command == "/remind-recurring" {
		schedule, reminderText, err := splitRecurringArgs(rest)
		if err != nil {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/remind-recurring <schedule> <message>`\nExample: `/remind-recurring daily 9am Daily standup`"), nil
		}
		recurring, triggerTime, err := parseRecurringSchedule(schedule)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid recurring schedule: %v", err)), nil
		}
		reminder := &agent.Reminder{
			ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
			Content:     reminderText,
			TriggerTime: triggerTime,
			Recurring:   recurring,
			Channel:     msg.Channel,
			CreatedBy:   msg.From.Name,
			Active:      true,
			CreatedAt:   time.Now(),
		}
		if err := storage.SaveReminder(reminder); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save recurring reminder: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("⏰ Recurring reminder set: '%s' (%s)", reminderText, schedule)), nil
	}

	timeExpr, reminderText, err := splitOneTimeReminderArgs(rest)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/remind <time> <message>`\nExamples: `/remind in 30m Review PR`, `/remind in 30s check logs`, `/remind at 3pm Standup`"), nil
	}

	triggerTime, err := parseReminderTimeExpression(timeExpr)
	if err != nil {
		if assistant := ch.findAssistantAgent(); assistant != nil {
			parsed, parseErr := assistant.ParseTime(timeExpr)
			if parseErr == nil {
				triggerTime = parsed
			} else {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid reminder time: %v", err)), nil
			}
		} else {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid reminder time: %v", err)), nil
		}
	}

	reminder := &agent.Reminder{
		ID:          fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Content:     reminderText,
		TriggerTime: triggerTime,
		Channel:     msg.Channel,
		CreatedBy:   msg.From.Name,
		Active:      true,
		CreatedAt:   time.Now(),
	}
	if err := storage.SaveReminder(reminder); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save reminder: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("⏰ Reminder set: '%s' at %s", reminderText, triggerTime.Format(time.RFC1123))), nil
}

// handleTask handles task-related commands
func (ch *CommandHandler) handleTask(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	switch command {
	case "/task-add":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-add <title>`"), nil
		}
		title := strings.Join(parts[1:], " ")
		task := &agent.Task{
			ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
			Title:     title,
			Priority:  3,
			Status:    "todo",
			CreatedAt: time.Now(),
			Channel:   msg.Channel,
			CreatedBy: msg.From.Name,
		}
		if err := storage.SaveTask(task); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save task: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Task added: [%s] %s", shortID(task.ID), title)), nil

	case "/task-list":
		tasks, err := storage.LoadTasks()
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to load tasks: %v", err)), nil
		}
		var pending []*agent.Task
		var done []*agent.Task
		for _, task := range tasks {
			if task.Channel != msg.Channel {
				continue
			}
			if task.Status == "done" {
				done = append(done, task)
			} else {
				pending = append(pending, task)
			}
		}
		if len(pending) == 0 && len(done) == 0 {
			return ch.systemResponse(msg.Channel, "📋 Task List:\nNo tasks found in this channel."), nil
		}
		sort.Slice(pending, func(i, j int) bool { return pending[i].CreatedAt.After(pending[j].CreatedAt) })
		sort.Slice(done, func(i, j int) bool { return done[i].CreatedAt.After(done[j].CreatedAt) })
		var b strings.Builder
		b.WriteString("📋 Task List:\n")
		if len(pending) > 0 {
			b.WriteString("Pending:\n")
			for i, task := range pending {
				b.WriteString(fmt.Sprintf("%d. [%s] %s (priority: %d)\n", i+1, shortID(task.ID), task.Title, task.Priority))
			}
		}
		if len(done) > 0 {
			if len(pending) > 0 {
				b.WriteString("\n")
			}
			b.WriteString("Done:\n")
			for i, task := range done {
				b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, shortID(task.ID), task.Title))
			}
		}
		return ch.systemResponse(msg.Channel, strings.TrimSpace(b.String())), nil

	case "/task-done":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/task-done <task-id>`"), nil
		}
		taskID := parts[1]
		tasks, err := storage.LoadTasks()
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to load tasks: %v", err)), nil
		}
		var matched *agent.Task
		for _, task := range tasks {
			if task.Channel != msg.Channel {
				continue
			}
			if task.ID == taskID || strings.HasPrefix(task.ID, taskID) || shortID(task.ID) == taskID {
				matched = task
				break
			}
		}
		if matched == nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Task '%s' not found in this channel", taskID)), nil
		}
		if matched.Status == "done" {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("ℹ️ Task '%s' is already marked done", shortID(matched.ID))), nil
		}
		matched.Status = "done"
		if err := storage.SaveTask(matched); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to update task: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("✅ Task '%s' marked as done", shortID(matched.ID))), nil

	default:
		return ch.systemResponse(msg.Channel, "❌ Unknown task command"), nil
	}
}

// handleNote handles note-related commands
func (ch *CommandHandler) handleNote(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	command := parts[0]
	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}

	switch command {
	case "/note-save":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-save <content>`"), nil
		}
		content := strings.Join(parts[1:], " ")
		note := &agent.Note{
			ID:        fmt.Sprintf("note_%d", time.Now().UnixNano()),
			Content:   content,
			Tags:      []string{},
			Channel:   msg.Channel,
			CreatedAt: time.Now(),
			CreatedBy: msg.From.Name,
		}
		if err := storage.SaveNote(note); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save note: %v", err)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("📝 Note saved: [%s] %s", shortID(note.ID), content)), nil

	case "/note-search":
		if len(parts) < 2 {
			return ch.systemResponse(msg.Channel, "❌ Usage: `/note-search <query>`"), nil
		}
		query := strings.Join(parts[1:], " ")
		results, err := storage.SearchNotes(query)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to search notes: %v", err)), nil
		}
		var b strings.Builder
		count := 0
		for _, note := range results {
			if note.Channel != msg.Channel {
				continue
			}
			count++
			b.WriteString(fmt.Sprintf("• [%s] %s\n", shortID(note.ID), note.Content))
		}
		if count == 0 {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("🔍 No notes found for '%s' in this channel", query)), nil
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("🔍 Found %d note(s) for '%s':\n%s", count, query, strings.TrimSpace(b.String()))), nil

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

	rest := strings.TrimSpace(strings.TrimPrefix(msg.Content, "/meeting-add"))
	timeExpr, title, err := splitOneTimeReminderArgs(rest)
	if err != nil {
		return ch.systemResponse(msg.Channel, "❌ Usage: `/meeting-add <time> <title>`\nExample: `/meeting-add tomorrow 2pm Team standup`"), nil
	}

	startTime, err := parseReminderTimeExpression(timeExpr)
	if err != nil {
		if assistant := ch.findAssistantAgent(); assistant != nil {
			parsed, parseErr := assistant.ParseTime(timeExpr)
			if parseErr == nil {
				startTime = parsed
			} else {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid meeting time: %v", err)), nil
			}
		} else {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid meeting time: %v", err)), nil
		}
	}

	storage, err := agent.NewAssistantStorage()
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to initialize assistant storage: %v", err)), nil
	}
	meeting := &agent.Meeting{
		ID:        fmt.Sprintf("meeting_%d", time.Now().UnixNano()),
		Title:     title,
		StartTime: startTime,
		Channel:   msg.Channel,
		CreatedBy: msg.From.Name,
		CreatedAt: time.Now(),
	}
	if err := storage.SaveMeeting(meeting); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to save meeting: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("📅 Meeting added: '%s' at %s", title, startTime.Format(time.RFC1123))), nil
}

// handleSummarize handles conversation summarization
func (ch *CommandHandler) handleSummarize(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	limit := 20
	if len(parts) > 1 {
		arg := strings.ToLower(parts[1])
		if arg == "last" && len(parts) > 2 {
			arg = parts[2]
		}
		if n, err := strconv.Atoi(arg); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	// Get recent messages from the channel
	messages, err := ch.hub.GetMessages(msg.Channel, limit)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to get messages: %v", err)), nil
	}

	if len(messages) == 0 {
		return ch.systemResponse(msg.Channel, "❌ No messages to summarize"), nil
	}

	var transcript strings.Builder
	for _, m := range messages {
		if m.Type == protocol.MessageTypeSystemInfo || m.Type == protocol.MessageTypeAgentStatus {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(m.Content), "/") {
			continue
		}
		transcript.WriteString(fmt.Sprintf("[%s][%s] %s\n", m.Timestamp.Format("15:04"), m.From.Name, m.Content))
	}
	if transcript.Len() == 0 {
		return ch.systemResponse(msg.Channel, "❌ No non-command messages to summarize"), nil
	}

	prompt := fmt.Sprintf("Summarize this channel conversation into concise bullets with action items and decisions.\n\n%s", transcript.String())
	aiProvider := ch.aiProvider
	if assistant := ch.findAssistantAgent(); assistant != nil && assistant.AI != nil {
		aiProvider = assistant.AI
	}
	summary, err := aiProvider.GenerateResponse(ctx, prompt, nil)
	if err != nil {
		// Fallback to deterministic summary if AI call fails.
		lines := strings.Split(strings.TrimSpace(transcript.String()), "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("📄 Summary fallback (AI unavailable):\n• %s", strings.Join(lines, "\n• "))), nil
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("📄 Summary of last %d messages:\n%s", len(messages), summary)), nil
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
		"• `/summarize [count]` - Summarize recent channel conversation\n\n" +
		"**Examples:**\n" +
		"• `/remind in 30 minutes Review the PR`\n" +
		"• `/remind at 3pm Standup meeting`\n" +
		"• `/task-add Fix the bug in login`\n" +
		"• `/note-save Important: API key is abc123`\n" +
		"• `/meeting-add tomorrow 2pm Team standup`"

	return ch.systemResponse(msg.Channel, help), nil
}

func splitOneTimeReminderArgs(input string) (string, string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("not enough arguments")
	}
	maxTimeParts := 5
	if len(parts)-1 < maxTimeParts {
		maxTimeParts = len(parts) - 1
	}
	for i := 1; i <= maxTimeParts; i++ {
		candidate := strings.Join(parts[:i], " ")
		if _, err := parseReminderTimeExpression(candidate); err == nil {
			message := strings.TrimSpace(strings.Join(parts[i:], " "))
			if message != "" {
				return candidate, message, nil
			}
		}
	}
	return "", "", fmt.Errorf("unable to parse time expression")
}

func splitRecurringArgs(input string) (string, string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("not enough arguments")
	}
	scheduleType := strings.ToLower(parts[0])
	switch scheduleType {
	case "daily", "weekly", "monthly":
	default:
		return "", "", fmt.Errorf("unsupported schedule type")
	}

	if len(parts) >= 3 && likelyClockToken(parts[1]) {
		schedule := scheduleType + " " + parts[1]
		message := strings.TrimSpace(strings.Join(parts[2:], " "))
		if message == "" {
			return "", "", fmt.Errorf("missing reminder message")
		}
		return schedule, message, nil
	}

	message := strings.TrimSpace(strings.Join(parts[1:], " "))
	if message == "" {
		return "", "", fmt.Errorf("missing reminder message")
	}
	return scheduleType, message, nil
}

func parseRecurringSchedule(schedule string) (*agent.RecurringSchedule, time.Time, error) {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(schedule)))
	if len(parts) == 0 {
		return nil, time.Time{}, fmt.Errorf("empty schedule")
	}

	recurring := &agent.RecurringSchedule{
		Type:     parts[0],
		Interval: 1,
		Time:     "09:00",
	}

	now := time.Now()
	switch recurring.Type {
	case "daily":
		trigger := now.Add(24 * time.Hour)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	case "weekly":
		trigger := now.Add(7 * 24 * time.Hour)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	case "monthly":
		trigger := now.AddDate(0, 1, 0)
		if len(parts) > 1 {
			parsed, err := parseClockTime(parts[1], now)
			if err != nil {
				return nil, time.Time{}, err
			}
			trigger = parsed
			recurring.Time = parsed.Format("15:04")
		}
		return recurring, trigger, nil
	default:
		return nil, time.Time{}, fmt.Errorf("unsupported recurring type")
	}
}

func parseReminderTimeExpression(input string) (time.Time, error) {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	now := time.Now()

	relativeRe := regexp.MustCompile(`^(?:in\s+)?(\d+)\s*(s|sec|secs|second|seconds|m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)$`)
	if m := relativeRe.FindStringSubmatch(trimmed); len(m) == 3 {
		amount, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "s", "sec", "secs", "second", "seconds":
			return now.Add(time.Duration(amount) * time.Second), nil
		case "m", "min", "mins", "minute", "minutes":
			return now.Add(time.Duration(amount) * time.Minute), nil
		case "h", "hr", "hrs", "hour", "hours":
			return now.Add(time.Duration(amount) * time.Hour), nil
		case "d", "day", "days":
			return now.AddDate(0, 0, amount), nil
		}
	}

	if strings.HasPrefix(trimmed, "in ") {
		return parseReminderTimeExpression(strings.TrimSpace(strings.TrimPrefix(trimmed, "in ")))
	}

	if strings.HasPrefix(trimmed, "tomorrow") {
		clock := strings.TrimSpace(strings.TrimPrefix(trimmed, "tomorrow"))
		clock = strings.TrimSpace(strings.TrimPrefix(clock, "at"))
		if clock == "" {
			return now.AddDate(0, 0, 1), nil
		}
		tomorrow := now.AddDate(0, 0, 1)
		return parseClockTime(clock, tomorrow)
	}

	if strings.HasPrefix(trimmed, "at ") {
		return parseClockTime(strings.TrimSpace(strings.TrimPrefix(trimmed, "at ")), now)
	}

	return parseClockTime(trimmed, now)
}

func parseClockTime(timeExpr string, day time.Time) (time.Time, error) {
	layouts := []string{"15:04", "3:04pm", "3:04 pm", "3pm", "3 pm", "3:04PM", "3PM"}
	normalized := strings.ToLower(strings.TrimSpace(timeExpr))
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, normalized); err == nil {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location())
			// If user gave a clock time that's already passed today, schedule for tomorrow.
			if day.Year() == time.Now().Year() && day.YearDay() == time.Now().YearDay() && candidate.Before(time.Now()) {
				candidate = candidate.AddDate(0, 0, 1)
			}
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time expression")
}

func likelyClockToken(v string) bool {
	token := strings.ToLower(strings.TrimSpace(v))
	return strings.Contains(token, ":") || strings.HasSuffix(token, "am") || strings.HasSuffix(token, "pm")
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
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

	// Find the agent
	var targetAgent *protocol.AgentInfo
	for _, agent := range ch.hub.ListAgents() {
		if strings.EqualFold(agent.Name, agentName) {
			targetAgent = agent
			break
		}
	}

	if targetAgent == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
	}

	if _, err := ch.SwitchAgentProvider(targetAgent.ID, provider, model, msg.Channel, msg.Metadata); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to switch %s: %v", agentName, err)), nil
	}
	modelLabel := model
	if modelLabel == "" {
		modelLabel = "(default)"
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ %s switched to %s (%s)", agentName, provider, modelLabel)), nil
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

	switchedCount, err := ch.SwitchAllProviders(provider, model, msg.Channel, msg.Metadata)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to switch providers: %v", err)), nil
	}

	modelLabel := model
	if modelLabel == "" {
		modelLabel = "(default)"
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Switched %d agents to %s (%s)", switchedCount, provider, modelLabel)), nil
}

type providerSwitchableAgent interface {
	SetAIProvider(newProvider ai.AIProvider) error
	GetAgentInfo() protocol.AgentInfo
}

func (ch *CommandHandler) resolveRuntimeAgent(agentID string) providerSwitchableAgent {
	if runtimeAgent, ok := ch.runtimeAgents[agentID]; ok && runtimeAgent != nil {
		return runtimeAgent
	}
	if repoAgent, ok := ch.repoAgents[agentID]; ok && repoAgent != nil {
		return repoAgent
	}
	if helperAgent, ok := ch.helperAgents[agentID]; ok && helperAgent != nil {
		return helperAgent
	}
	if ch.assistantAgent != nil && ch.assistantAgent.Info.ID == agentID {
		return ch.assistantAgent.Agent
	}
	for _, confluenceAgent := range ch.confluenceAgents {
		if confluenceAgent != nil && confluenceAgent.Info.ID == agentID {
			return confluenceAgent
		}
	}
	for _, cliAgent := range ch.cliAgents {
		if cliAgent != nil && cliAgent.Info.ID == agentID {
			return cliAgent
		}
	}
	return nil
}

func defaultModelForProvider(provider string) string {
	switch provider {
	case "ollama":
		return "llama3.1"
	case "claude":
		return "claude-sonnet"
	case "lmstudio":
		return ""
	default:
		return ""
	}
}

func buildProviderForSwitch(provider, model string, metadata map[string]interface{}) (ai.AIProvider, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = defaultModelForProvider(provider)
	}

	switch provider {
	case "ollama":
		return ai.NewOllamaProviderWithConfig("", resolvedModel), resolvedModel, nil
	case "lmstudio":
		endpoint := ""
		if metadata != nil {
			if ep, ok := metadata["lm_studio_endpoint"].(string); ok {
				endpoint = ep
			}
		}
		return ai.NewLMStudioProviderWithConfig(endpoint, resolvedModel), resolvedModel, nil
	case "claude":
		var apiKey, aiHubEndpoint string
		useAIHub := false
		if metadata != nil {
			if key, ok := metadata["anthropic_api_key"].(string); ok {
				apiKey = key
			}
			if use, ok := metadata["use_ai_hub"].(bool); ok {
				useAIHub = use
			}
			if endpoint, ok := metadata["ai_hub_endpoint"].(string); ok {
				aiHubEndpoint = endpoint
			}
		}
		if apiKey != "" {
			return ai.NewClaudeProviderWithConfig(apiKey, useAIHub, aiHubEndpoint, resolvedModel), resolvedModel, nil
		}
		claudeProvider, err := ai.NewClaudeProvider()
		if err != nil {
			return nil, "", fmt.Errorf("failed to initialize claude provider: %w", err)
		}
		return claudeProvider, resolvedModel, nil
	default:
		return nil, "", fmt.Errorf("invalid provider %q (allowed: claude, ollama, lmstudio)", provider)
	}
}

// SwitchAgentProvider switches a single live agent's provider/model.
func (ch *CommandHandler) SwitchAgentProvider(agentID, provider, model, channel string, metadata map[string]interface{}) (*protocol.AgentInfo, error) {
	targetAgent, err := ch.hub.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found")
	}

	runtimeAgent := ch.resolveRuntimeAgent(agentID)
	if runtimeAgent == nil {
		return nil, fmt.Errorf("runtime instance for agent %q not found", targetAgent.Name)
	}

	newProvider, resolvedModel, err := buildProviderForSwitch(provider, model, metadata)
	if err != nil {
		return nil, err
	}

	if err := runtimeAgent.SetAIProvider(newProvider); err != nil {
		return nil, err
	}

	// Keep hub metadata in sync for list/detail APIs.
	targetAgent.AIProvider = strings.ToLower(provider)
	targetAgent.AIModel = resolvedModel
	targetAgent.Model = resolvedModel
	runtimeInfo := runtimeAgent.GetAgentInfo()
	targetAgent.ApprovalMode = runtimeInfo.ApprovalMode

	// Emit a status event in the caller's channel so UI updates immediately.
	broadcastChannel := channel
	if strings.TrimSpace(broadcastChannel) == "" {
		broadcastChannel = "general"
	}
	statusMsg := protocol.NewMessage(
		protocol.MessageTypeAgentStatus,
		broadcastChannel,
		*targetAgent,
		fmt.Sprintf("🔄 %s switched to %s (%s)", targetAgent.Name, targetAgent.AIProvider, targetAgent.AIModel),
	)
	statusMsg.Metadata = map[string]interface{}{
		"ai_provider": targetAgent.AIProvider,
		"ai_model":    targetAgent.AIModel,
		"model":       targetAgent.Model,
	}
	ch.hub.SendMessage(statusMsg)

	return targetAgent, nil
}

// SwitchAllProviders switches all currently registered live agents.
func (ch *CommandHandler) SwitchAllProviders(provider, model, channel string, metadata map[string]interface{}) (int, error) {
	agents := ch.hub.ListAgents()
	switchedCount := 0
	var failures []string

	for _, agentInfo := range agents {
		if _, err := ch.SwitchAgentProvider(agentInfo.ID, provider, model, channel, metadata); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", agentInfo.Name, err))
			continue
		}
		switchedCount++
	}

	if len(failures) > 0 {
		return switchedCount, fmt.Errorf(strings.Join(failures, "; "))
	}
	return switchedCount, nil
}

// ── Channel management commands ──────────────────────────────────────────

func (ch *CommandHandler) handleCreateChannelCmd(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /create-channel <name> [description]"), nil
	}
	name := strings.ToLower(parts[1])
	description := ""
	if len(parts) > 2 {
		description = strings.Join(parts[2:], " ")
	}

	channel := ch.hub.CreateChannelWithType(name, description, "", protocol.ChannelTypeCustom, msg.From.Name)
	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Created channel **#%s** (id: %s)", channel.Name, channel.ID)), nil
}

func (ch *CommandHandler) handleAddToChannel(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "Usage: /add-to-channel <channel> <agent-name>"), nil
	}
	channelName := parts[1]
	agentName := strings.Join(parts[2:], " ")

	// Find agent by name
	agents := ch.hub.ListAgents()
	var targetAgent *protocol.AgentInfo
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			targetAgent = a
			break
		}
	}
	if targetAgent == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Agent '%s' not found", agentName)), nil
	}

	if err := ch.hub.AddAgentToChannel(targetAgent.ID, channelName); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Failed to add agent: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Added **%s** to **#%s**", targetAgent.Name, channelName)), nil
}

func (ch *CommandHandler) handleRemoveFromChannel(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "Usage: /remove-from-channel <channel> <agent-name>"), nil
	}
	channelName := parts[1]
	agentName := strings.Join(parts[2:], " ")

	agents := ch.hub.ListAgents()
	var targetAgent *protocol.AgentInfo
	for _, a := range agents {
		if strings.EqualFold(a.Name, agentName) {
			targetAgent = a
			break
		}
	}
	if targetAgent == nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Agent '%s' not found", agentName)), nil
	}

	if err := ch.hub.RemoveAgentFromChannel(targetAgent.ID, channelName); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Failed to remove agent: %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Removed **%s** from **#%s**", targetAgent.Name, channelName)), nil
}

func (ch *CommandHandler) handleListChannels(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	channels := ch.hub.ListChannels()
	if len(channels) == 0 {
		return ch.systemResponse(msg.Channel, "No channels found."), nil
	}

	var sb strings.Builder
	sb.WriteString("**Channels:**\n")
	for _, c := range channels {
		typeLabel := string(c.Type)
		if typeLabel == "" {
			typeLabel = "public"
		}
		sb.WriteString(fmt.Sprintf("• **#%s** (%s) — %d agents", c.Name, typeLabel, len(c.Agents)))
		if c.Description != "" {
			sb.WriteString(fmt.Sprintf(" — %s", c.Description))
		}
		sb.WriteString("\n")
	}
	return ch.systemResponse(msg.Channel, sb.String()), nil
}

func (ch *CommandHandler) handleDeleteChannelCmd(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /delete-channel <name>"), nil
	}
	name := parts[1]

	if err := ch.hub.DeleteChannel(name); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Failed: %v", err)), nil
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Deleted channel **#%s**", name)), nil
}

func (ch *CommandHandler) handleCreateCLIAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		types := agent.ListCLIAgentTypes()
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("Usage: `/create-cli-agent <type> [name] [work-dir]`\n\n"+
				"**Available types:** %s\n\n"+
				"**Examples:**\n```\n"+
				"/create-cli-agent cursor\n"+
				"/create-cli-agent gemini MyGemini /path/to/project\n"+
				"/create-cli-agent claude ClaudeDev\n"+
				"```", strings.Join(types, ", "))), nil
	}

	cliType := strings.ToLower(parts[1])
	cfg, ok := agent.GetCLIAgentConfig(cliType)
	if !ok {
		types := agent.ListCLIAgentTypes()
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("Unknown CLI agent type '%s'.\n\nAvailable types: %s", cliType, strings.Join(types, ", "))), nil
	}

	// Resolve work directory
	workDir := ""
	if len(parts) >= 4 {
		workDir = parts[3]
	}
	if workDir == "" && cfg.WorkDirEnv != "" {
		workDir = os.Getenv(cfg.WorkDirEnv)
	}
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			workDir = "."
		}
	}

	// Create provider
	opts := []ai.CLIAgentOption{
		ai.WithBaseArgs(cfg.BaseArgs),
		ai.WithModel(cfg.ModelName),
	}
	provider := ai.NewCLIAgentProvider(cfg.Command, workDir, cfg.ProviderName, opts...)

	// Forward configured env vars
	for _, envKey := range cfg.EnvVars {
		if val := os.Getenv(envKey); val != "" {
			provider.Env[envKey] = val
		}
	}

	if !provider.IsCLIInstalled() {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("CLI binary `%s` not found on PATH.\n\n%s", cfg.Command, cfg.InstallHint)), nil
	}

	// Resolve name
	name := cfg.DefaultName
	if len(parts) >= 3 {
		name = parts[2]
	}
	name = protocol.NormalizeAgentName(name)

	// Check for duplicate
	for _, existing := range ch.hub.ListAgents() {
		if strings.EqualFold(existing.Name, name) {
			return ch.systemResponse(msg.Channel,
				fmt.Sprintf("Agent '%s' already exists. Use a different name or `/delete-agent %s` first.", name, name)), nil
		}
	}

	// Create agent from registry config
	agentInstance := agent.NewCLIAgentFromConfig(cfg, name, provider, ch.hub)
	agentInstance.SetCollabClient(ch.hub.NewCollaborationClientAdapter())

	if cfg.ApprovalMode != "" {
		agentInstance.Info.ApprovalMode = cfg.ApprovalMode
	}

	if err := ch.hub.RegisterAgent(&agentInstance.Info); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Failed to register agent: %v", err)), nil
	}

	if err := ch.hub.JoinChannel(agentInstance.Info.ID, msg.Channel); err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("Failed to join agent to channel: %v", err)), nil
	}

	go func() {
		if err := agentInstance.Start(ctx, msg.Channel); err != nil {
			log.Printf("Failed to start CLI agent %s: %v", name, err)
		}
	}()

	ch.cliAgents[name] = agentInstance
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance

	// Persist for My Agents panel
	agent.SaveCLIAgent(cliType, name, workDir)

	// Send join message
	joinMsg := protocol.NewMessage(
		protocol.MessageTypeAgentJoin,
		msg.Channel,
		agentInstance.Info,
		cfg.JoinMessage,
	)
	if err := ch.hub.SendMessage(joinMsg); err != nil {
		log.Printf("Failed to send CLI agent join message: %v", err)
	}

	expertiseStr := strings.Join(cfg.Expertise, ", ")
	if len(cfg.Expertise) > 5 {
		expertiseStr = strings.Join(cfg.Expertise[:5], ", ") + fmt.Sprintf(" and %d more", len(cfg.Expertise)-5)
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("Created **%s** CLI agent: **%s**\n\n"+
			"**Type:** %s\n"+
			"**Binary:** `%s`\n"+
			"**Work Dir:** %s\n"+
			"**Expertise:** %s\n\n"+
			"Mention with `@%s` to ask questions.",
			cliType, name, cfg.ProviderName, cfg.Command, workDir, expertiseStr, name)), nil
}

func (ch *CommandHandler) handleListCLIAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	types := agent.ListCLIAgentTypes()
	lines := []string{"**Available CLI Agent Types:**\n"}

	for _, t := range types {
		cfg, _ := agent.GetCLIAgentConfig(t)
		provider := ai.NewCLIAgentProvider(cfg.Command, ".", cfg.ProviderName)
		installed := provider.IsCLIInstalled()

		status := "not installed"
		if installed {
			status = "installed"
		}

		lines = append(lines, fmt.Sprintf("- **%s** (`%s`) -- %s\n  %s",
			t, cfg.Command, status, cfg.InstallHint))
	}

	// Show currently running CLI agents
	if len(ch.cliAgents) > 0 {
		lines = append(lines, "\n**Running CLI Agents:**\n")
		for name, a := range ch.cliAgents {
			lines = append(lines, fmt.Sprintf("- **%s** (%s)", name, a.Info.AIProvider))
		}
	}

	return ch.systemResponse(msg.Channel, strings.Join(lines, "\n")), nil
}

func (ch *CommandHandler) handleOpenTerminal(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	cwd := ""
	if len(parts) >= 2 {
		cwd = parts[1]
	}
	agentName := msg.From.Name
	if msg.From.ID != "" && msg.From.ID != "system" {
		if a, err := ch.hub.GetAgent(msg.From.ID); err == nil {
			agentName = a.Name
		}
	}

	sysMsg := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"",
	)
	sysMsg.Metadata = map[string]interface{}{
		"event":      "agent-open-terminal",
		"agent_name": agentName,
		"cwd":        cwd,
	}
	ch.hub.BroadcastDirect(msg.Channel, sysMsg)

	label := "terminal tab"
	if cwd != "" {
		label = fmt.Sprintf("terminal tab at %s", cwd)
	}
	return ch.systemResponse(msg.Channel, fmt.Sprintf("Opening %s for **%s**", label, agentName)), nil
}

// SetAssistantAgent sets the assistant agent reference for meeting notes functionality
func (ch *CommandHandler) SetAssistantAgent(assistant *agent.AssistantAgent) {
	ch.assistantAgent = assistant
}

// RegisterRuntimeAgent tracks server-created runtime agents so collaboration
// wiring can reliably reach specialists/moderator/assistant and startup CLIs.
func (ch *CommandHandler) RegisterRuntimeAgent(agentInstance *agent.Agent) {
	if agentInstance == nil {
		return
	}
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance
}

// Ensure CommandHandler implements CommandHandlerInterface
var _ agent.CommandHandlerInterface = (*CommandHandler)(nil)

// GetCommandDefinitions returns metadata for every registered slash command.
func (ch *CommandHandler) GetCommandDefinitions() []protocol.CommandDefinition {
	return ch.buildCommandDefinitions()
}

func (ch *CommandHandler) buildCommandDefinitions() []protocol.CommandDefinition {
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

		// ── Expert Agents ─────────────────────────────────────────────
		{
			Name:        "/create-expert",
			Description: "Create a specialist agent (rust, backend, frontend, devops, database, security)",
			Category:    "Expert Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "type", Description: "Expert type (rust, backend, frontend, devops, database, security)", Type: "string", Required: true, Options: []string{"rust", "backend", "frontend", "devops", "database", "security"}},
				{Name: "name", Description: "Custom name for the agent", Type: "string", Required: false},
				{Name: "provider", Description: "AI provider", Type: "provider", Required: false, Options: providerOpts, Default: "ollama"},
				{Name: "model", Description: "AI model name", Type: "model", Required: false},
			},
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
			Name:        "/remind",
			Description: "Set a one-time reminder",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "time", Description: "Reminder time (e.g. in 30m, at 3pm)", Type: "string", Required: true},
				{Name: "message", Description: "Reminder content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/remind-recurring",
			Description: "Set a recurring reminder",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "schedule", Description: "daily|weekly|monthly", Type: "string", Required: true},
				{Name: "message", Description: "Reminder content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/task-add",
			Description: "Add a task",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "title", Description: "Task title", Type: "string", Required: true},
			},
		},
		{
			Name:        "/task-list",
			Description: "List tasks in this channel",
			Category:    "Assistant",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/task-done",
			Description: "Mark a task as complete",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "task-id", Description: "Task id or short id prefix", Type: "string", Required: true},
			},
		},
		{
			Name:        "/note-save",
			Description: "Save a note",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "content", Description: "Note content", Type: "string", Required: true},
			},
		},
		{
			Name:        "/note-search",
			Description: "Search saved notes",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "query", Description: "Search query", Type: "string", Required: true},
			},
		},
		{
			Name:        "/meeting-add",
			Description: "Add a meeting to schedule",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "time", Description: "Start time (e.g. tomorrow 2pm)", Type: "string", Required: true},
				{Name: "title", Description: "Meeting title", Type: "string", Required: true},
			},
		},
		{
			Name:        "/summarize",
			Description: "Summarize recent channel messages",
			Category:    "Assistant",
			Arguments: []protocol.CommandArgument{
				{Name: "count", Description: "Optional message count, e.g. 10", Type: "string", Required: false},
			},
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

		// ── Channels ───────────────────────────────────────────────────
		{
			Name:        "/create-channel",
			Description: "Create a custom channel with optional description",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "name", Description: "Channel name (slug)", Type: "string", Required: true},
				{Name: "description", Description: "Channel description", Type: "string", Required: false},
			},
		},
		{
			Name:        "/add-to-channel",
			Description: "Add an agent to a channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "channel", Description: "Channel name", Type: "string", Required: true},
				{Name: "agent-name", Description: "Agent to add", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/remove-from-channel",
			Description: "Remove an agent from a channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "channel", Description: "Channel name", Type: "string", Required: true},
				{Name: "agent-name", Description: "Agent to remove", Type: "agent-name", Required: true},
			},
		},
		{
			Name:        "/list-channels",
			Description: "List all channels with member counts",
			Category:    "Channels",
			Arguments:   []protocol.CommandArgument{},
		},
		{
			Name:        "/delete-channel",
			Description: "Delete a custom or DM channel",
			Category:    "Channels",
			Arguments: []protocol.CommandArgument{
				{Name: "name", Description: "Channel name to delete", Type: "string", Required: true},
			},
		},

		// ── Terminal ───────────────────────────────────────────────────
		{
			Name:        "/open-terminal",
			Description: "Open a new terminal tab (optionally in a given directory)",
			Category:    "Terminal",
			Arguments: []protocol.CommandArgument{
				{Name: "cwd", Description: "Working directory for the terminal", Type: "path", Required: false},
			},
		},

		// ── CLI Agents ────────────────────────────────────────────────
		{
			Name:        "/create-cli-agent",
			Description: "Create a CLI proxy agent (Cursor, Gemini, Claude, Copilot)",
			Category:    "CLI Agents",
			Arguments: []protocol.CommandArgument{
				{Name: "type", Description: "CLI agent type", Type: "string", Required: true, Options: agent.ListCLIAgentTypes()},
				{Name: "name", Description: "Custom agent name", Type: "string", Required: false},
				{Name: "work-dir", Description: "Working directory for the CLI", Type: "path", Required: false},
			},
		},
		{
			Name:        "/list-cli-agents",
			Description: "List available CLI agent types and their install status",
			Category:    "CLI Agents",
			Arguments:   []protocol.CommandArgument{},
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

		// ── Collaboration ─────────────────────────────────────────────
		{
			Name:        "/collaborate",
			Description: "Start a multi-agent collaboration. Mention 2+ agents followed by a description.",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "description", Description: "@Agent1 @Agent2 ... description of what to build", Type: "string", Required: true},
			},
		},
		{
			Name:        "/approve-plan",
			Description: "Approve a collaboration plan and begin execution",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (first 8 chars is enough)", Type: "string", Required: true},
			},
		},
		{
			Name:        "/revise-plan",
			Description: "Send feedback to revise a collaboration plan",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID", Type: "string", Required: true},
				{Name: "feedback", Description: "Revision feedback for the agents", Type: "string", Required: true},
			},
		},
		{
			Name:        "/cancel-plan",
			Description: "Cancel an active collaboration",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID", Type: "string", Required: true},
			},
		},
		{
			Name:        "/collab-status",
			Description: "Show status of active collaborations",
			Category:    "Collaboration",
			Arguments: []protocol.CommandArgument{
				{Name: "collab-id", Description: "Collaboration ID (optional, shows all if omitted)", Type: "string", Required: false},
			},
		},
	}
}

func (ch *CommandHandler) validateCommandDefinitions() {
	executors := ch.commandExecutors()
	defs := ch.buildCommandDefinitions()

	defSet := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		defSet[strings.ToLower(def.Name)] = struct{}{}
	}

	var missingInDefs []string
	for name := range executors {
		if _, ok := defSet[name]; !ok {
			missingInDefs = append(missingInDefs, name)
		}
	}

	execSet := make(map[string]struct{}, len(executors))
	for name := range executors {
		execSet[name] = struct{}{}
	}

	var missingInExecutors []string
	for _, def := range defs {
		name := strings.ToLower(def.Name)
		if _, ok := execSet[name]; !ok {
			missingInExecutors = append(missingInExecutors, name)
		}
	}

	sort.Strings(missingInDefs)
	sort.Strings(missingInExecutors)

	if len(missingInDefs) > 0 {
		log.Printf("⚠️  Command parity mismatch: command handlers missing from definitions: %s", strings.Join(missingInDefs, ", "))
	}
	if len(missingInExecutors) > 0 {
		log.Printf("⚠️  Command parity mismatch: command definitions missing handlers: %s", strings.Join(missingInExecutors, ", "))
	}
}

// ── Collaboration Command Handlers ──────────────────────────────────

// handleCollaborate starts a multi-agent collaboration.
// Usage: /collaborate @Agent1 @Agent2 @Agent3 build a CLI tool that encrypts files
func (ch *CommandHandler) handleCollaborate(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /collaborate @Agent1 @Agent2 ... description\nAt least 2 agents and a description are required."), nil
	}

	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ch.systemResponse(msg.Channel, "❌ Collaboration manager is not available."), nil
	}

	// Parse agent mentions and description
	mentionStrings := protocol.ParseMentions(strings.Join(parts[1:], " "))
	if len(mentionStrings) < 2 {
		return ch.systemResponse(msg.Channel, "❌ At least 2 agents must be @mentioned.\nUsage: /collaborate @Agent1 @Agent2 description"), nil
	}

	// Resolve mentions to agent IDs
	resolved := make(map[string]bool)
	agentIDs := ch.hub.ResolveMentionsWithValidation(mentionStrings, resolved)
	if len(agentIDs) < 2 {
		unresolved := []string{}
		for _, m := range mentionStrings {
			if !resolved[m] {
				unresolved = append(unresolved, "@"+m)
			}
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Could not resolve enough agents. Unresolved: %s\nAvailable agents: %s",
			strings.Join(unresolved, ", "), ch.hub.getAgentListString())), nil
	}

	// Extract description (everything after mentions)
	description := strings.Join(parts[1:], " ")
	for _, m := range mentionStrings {
		description = strings.Replace(description, "@"+m, "", 1)
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return ch.systemResponse(msg.Channel, "❌ A description is required after the agent mentions."), nil
	}

	collab, err := cm.CreateCollaboration(description, agentIDs, msg.Channel, msg.From.Name, collaboration.DiscussionConfig{})
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create collaboration: %v", err)), nil
	}

	// Ensure all collaboration participants are actually joined to the channel
	// where the collaboration is happening so they can subscribe/respond.
	for _, participantID := range agentIDs {
		if err := ch.hub.AddAgentToChannel(participantID, msg.Channel); err != nil {
			log.Printf("[Collaboration] Warning: failed to add participant %s to channel %s: %v", participantID, msg.Channel, err)
		}
	}

	// Build agent list for display
	var agentListStr strings.Builder
	for i, a := range collab.Agents {
		if i > 0 {
			agentListStr.WriteString(", ")
		}
		agentListStr.WriteString(fmt.Sprintf("**@%s** (%s)", a.AgentName, a.Role))
	}

	// Send the seed message to kick off the planning discussion
	seedMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("🤝 **Collaboration Started** (ID: `%s`)\n\n**Goal:** %s\n\n**Participants:** %s\n\n**Phase:** Planning (agents will discuss and propose a plan)\n\nAgents, please discuss and create a structured plan with tasks assigned to the agent best suited for each task. Use `- Task N: @AgentName - description` format for tasks.",
			collab.ID[:8], description, agentListStr.String()),
	)
	seedMsg.SetCollaborationID(collab.ID)
	seedMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	inheritWorkspaceContextMetadata(msg, seedMsg)

	if err := ch.hub.SendMessage(seedMsg); err != nil {
		log.Printf("[Collaboration] Failed to send seed message: %v", err)
	}

	// Set the Collab field on participating agents so they can check collaboration state
	collabClient := ch.hub.NewCollaborationClientAdapter()
	for _, a := range collab.Agents {
		ch.setCollabClientOnAgent(a.AgentID, a.AgentName, collabClient)
	}

	// Send the first turn prompt to the first agent
	firstAgent := collab.Agents[0]
	turnMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("@%s -- You're up first. Please share your initial thoughts on how to approach: %s\n\nConsider the strengths of each participant and propose initial task assignments.",
			firstAgent.AgentName, description),
	)
	turnMsg.SetCollaborationID(collab.ID)
	turnMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	turnMsg.Mentions = []string{firstAgent.AgentID}
	inheritWorkspaceContextMetadata(msg, turnMsg)

	if err := ch.hub.SendMessage(turnMsg); err != nil {
		log.Printf("[Collaboration] Failed to send first turn message: %v", err)
	}

	return nil, nil
}

// setCollabClientOnAgent sets the CollaborationClient on any agent type
// that embeds the base Agent struct. It searches known agent registries.
func (ch *CommandHandler) setCollabClientOnAgent(agentID, agentName string, client agent.CollaborationClient) {
	if runtimeAgent, ok := ch.runtimeAgents[agentID]; ok && runtimeAgent != nil {
		runtimeAgent.SetCollabClient(client)
		return
	}
	if ch.assistantAgent != nil && ch.assistantAgent.Info.ID == agentID {
		ch.assistantAgent.SetCollabClient(client)
		return
	}
	for _, ra := range ch.repoAgents {
		if ra.GetAgentInfo().ID == agentID {
			ra.SetCollabClient(client)
			return
		}
	}
	for _, ha := range ch.helperAgents {
		if ha.GetAgentInfo().ID == agentID {
			ha.SetCollabClient(client)
			return
		}
	}
	for _, ca := range ch.cliAgents {
		if ca.Info.ID == agentID {
			ca.SetCollabClient(client)
			return
		}
	}
	// System agents (specialist, moderator, assistant) are tracked differently;
	// we set the field via the hub's registered agent lookup.
	log.Printf("[Collaboration] Setting collab client on agent %s (%s) via hub lookup", agentName, shortID(agentID))
}

func (ch *CommandHandler) handleApprovePlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /approve-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	collab, err := cm.ApprovePlan(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	if len(collab.Tasks) == 0 && collab.Plan != nil && strings.TrimSpace(collab.Plan.Content) != "" {
		extractedTasks := collaboration.ExtractTasksFromPlan(collab.Plan.Content, collab.Agents)
		if len(extractedTasks) > 0 {
			if err := cm.SetTasks(collabID, extractedTasks); err != nil {
				log.Printf("[Collaboration] Failed to set extracted tasks for %s: %v", collabID[:8], err)
			}
		}
	}

	// Transition to executing
	collab, err = cm.TransitionToExecuting(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start execution: %v", err)), nil
	}

	// Notify agents about their assigned tasks
	var taskSummary strings.Builder
	taskSummary.WriteString(fmt.Sprintf("✅ **Plan Approved** (Collaboration `%s`)\n\n", collabID[:8]))
	taskSummary.WriteString("**Assigned Tasks:**\n\n")

	for i, task := range collab.Tasks {
		status := "⬜"
		taskSummary.WriteString(fmt.Sprintf("%s **Task %d:** %s\n   Assigned to: **@%s**\n\n", status, i+1, task.Description, task.AssignedName))

		// Send individual task messages to each assigned agent
		taskMsg := protocol.NewMessage(
			protocol.MessageTypeCollabTask,
			msg.Channel,
			protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
			fmt.Sprintf("@%s -- Your assigned task:\n\n**%s**\n\n%s\n\nPlease complete this task. You can @mention other collaboration participants if you need their input.",
				task.AssignedName, task.Title, task.Description),
		)
		taskMsg.SetCollaborationID(collabID)
		taskMsg.SetCollaborationPhase(string(collaboration.PhaseExecuting))
		taskMsg.SetTaskID(task.ID)
		taskMsg.SetTaskStatus(string(collaboration.TaskPending))
		inheritWorkspaceContextMetadata(msg, taskMsg)

		if task.AssignedTo != "" {
			taskMsg.Mentions = []string{task.AssignedTo}
		}

		if err := ch.hub.SendMessage(taskMsg); err != nil {
			log.Printf("[Collaboration] Failed to send task message: %v", err)
		}
	}

	return ch.systemResponse(msg.Channel, taskSummary.String()), nil
}

func (ch *CommandHandler) handleRevisePlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 3 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /revise-plan <collab-id> <feedback>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	feedback := strings.Join(parts[2:], " ")

	collab, err := cm.RevisePlan(collabID, feedback)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	// Send feedback to the collaboration channel
	revisionMsg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		msg.Channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		fmt.Sprintf("📝 **Plan Revision Requested** (Collaboration `%s`)\n\n**Feedback:** %s\n\nAgents, please revise the plan based on this feedback.",
			collabID[:8], feedback),
	)
	revisionMsg.SetCollaborationID(collabID)
	revisionMsg.SetCollaborationPhase(string(collaboration.PhasePlanning))
	inheritWorkspaceContextMetadata(msg, revisionMsg)

	// Mention all agents to notify them
	for _, a := range collab.Agents {
		revisionMsg.Mentions = append(revisionMsg.Mentions, a.AgentID)
	}

	if err := ch.hub.SendMessage(revisionMsg); err != nil {
		log.Printf("[Collaboration] Failed to send revision message: %v", err)
	}

	return nil, nil
}

func (ch *CommandHandler) handleCancelPlan(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "❌ Usage: /cancel-plan <collab-id>"), nil
	}

	cm := ch.hub.GetCollaborationManager()
	collabID := ch.resolveCollabID(parts[1])
	if collabID == "" {
		return ch.systemResponse(msg.Channel, "❌ Collaboration not found. Use /collab-status to see active collaborations."), nil
	}

	_, err := cm.CancelCollaboration(collabID)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("🛑 **Collaboration Cancelled** (`%s`)", collabID[:8])), nil
}

func inheritWorkspaceContextMetadata(src, dst *protocol.Message) {
	if src == nil || dst == nil || src.Metadata == nil {
		return
	}
	rawCtx, ok := src.Metadata["workspace_context"]
	if !ok {
		return
	}
	ctxMap, ok := rawCtx.(map[string]interface{})
	if !ok {
		return
	}
	safeCtx := map[string]interface{}{}
	if workspaceName, ok := ctxMap["workspace_name"].(string); ok {
		safeCtx["workspace_name"] = workspaceName
	}
	if workspacePath, ok := ctxMap["workspace_path"].(string); ok {
		safeCtx["workspace_path"] = workspacePath
	}
	if fileTree, ok := ctxMap["file_tree"].(string); ok {
		if len(fileTree) > 12000 {
			fileTree = fileTree[:12000] + "\n... (truncated)"
		}
		safeCtx["file_tree"] = fileTree
	}
	if openFiles, ok := ctxMap["open_files"].([]interface{}); ok {
		trimmedFiles := make([]map[string]interface{}, 0, len(openFiles))
		for _, entry := range openFiles {
			fileMeta, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			trimmed := map[string]interface{}{}
			if path, ok := fileMeta["path"].(string); ok {
				trimmed["path"] = path
			}
			if language, ok := fileMeta["language"].(string); ok {
				trimmed["language"] = language
			}
			if isActive, ok := fileMeta["is_active"].(bool); ok {
				trimmed["is_active"] = isActive
			}
			if len(trimmed) > 0 {
				trimmedFiles = append(trimmedFiles, trimmed)
			}
		}
		if len(trimmedFiles) > 0 {
			safeCtx["open_files"] = trimmedFiles
		}
	}
	if len(safeCtx) == 0 {
		return
	}
	if dst.Metadata == nil {
		dst.Metadata = map[string]interface{}{}
	}
	dst.Metadata["workspace_context"] = safeCtx
}

func (ch *CommandHandler) handleCollabStatus(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	cm := ch.hub.GetCollaborationManager()

	if len(parts) >= 2 {
		collabID := ch.resolveCollabID(parts[1])
		if collabID == "" {
			return ch.systemResponse(msg.Channel, "❌ Collaboration not found."), nil
		}
		collab, err := cm.GetCollaboration(collabID)
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
		}
		return ch.systemResponse(msg.Channel, ch.formatCollabDetail(collab)), nil
	}

	// List all active collaborations
	active := cm.ListActive()
	if len(active) == 0 {
		return ch.systemResponse(msg.Channel, "No active collaborations. Use `/collaborate @Agent1 @Agent2 description` to start one."), nil
	}

	var sb strings.Builder
	sb.WriteString("**Active Collaborations:**\n\n")
	for _, c := range active {
		agentNames := make([]string, 0, len(c.Agents))
		for _, a := range c.Agents {
			agentNames = append(agentNames, "@"+a.AgentName)
		}
		sb.WriteString(fmt.Sprintf("- `%s` | **%s** | Phase: %s | Agents: %s\n",
			c.ID[:8], c.Title, c.Phase, strings.Join(agentNames, ", ")))
	}
	return ch.systemResponse(msg.Channel, sb.String()), nil
}

func (ch *CommandHandler) formatCollabDetail(c *collaboration.Collaboration) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Collaboration: %s** (`%s`)\n\n", c.Title, c.ID[:8]))
	sb.WriteString(fmt.Sprintf("**Phase:** %s\n", c.Phase))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n", c.Description))
	sb.WriteString(fmt.Sprintf("**Created by:** %s\n", c.CreatedBy))
	sb.WriteString(fmt.Sprintf("**Created at:** %s\n\n", c.CreatedAt.Format(time.RFC822)))

	sb.WriteString("**Participants:**\n")
	for _, a := range c.Agents {
		sb.WriteString(fmt.Sprintf("- @%s (%s) -- %s\n", a.AgentName, a.AgentType, a.Role))
	}

	if c.Discussion != nil {
		sb.WriteString(fmt.Sprintf("\n**Discussion:** Round %d/%d | Messages: %d/%d | Status: %s\n",
			c.Discussion.CurrentRound, c.Discussion.MaxRounds,
			c.Discussion.TotalMessageCount, c.Discussion.MaxTotalMessages,
			c.Discussion.Status))
	}

	if len(c.Tasks) > 0 {
		sb.WriteString("\n**Tasks:**\n")
		for i, t := range c.Tasks {
			icon := "⬜"
			switch t.Status {
			case collaboration.TaskInProgress:
				icon = "🔄"
			case collaboration.TaskCompleted:
				icon = "✅"
			case collaboration.TaskBlocked:
				icon = "🚫"
			}
			sb.WriteString(fmt.Sprintf("%s **Task %d:** %s (assigned to @%s) - %s\n", icon, i+1, t.Title, t.AssignedName, t.Status))
		}
	}

	if c.Plan != nil && c.Plan.Content != "" {
		sb.WriteString(fmt.Sprintf("\n**Plan (v%d):**\n%s\n", c.Plan.Version, c.Plan.Content))
	}

	return sb.String()
}

// resolveCollabID accepts either a full UUID or a short prefix and
// returns the full collaboration ID if found.
func (ch *CommandHandler) resolveCollabID(input string) string {
	cm := ch.hub.GetCollaborationManager()
	if cm == nil {
		return ""
	}

	// Try exact match first
	if _, err := cm.GetCollaboration(input); err == nil {
		return input
	}

	// Try prefix match
	for _, c := range cm.ListActive() {
		if strings.HasPrefix(c.ID, input) {
			return c.ID
		}
	}

	return ""
}
