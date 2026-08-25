package hub

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/ai"
	loraregistry "github.com/camronwood/neural-junkie/internal/lora/registry"
	"github.com/camronwood/neural-junkie/internal/learning"
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
			ch.enrichExport(export, agentID)

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

// handleImportAgentMCP imports an agent from an MCP export / Share Agent bundle.
//
// Usage: /import-agent-mcp <file-path> [--hydrate] [--repo <path>]
//
//   --hydrate     force knowledge-only hydration from embedded resources
//                 instead of re-indexing the repository from disk.
//   --repo <path> remap the repository path (e.g. onto this machine's
//                 checkout) instead of using the path baked into the export.
//
// When neither flag is given and the export's repository path does not
// exist on this machine, the import auto-hydrates from the bundled
// resources so the agent still comes up with its knowledge intact.
func (ch *CommandHandler) handleImportAgentMCP(ctx context.Context, msg *protocol.Message, parts []string) (*protocol.Message, error) {
	if len(parts) < 2 {
		return ch.systemResponse(msg.Channel, "Usage: /import-agent-mcp <file-path> [--hydrate] [--repo <path>]"), nil
	}

	filePath, hydrate, repoOverride := parseImportAgentMCPFlags(parts)

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
		repoPath := strings.TrimSpace(repoOverride)
		if repoPath == "" {
			repoPath = export.Agent.Repository
		}
		if !hydrate {
			if repoPath == "" {
				hydrate = true
			} else if _, statErr := os.Stat(repoPath); statErr != nil {
				// The path baked into the export isn't available on this
				// machine (different user/OS) — fall back to hydrating
				// from the bundle's embedded resources instead of failing.
				hydrate = true
			}
		}

		var repoAgent *agent.RepoAgent
		if hydrate {
			skipPath := repoPath
			if skipPath == "" {
				skipPath = "(hydrated)"
			}
			repoAgent, err = agent.NewRepoAgentWithOptions(export.Agent.Name, skipPath, ch.aiProvider, ch.hub, agent.RepoAgentOptions{SkipPathCheck: true})
			if err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create repo agent: %v", err)), nil
			}
			if err := repoAgent.HydrateFromExport(export, repoPath); err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to hydrate agent knowledge: %v", err)), nil
			}
		} else {
			repoAgent, err = agent.NewRepoAgent(export.Agent.Name, repoPath, ch.aiProvider, ch.hub)
			if err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to create repo agent: %v", err)), nil
			}
		}

		// Register with hub
		if err := ch.hub.RegisterAgent(&repoAgent.Info); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to register agent: %v", err)), nil
		}

		// Join channel
		if err := ch.hub.JoinChannel(repoAgent.Info.ID, msg.Channel); err != nil {
			return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to join channel: %v", err)), nil
		}

		if hydrate {
			if err := repoAgent.StartHydrated(ctx, msg.Channel); err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
			}
		} else {
			if err := repoAgent.StartWithIndexing(ctx, msg.Channel); err != nil {
				return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Failed to start agent: %v", err)), nil
			}
		}

		// Track the repo agent
		ch.repoAgents[repoAgent.Info.ID] = repoAgent

		extras := ch.applyShareBundleExtras(repoAgent.Info.ID, export)

		mode := "indexed from disk"
		if hydrate {
			mode = "hydrated from bundle"
		}
		return ch.systemResponse(msg.Channel, fmt.Sprintf("✅ Imported repo agent '%s' from %s (%s)\n📄 Resources: %d | 💬 Prompts: %d%s",
			export.Agent.Name, filePath, mode, export.GetResourceCount(), export.GetPromptCount(), extras)), nil

	default:
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Unsupported agent type: %s", export.Agent.Type)), nil
	}
}

// parseImportAgentMCPFlags splits /import-agent-mcp arguments into the file
// path plus the optional --hydrate / --repo <path> flags.
func parseImportAgentMCPFlags(parts []string) (filePath string, hydrate bool, repoOverride string) {
	if len(parts) < 2 {
		return "", false, ""
	}
	filePath = parts[1]
	for i := 2; i < len(parts); i++ {
		switch parts[i] {
		case "--hydrate":
			hydrate = true
		case "--repo":
			if i+1 < len(parts) {
				repoOverride = parts[i+1]
				i++
			}
		}
	}
	return filePath, hydrate, repoOverride
}

