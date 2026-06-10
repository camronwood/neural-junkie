package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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

	return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Agent '%s' not found. Use /list-agents to see available agents.", agentName)), nil
}

// handleListExports lists all exported agents

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
		response.WriteString(fmt.Sprintf("📊 **Total:** %d exports (%d repo) | %.1f KB total",
			stats.TotalExports, stats.RepoExports, float64(stats.TotalSize)/1024))
	}

	return ch.systemResponse(msg.Channel, response.String()), nil
}

// handleDeleteExport deletes an exported agent

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
	for _, a := range ch.hub.ListAgents() {
		if strings.EqualFold(a.Name, export.Agent.Name) {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ An agent named '%s' already exists", export.Agent.Name)), nil
		}
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

	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unsupported agent type: %s", export.Agent.Type)), nil
	}
}

// handleExportAllAgents exports all available agents to MCP format

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

	// Build response
	var response strings.Builder
	if len(exported) > 0 {
		response.WriteString(fmt.Sprintf("✅ Exported %d agents:\n", len(exported)))
		for _, name := range exported {
			response.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	} else {
		response.WriteString("📦 No agents available to export. Create agents first with /create-repo-agent.\n")
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
	return ch.systemResponse(msg.Channel, "ℹ️ GitHub token format looks valid. A live API call is not implemented yet — verify access by running a GitHub-related command."), nil
}

// handleTestConfluenceConnection tests Confluence API connection
