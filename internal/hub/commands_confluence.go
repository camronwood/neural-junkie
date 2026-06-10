package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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

	return ch.systemResponse(msg.Channel, "ℹ️ Confluence credentials format look valid. A live API call is not implemented yet — verify by indexing a space or asking the Confluence agent."), nil
}

// handleMigrateAgentNames migrates existing agents with problematic names to @mention-compatible format
