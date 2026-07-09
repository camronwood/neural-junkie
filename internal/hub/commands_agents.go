package hub

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

// handleCreateRepoAgent creates a new repository expert agent
func (ch *CommandHandler) handleCreateRepoAgent(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /create-repo-agent <repo-path> [agent-name] [provider] [model]\nFlags: --model <tag> --adapter-repo <hf-repo-id>\nProviders: ollama (default), claude, lmstudio, huggingface\nExample: /create-repo-agent /path/to/repo MyRepoExpert ollama qwen2.5-coder:14b"), nil
	}

	parts, flagModel, adapterRepo := parseCreateRepoAgentFlags(parts)
	repoPath := parts[1]
	agentName, provider, model := parseRepoAgentCreateArgs(parts, flagModel)

	if agentName == "" {
		agentName = defaultRepoAgentName(repoPath)
	}

	// Validate path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Invalid repository path: %v", err)), nil
	}

	if adapterRepo != "" {
		if ch.appConfig == nil || !ch.appConfig.AnyPackCapability("lora-compose") {
			return ch.systemResponse(msg.Channel, "❌ Specialist tuning pack required for --adapter-repo (Settings → Domain packs → Specialist tuning)"), nil
		}
	}
	if adapterRepo != "" && provider == "ollama" {
		composedTag, composeErr := ch.composeRepoLoRA(ctx, absPath, adapterRepo, model)
		if composeErr != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ LoRA compose failed: %v", composeErr)), nil
		}
		model = composedTag
	} else if model == "" && provider == "ollama" {
		model = config.DevOllamaCodeModel
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
			model = config.DevOllamaCodeModel
		}
		aiProvider = ai.NewOllamaProviderWithConfig("http://localhost:11434", model)
	} else if provider == "huggingface" || provider == "hf" {
		if model == "" {
			return ch.systemResponse(msg.Channel, "❌ huggingface provider requires a model (Hub repo id, e.g. Qwen/Qwen2.5-Coder-7B-Instruct)"), nil
		}
		token := ai.ResolveHFToken("")
		if token == "" {
			return ch.systemResponse(msg.Channel, "❌ HF token required: set HF_TOKEN or add a huggingface provider in Settings"), nil
		}
		aiProvider = ai.NewHuggingFaceProvider("", token, model)
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
		_, err := ch.hub.GetWorkspaceManager().AddWorkspace(workspaceName, absPath, AddWorkspaceOptions{Create: false})
		if err != nil {
			log.Printf("Warning: failed to auto-create workspace for repo agent: %v", err)
		} else {
			log.Printf("Auto-created workspace %q for repo agent at %s", workspaceName, absPath)
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

	if ch.appConfig != nil && ch.appConfig.AnyPackCapability("lora-training") && !isAutoCreated {
		statusMsg += "\n\n💡 After 10+ Q&A turns, open agent info → **Train LoRA** to bake sessions into a repo adapter."
	}

	resp := ch.systemResponse(msg.Channel, statusMsg)
	if resp.Metadata == nil {
		resp.Metadata = map[string]interface{}{}
	}
	resp.Metadata["client_action"] = map[string]interface{}{
		"type": "select_repo_workspace",
		"path": absPath,
		"name": agentName,
	}
	return resp, nil
}

// handleDeleteAgent deletes an agent

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
		cacheDeleted, err := DeleteCachedAgent("", agentName, "")
		if err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to delete cached agent: %v", err)), nil
		}
		if cacheDeleted {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Cached agent '%s' has been deleted", agentName)), nil
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
	}

	ch.agentsMu.Lock()
	defer ch.agentsMu.Unlock()

	// If it's a repo agent, clean up stored data
	if repoAgent, ok := ch.repoAgents[agentID]; ok {
		if err := repoAgent.Cleanup(); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("⚠️  Warning: Failed to cleanup agent data: %v", err)), nil
		}
		delete(ch.repoAgents, agentID)
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

	delete(ch.runtimeAgents, agentID)
	delete(ch.cliAgents, agentID)

	agent.DeleteExpertAgent(agentName)
	if cliStorage, err := agent.NewCLIAgentStorage(); err == nil {
		_, _ = cliStorage.DeleteByName(agentName)
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Agent '%s' has been deleted", agentName)), nil
}