// applyShareBundleExtras hydrates custom rules and learnings carried in a
// Share Agent bundle onto a freshly imported agent, returning a short
// human-readable summary suffix for the command response.
func (ch *CommandHandler) applyShareBundleExtras(agentID string, export *mcp_export.AgentExport) string {
	if export == nil {
		return ""
	}
	var b strings.Builder
	if md := strings.TrimSpace(export.CustomRulesMarkdown); md != "" {
		if err := ch.hub.SetAgentCustomRulesMarkdown(agentID, md); err == nil {
			b.WriteString("\n📋 Custom rules applied")
		}
	}
	if len(export.Learnings) > 0 {
		added, skipped := ch.hub.ImportShareLearnings(agentID, export.Agent.Name, export.Agent.Type, export.Learnings)
		if added > 0 || skipped > 0 {
			b.WriteString(fmt.Sprintf("\n🧠 Learnings: %d added, %d skipped (duplicates)", added, skipped))
		}
	}
	if export.LoRA != nil && (export.LoRA.ComposedTag != "" || export.LoRA.HFRepoID != "") {
		b.WriteString(fmt.Sprintf("\n🎛️ LoRA metadata available: %s (base %s) — pull/train manually if desired", export.LoRA.ComposedTag, export.LoRA.BaseOllamaTag))
	}
	return b.String()
}

// ExportAgentBundle builds a full Share Agent bundle (export + custom rules +
// agent-scoped learnings + LoRA metadata) for the given agent ID. Currently
// supported for repo agents only — mirrors handleExportAgentMCP but returns
// the bundle directly instead of writing it to the export store, for the
// HTTP "Share" endpoint.
func (ch *CommandHandler) ExportAgentBundle(agentID string) (*mcp_export.AgentExport, error) {
	ch.agentsMu.RLock()
	repoAgent, ok := ch.repoAgents[agentID]
	ch.agentsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("repo agent %s not found (only repo agents support Share Agent bundles today)", agentID)
	}
	export, err := repoAgent.ExportToMCP()
	if err != nil {
		return nil, err
	}
	ch.enrichExport(export, agentID)
	return export, nil
}

// enrichExport adds custom rules markdown, agent-scoped learnings, and active
// LoRA metadata (when present) onto an export produced by ExportToMCP, ahead
// of persisting a Share Agent bundle.
func (ch *CommandHandler) enrichExport(export *mcp_export.AgentExport, agentID string) {
	if export == nil {
		return
	}
	if md := ch.hub.GetAgentCustomRulesMarkdown(agentID); md != "" {
		export.CustomRulesMarkdown = md
	}
	for _, e := range learning.ListGlobal(agentID) {
		export.Learnings = append(export.Learnings, mcp_export.LearningEntry{
			Content:     e.Content,
			Category:    string(e.Category),
			Scope:       string(e.Scope),
			AgentName:   e.AgentName,
			AgentType:   e.AgentType,
			ContentHash: e.ContentHash,
		})
	}
	if reg, err := loraregistry.NewStore(""); err == nil {
		if entry, ok := reg.ActiveForAgent(agentID); ok {
			export.LoRA = &mcp_export.LoRAMetadata{
				ComposedTag:   entry.OllamaTag,
				BaseOllamaTag: entry.BaseOllamaTag,
				TrainingManifest: &mcp_export.TrainingManifest{
					RowCount:    entry.RowCount,
					DatasetHash: entry.DatasetHash,
					LastTrained: entry.ExportedAt.Format(time.RFC3339),
				},
			}
		}
	}
}

// handleExportAllAgents exports all available agents to MCP format

// handleExportAllAgents exports all available agents to MCP format
func (ch *CommandHandler) handleExportAllAgents(ctx context.Context, msg *protocol.Message) (*protocol.Message, error) {
	var exported []string
	var errors []string

	// Export all repo agents
	for agentID, repoAgent := range ch.repoAgents {
		export, err := repoAgent.ExportToMCP()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Repo agent '%s': %v", repoAgent.Info.Name, err))
			continue
		}
		ch.enrichExport(export, agentID)

		if err := ch.exportStorage.SaveExport(export); err != nil {
			errors = append(errors, fmt.Sprintf("Repo agent '%s': %v", repoAgent.Info.Name, err))
			continue
		}

		exported = append(exported, fmt.Sprintf("%s (repo)", repoAgent.Info.Name))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Connection failed: %v", err)), nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ Connection failed: %v", err)), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ch.systemResponse(msg.Channel, "❌ GitHub connection failed: invalid or expired token"), nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ch.systemResponse(msg.Channel, fmt.Sprintf("❌ GitHub API returned HTTP %d", resp.StatusCode)), nil
	}

	return ch.systemResponse(msg.Channel, "✅ GitHub connection successful!"), nil
}

// handleTestConfluenceConnection tests Confluence API connection
