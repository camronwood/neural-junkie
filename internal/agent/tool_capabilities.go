package agent

import (
	"encoding/json"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// DescribeToolCapabilities returns tools and model routing for a running agent.
func (a *Agent) DescribeToolCapabilities() protocol.AgentToolCapabilities {
	out := protocol.AgentToolCapabilities{
		AgentID:      a.Info.ID,
		AgentName:    a.Info.Name,
		AgentType:    string(a.Info.Type),
		Tools:        []protocol.AgentToolDefinition{},
		ChatModel:    a.Info.AIModel,
		ChatProvider: a.Info.AIProvider,
	}

	if a.Info.Type == protocol.AgentTypeCLI || isCLIProvider(a.Info.AIProvider) {
		out.Notes = append(out.Notes, "CLI agent — uses external CLI tools (filesystem, shell, etc.), not hub MCP tools.")
		return out
	}

	mcpCfg := mcp.GetMCPServerConfig(string(a.Info.Type))
	out.MCPEnabled = mcpCfg.Enabled
	if mcpCfg.Enabled {
		out.MCPPort = mcpCfg.Port
	}

	eff := a.GetAIProvider()
	out.ChatNativeTools = providerSupportsTools(eff)
	loopModel, usesFallback, loopMode := a.effectiveToolLoopRouting(eff)
	out.ToolLoopModel = loopModel
	out.ToolLoopUsesFallback = usesFallback
	out.ToolLoopMode = loopMode

	for _, td := range a.agentToolDefinitions(nil) {
		source := "mcp"
		if td.Name == generateImageToolName || td.Name == generateMusicToolName {
			source = "builtin"
		}
		out.Tools = append(out.Tools, protocol.AgentToolDefinition{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  parseToolInputSchema(td.InputSchema),
			Source:      source,
		})
	}
	out.ToolCount = len(out.Tools)

	if !mcpCfg.Enabled && a.MCPServer != nil {
		out.Notes = append(out.Notes, "MCP is disabled in Settings for this agent type.")
	}
	if out.ToolCount == 0 && mcpCfg.Enabled && a.MCPServer == nil {
		out.Notes = append(out.Notes, "MCP server is not running for this agent.")
	}
	if out.ToolCount > 0 && out.ToolLoopMode == "react" {
		out.Notes = append(out.Notes, "Chat model uses ReAct text tool calls on the same model.")
	}
	if out.ToolCount > 0 && !out.ChatNativeTools && out.ToolLoopUsesFallback {
		out.Notes = append(out.Notes, "Chat model does not support native tools; MCP tools run via the tool-loop model.")
	}

	return out
}

// CapabilitiesFromAgentInfo builds a minimal capabilities payload when the runtime agent is unavailable.
func CapabilitiesFromAgentInfo(info *protocol.AgentInfo) protocol.AgentToolCapabilities {
	if info == nil {
		return protocol.AgentToolCapabilities{}
	}
	out := protocol.AgentToolCapabilities{
		AgentID:      info.ID,
		AgentName:    info.Name,
		AgentType:    string(info.Type),
		Tools:        []protocol.AgentToolDefinition{},
		ChatModel:    info.AIModel,
		ChatProvider: info.AIProvider,
	}
	if info.Type == protocol.AgentTypeCLI || isCLIProvider(info.AIProvider) {
		out.Notes = append(out.Notes, "CLI agent — uses external CLI tools, not hub MCP.")
		return out
	}
	mcpCfg := mcp.GetMCPServerConfig(string(info.Type))
	out.MCPEnabled = mcpCfg.Enabled
	out.MCPPort = mcpCfg.Port
	if !mcpCfg.Enabled {
		out.Notes = append(out.Notes, "Enable MCP in Settings or the relevant domain pack to use specialist tools.")
	} else {
		out.Notes = append(out.Notes, "Agent is not running in the hub process; start the specialist or open a DM to see live tools.")
	}
	return out
}

func isCLIProvider(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	return strings.HasSuffix(p, "-cli") || p == "cursor-cli" || p == "gemini-cli" || p == "claude-cli" || p == "copilot-cli" || p == "codex-cli"
}

func providerSupportsTools(eff ai.AIProvider) bool {
	if eff == nil {
		return false
	}
	if _, ok := eff.(*ai.ReActToolProvider); ok {
		return false
	}
	tp, ok := eff.(ai.ToolCapableProvider)
	return ok && tp.SupportsTools()
}

func (a *Agent) effectiveToolLoopModel(eff ai.AIProvider) (model string, usesFallback bool) {
	model, usesFallback, _ = a.effectiveToolLoopRouting(eff)
	return model, usesFallback
}

func (a *Agent) effectiveToolLoopRouting(eff ai.AIProvider) (model string, usesFallback bool, mode string) {
	if eff == nil {
		return "", false, ""
	}
	chatModel := modelNameFromProvider(eff)
	if providerSupportsTools(eff) {
		return chatModel, false, "native"
	}
	if reactToolsEnabledForModel(chatModel) {
		return chatModel, false, "react"
	}
	fallbackModel := domainToolFallbackModel(a.Info.Type)
	toolEff := toolCapableProviderForDescribe(eff, fallbackModel)
	if toolEff != nil && providerSupportsTools(toolEff) {
		return modelNameFromProvider(toolEff), modelNameFromProvider(toolEff) != chatModel, "fallback"
	}
	return chatModel, false, ""
}

func toolCapableProviderForDescribe(eff ai.AIProvider, fallbackModel string) ai.AIProvider {
	if providerSupportsTools(eff) {
		return eff
	}
	chatModel := modelNameFromProvider(eff)
	if reactToolsEnabledForModel(chatModel) {
		return ai.NewReActToolProvider(eff)
	}
	if fallbackModel == "" {
		fallbackModel = ai.OllamaBiologyFallbackModel
	}
	if fb := ollamaFallbackProvider(eff, fallbackModel); fb != nil {
		if tp, ok := fb.(ai.ToolCapableProvider); ok && tp.SupportsTools() {
			return fb
		}
	}
	return eff
}

func modelNameFromProvider(eff ai.AIProvider) string {
	if eff == nil {
		return ""
	}
	type modelGetter interface {
		GetModel() string
	}
	if mg, ok := eff.(modelGetter); ok {
		return mg.GetModel()
	}
	return ""
}

func parseToolInputSchema(schema json.RawMessage) []protocol.AgentToolParam {
	if len(schema) == 0 {
		return nil
	}
	var doc struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil
	}
	reqSet := make(map[string]bool, len(doc.Required))
	for _, r := range doc.Required {
		reqSet[r] = true
	}
	var out []protocol.AgentToolParam
	for name, prop := range doc.Properties {
		out = append(out, protocol.AgentToolParam{
			Name:        name,
			Required:    reqSet[name],
			Description: prop.Description,
		})
	}
	return out
}