// handleReindexAgent triggers a reindex of a repository agent

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

			// If it's a confluence agent we manage, pause it
			for _, confluenceAgent := range ch.confluenceAgents {
				if confluenceAgent.Info.ID == a.ID {
					confluenceAgent.Pause()
					break
				}
			}

			ch.AbortAgentGenerations(a.ID)

			return ch.systemResponse(msg.Channel, fmt.Sprintf("⏸️  Agent '%s' has been paused (in-flight generation stopped)", agentName)), nil
		}
	}

	return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found", agentName)), nil
}

// handleUnpauseAgent unpauses an agent

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
		response.WriteString("• `/create-expert <type> [name]` - Specialist agent (backend, frontend, architecture, code-review, assistant, ...)\n")
		response.WriteString("• `/create-confluence-agent <space>` - Confluence expert\n")
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleEnableWatch enables automatic file watching for a repo agent

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

// handleCreateExpert creates a specialist agent (preset or custom domain slug).

// handleCreateExpert creates a specialist agent (preset or custom domain slug).
func (ch *CommandHandler) handleCreateExpert(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel,
			"Usage: `/create-expert <type> [name ...] [provider] [model]`\n\n"+
				"**Preset types** (curated engineering specialists):\n"+
				"• `backend`, `frontend`, `devops`, `security`, `architecture`, `code-review`, `biology`, `cad`, `assistant`\n\n"+
				"• Legacy explicit slugs also work when the development pack is enabled: `rust`, `database`\n\n"+
				"**Custom experts:** use any other slug (e.g. `guitar`, `legal-advice`).\n\n"+
				"The expert is created in a **private DM** with you — it is **not** added to this channel. Invite the agent from the channel member UI when you want help here.\n\n"+
				"**Examples:**\n"+
				"```\n"+
				"/create-expert architecture\n"+
				"/create-expert code-review CodeReviewer\n"+
				"/create-expert guitar GuitarCoach\n"+
				"/create-expert backend BackendEngineer ollama qwen2.5-coder:14b\n"+
				"```\n\n"+
				"Use **spaces** between arguments (not commas)."), nil
	}

	parts = parseCreateExpertParts(parts)

	expertSlug, name, providerName, modelOverride := splitCreateExpertArgs(parts)

	if ch.appConfig != nil {
		if denied := ch.appConfig.PresetExpertDeniedMessage(expertSlug); denied != "" {
			return ch.systemResponse(msg.Channel, "❌ "+denied), nil
		}
	}

	spec, err := ResolveExpert(expertSlug, "")
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	name = strings.TrimSpace(name)
	if name == "" {
		if spec.IsPreset {
			defaults := map[protocol.AgentType]string{
				protocol.AgentTypeRust:         "RustExpert",
				protocol.AgentTypeBackend:      "BackendEngineer",
				protocol.AgentTypeFrontend:     "FrontendEngineer",
				protocol.AgentTypeDevOps:       "PlatformEngineer",
				protocol.AgentTypeDatabase:     "DatabaseSpecialist",
				protocol.AgentTypeSecurity:     "SecurityReviewer",
				protocol.AgentTypeArchitecture: "SoftwareArchitect",
				protocol.AgentTypeCodeReview:   "CodeReviewer",
				protocol.AgentTypeBiology:            "BiologyExpert",
				protocol.AgentTypeGenomics:           "GenomicsExpert",
				protocol.AgentTypeStructuralBiology:  "StructuralBiologyExpert",
				protocol.AgentTypeCheminformatics:    "ChemInformaticsExpert",
				protocol.AgentTypeAssistant:    "Assistant",
			}
			name = defaults[spec.AgentType]
		} else {
			name = strings.ReplaceAll(spec.Label, " ", "") + "Expert"
		}
	}

	agentInstance, err := ch.prepareExpertAgent(spec, name, "", providerName, modelOverride)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}

	channelDisplayName := strings.TrimSpace(name)
	if channelDisplayName == "" {
		channelDisplayName = agentInstance.Info.Name
	}

	createdBy := strings.TrimSpace(msg.From.Name)
	if createdBy == "" {
		_ = ch.hub.UnregisterAgent(agentInstance.Info.ID)
		delete(ch.runtimeAgents, agentInstance.Info.ID)
		return ch.systemResponse(msg.Channel,
			"❌ Cannot create expert: your display name is empty. Set your name in the app and try again."), nil
	}

	dmCh, err := ch.startExpertInDMOnly(agentInstance, createdBy, channelDisplayName)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ %v", err)), nil
	}
	ch.persistExpertAgentRecord(agentInstance, createdBy, expertSlug, "", "", providerName, modelOverride, dmCh)
	ch.setDMRedirect(dmCh.Name)

	expertiseStr := strings.Join(agentInstance.Info.Expertise, ", ")
	if len(agentInstance.Info.Expertise) > 5 {
		expertiseStr = strings.Join(agentInstance.Info.Expertise[:5], ", ") +
			fmt.Sprintf(" and %d more", len(agentInstance.Info.Expertise)-5)
	}

	providerDisplay := agentInstance.Info.AIModel
	if providerDisplay == "" {
		providerDisplay = agentInstance.Info.Model
	}
	if providerName != "" {
		providerDisplay = providerName + " / " + providerDisplay
	}

	name = agentInstance.Info.Name

	typeLabel := string(agentInstance.Info.Type)
	if !spec.IsPreset {
		typeLabel = "custom (" + spec.Label + ")"
	}

	return ch.systemResponse(msg.Channel,
		fmt.Sprintf("🤖 Created **%s** expert agent: **%s**\n\n"+
			"**Type:** %s\n"+
			"**Provider:** %s\n"+
			"**Expertise:** %s\n\n"+
			"**DM:** `%s` — chat there with `@%s` once setup finishes. To use them in this channel, invite **%s** from the member list.\n\n"+
			"_The agent is starting in the background — watch this channel for status, or open the DM from the sidebar when ready._",
			spec.Label, name, typeLabel, providerDisplay, expertiseStr, dmCh.Name, name, name)), nil
}

// handleHelp shows available commands

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
		var hint strings.Builder
		hint.WriteString(fmt.Sprintf(" To add them here: `/add-to-channel %s %s`", msg.Channel, agentName))
		if others := ch.hub.GetAgentChannels(agentID); len(others) > 0 {
			hint.WriteString(fmt.Sprintf(" (currently in: %s)", strings.Join(others, ", ")))
		}
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("❌ Agent '%s' is not in this channel.%s", agentName, hint.String())), nil
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
			// Check if this is a user-created agent (repo, confluence)
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

// findAssistantAgent finds the assistant agent in the hub
func (ch *CommandHandler) findAssistantAgent() *agent.AssistantAgent {
	return ch.assistantAgent
}

// minInt returns the minimum of two integers

func parseCreateRepoAgentFlags(parts []string) ([]string, string, string) {
	if len(parts) < 2 {
		return parts, "", ""
	}
	out := []string{parts[0], parts[1]}
	var flagModel, adapterRepo string
	for i := 2; i < len(parts); i++ {
		switch parts[i] {
		case "--model":
			if i+1 < len(parts) {
				flagModel = parts[i+1]
				i++
			}
		case "--adapter-repo":
			if i+1 < len(parts) {
				adapterRepo = parts[i+1]
				i++
			}
		default:
			out = append(out, parts[i])
		}
	}
	return out, flagModel, adapterRepo
}

func (ch *CommandHandler) composeRepoLoRA(ctx context.Context, repoPath, adapterRepo, model string) (string, error) {
	cacheDir := ""
	if ch.appConfig != nil {
		cacheDir = ch.appConfig.HF.CacheDir
	}
	mgr, err := hfhub.NewManager(cacheDir)
	if err != nil {
		return "", err
	}
	token := hfhub.TokenFromConfig(ch.appConfig)
	filename := "adapter_model.safetensors"
	if entry, err := hfhub.FindCatalogEntry(adapterRepo); err == nil && len(entry.Files) > 0 {
		filename = entry.Files[0].Filename
	}
	if err := hfhub.EnsureLoRAFiles(ctx, mgr, token, adapterRepo, filename); err != nil {
		return "", err
	}
	path, err := mgr.LocalPath(adapterRepo, filename)
	if err != nil {
		return "", err
	}
	tag := strings.TrimSpace(model)
	if tag == "" {
		tag = hfhub.RepoLoRATag(repoPath)
	}
	if err := hfhub.ImportAdapterToOllama(ctx, hfhub.DefaultLoRABaseTag, path, tag); err != nil {
		return "", err
	}
	return tag, nil
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

	resolved, found := agent.ResolveCLI(cfg)
	if !found {
		return ch.systemResponse(msg.Channel,
			fmt.Sprintf("CLI not found on PATH (tried: %s).\n\n%s", agent.CLIProbeLabel(cfg), cfg.InstallHint)), nil
	}

	// Create provider
	opts := []ai.CLIAgentOption{
		ai.WithBaseArgs(resolved.BaseArgs),
		ai.WithModel(cfg.ModelName),
	}
	provider := ai.NewCLIAgentProvider(resolved.Command, workDir, cfg.ProviderName, opts...)

	// Forward configured env vars
	for _, envKey := range cfg.EnvVars {
		if val := os.Getenv(envKey); val != "" {
			provider.Env[envKey] = val
		}
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

	ch.cliAgents[agentInstance.Info.ID] = agentInstance
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance

	// Persist for My Agents panel
	createdBy := strings.TrimSpace(msg.From.Name)
	agent.SaveCLIAgent(cliType, name, workDir, createdBy)

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
			cliType, name, cfg.ProviderName, resolved.Command, workDir, expertiseStr, name)), nil
}

func (ch *CommandHandler) handleListCLIAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	types := agent.ListCLIAgentTypes()
	lines := []string{"**Available CLI Agent Types:**\n"}

	for _, t := range types {
		cfg, _ := agent.GetCLIAgentConfig(t)
		resolved, found := agent.ResolveCLI(cfg)

		status := "not installed"
		binaryLabel := agent.CLIProbeLabel(cfg)
		if found {
			status = "installed"
			binaryLabel = resolved.Command
		}

		lines = append(lines, fmt.Sprintf("- **%s** (`%s`) -- %s\n  %s",
			t, binaryLabel, status, cfg.InstallHint))
	}

	// Show currently running CLI agents
	if len(ch.cliAgents) > 0 {
		lines = append(lines, "\n**Running CLI Agents:**\n")
		for _, a := range ch.cliAgents {
			if a == nil {
				continue
			}
			lines = append(lines, fmt.Sprintf("- **%s** (%s)", a.Info.Name, a.Info.AIProvider))
		}
	}

	return ch.systemResponse(msg.Channel, strings.Join(lines, "\n")), nil
}

func (ch *CommandHandler) handleOpenTerminal(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	cwd := ""
	if len(parts) >= 2 {
		cwd = parts[1]
	}
	if strings.TrimSpace(cwd) == "" {
		cwd = ch.collaborationCwdForChannel(msg.Channel)
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

// collaborationCwdForChannel returns source repo or sandbox path for a collab channel terminal.

// SetAssistantAgent sets the assistant agent reference for meeting notes functionality
func (ch *CommandHandler) SetAssistantAgent(assistant *agent.AssistantAgent) {
	ch.assistantAgent = assistant
}

// GetAssistantAgent returns the registered assistant agent, if any.

// GetAssistantAgent returns the registered assistant agent, if any.
func (ch *CommandHandler) GetAssistantAgent() *agent.AssistantAgent {
	return ch.assistantAgent
}

// RegisterRuntimeAgent tracks server-created runtime agents so collaboration
// wiring can reliably reach specialists/moderator/assistant and startup CLIs.

// RegisterRuntimeAgent tracks server-created runtime agents so collaboration
// wiring can reliably reach specialists/moderator/assistant and startup CLIs.
func (ch *CommandHandler) RegisterRuntimeAgent(agentInstance *agent.Agent) {
	if agentInstance == nil {
		return
	}
	ch.agentsMu.Lock()
	ch.runtimeAgents[agentInstance.Info.ID] = agentInstance
	ch.agentsMu.Unlock()
}

// AbortAgentGenerations cancels in-flight LLM work for a single agent across all channels.

// AbortAgentGenerations cancels in-flight LLM work for a single agent across all channels.
func (ch *CommandHandler) AbortAgentGenerations(agentID string) {
	if ch == nil || agentID == "" {
		return
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	if ra, ok := ch.runtimeAgents[agentID]; ok && ra != nil {
		ra.AbortAllChannels()
	}
	for _, ca := range ch.cliAgents {
		if ca != nil && ca.Info.ID == agentID {
			ca.AbortAllChannels()
		}
	}
	if ch.assistantAgent != nil && ch.assistantAgent.Agent != nil && ch.assistantAgent.Info.ID == agentID {
		ch.assistantAgent.AbortAllChannels()
	}
	if ra, ok := ch.repoAgents[agentID]; ok && ra != nil && ra.Agent != nil {
		ra.AbortAllChannels()
	}
	for _, ca := range ch.confluenceAgents {
		if ca != nil && ca.Agent != nil && ca.Info.ID == agentID {
			ca.AbortAllChannels()
		}
	}
}

// AbortRuntimeAgentsOnChannel cancels in-flight generations for runtime agents on channel.

// AbortRuntimeAgentsOnChannel cancels in-flight generations for runtime agents on channel.
func (ch *CommandHandler) AbortRuntimeAgentsOnChannel(channel string) {
	if ch == nil || channel == "" {
		return
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	for _, ra := range ch.runtimeAgents {
		if ra == nil {
			continue
		}
		ra.AbortChannel(channel)
	}
	for _, ca := range ch.cliAgents {
		if ca == nil {
			continue
		}
		ca.AbortChannel(channel)
	}
	if ch.assistantAgent != nil && ch.assistantAgent.Agent != nil {
		ch.assistantAgent.AbortChannel(channel)
	}
	for _, ra := range ch.repoAgents {
		if ra == nil || ra.Agent == nil {
			continue
		}
		ra.AbortChannel(channel)
	}
	for _, ca := range ch.confluenceAgents {
		if ca == nil || ca.Agent == nil {
			continue
		}
		ca.AbortChannel(channel)
	}
}

// StopAndUnregisterRuntimeAgent stops an in-process specialist and drops it from runtime tracking.

// StopAndUnregisterRuntimeAgent stops an in-process specialist and drops it from runtime tracking.
func (ch *CommandHandler) StopAndUnregisterRuntimeAgent(agentID string) {
	if ch == nil || agentID == "" {
		return
	}
	if ra, ok := ch.runtimeAgents[agentID]; ok && ra != nil {
		ra.Stop()
		delete(ch.runtimeAgents, agentID)
	}
}

// findAgentByID returns an in-process *agent.Agent for collab/chat subscriptions.
func (ch *CommandHandler) findAgentByID(agentID string) *agent.Agent {
	if ch == nil || agentID == "" {
		return nil
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	if a := ch.runtimeAgents[agentID]; a != nil {
		return a
	}
	for _, ca := range ch.cliAgents {
		if ca != nil && ca.Info.ID == agentID {
			return ca
		}
	}
	if ch.assistantAgent != nil && ch.assistantAgent.Agent != nil && ch.assistantAgent.Info.ID == agentID {
		return ch.assistantAgent.Agent
	}
	for _, ca := range ch.confluenceAgents {
		if ca != nil && ca.Agent != nil && ca.GetAgentInfo().ID == agentID {
			return ca.Agent
		}
	}
	return nil
}

// resolveLiveAgentID returns the registered hub agent ID for a collaboration participant.
func (ch *CommandHandler) resolveLiveAgentID(agentID, agentName string, agentType protocol.AgentType) string {
	if ch == nil {
		return agentID
	}
	if a := ch.findAgentByID(agentID); a != nil {
		return agentID
	}
	if ch.hub != nil && strings.TrimSpace(agentName) != "" {
		if info := ch.hub.FindLiveAgentByDisplayName(agentName, agentType); info != nil {
			return info.ID
		}
	}
	return agentID
}

// EnsureAgentSubscribedToChannel starts the agent's hub subscription on channelName.
// JoinChannel alone only updates membership; DM-spawned agents disable channel discovery
// and must be subscribed explicitly (e.g. collaboration rooms).
func (ch *CommandHandler) EnsureAgentSubscribedToChannel(ctx context.Context, agentID, channelName string) error {
	return ch.ensureAgentSubscribedToChannel(ctx, agentID, "", protocol.AgentType(""), channelName)
}

func (ch *CommandHandler) ensureAgentSubscribedToChannel(
	ctx context.Context,
	agentID, agentName string,
	agentType protocol.AgentType,
	channelName string,
) error {
	if ch == nil || channelName == "" {
		return nil
	}
	agentID = ch.resolveLiveAgentID(agentID, agentName, agentType)
	if agentID == "" {
		return nil
	}
	if a := ch.findAgentByID(agentID); a != nil {
		return a.AddChannel(ctx, channelName)
	}
	ch.agentsMu.RLock()
	defer ch.agentsMu.RUnlock()
	for _, ra := range ch.repoAgents {
		if ra != nil && ra.GetAgentInfo().ID == agentID {
			return ra.AddChannel(ctx, channelName)
		}
	}
	return nil
}

// Ensure CommandHandler implements CommandHandlerInterface
var _ agent.CommandHandlerInterface = (*CommandHandler)(nil)

// GetCommandDefinitions returns metadata for every registered slash command.
