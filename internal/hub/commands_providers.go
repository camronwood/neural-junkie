package hub

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

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
	case "huggingface", "hf":
		return "Qwen/Qwen2.5-Coder-7B-Instruct"
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
	case "huggingface", "hf":
		token := ai.ResolveHFToken("")
		if token == "" {
			return nil, "", fmt.Errorf("HF token required: set HF_TOKEN or add a huggingface provider in Settings")
		}
		if resolvedModel == "" {
			return nil, "", fmt.Errorf("model (Hugging Face repo id) is required for huggingface provider")
		}
		return ai.NewHuggingFaceProvider("", token, resolvedModel), resolvedModel, nil
	default:
		return nil, "", fmt.Errorf("invalid provider %q (allowed: claude, ollama, lmstudio, huggingface)", provider)
	}
}

// SwitchAgentProvider switches a single live agent's provider/model.

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

	ch.persistAgentProviderSwitch(targetAgent, provider, resolvedModel)

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

// SwitchAllProviders switches all currently registered live agents.
func (ch *CommandHandler) SwitchAllProviders(provider, model, channel string, metadata map[string]interface{}) (int, error) {
	if ch.appConfig != nil {
		ch.appConfig.ClearAllAgentModels()
		_ = ch.appConfig.Save()
	}
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
		return switchedCount, fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return switchedCount, nil
}

func (ch *CommandHandler) persistAgentProviderSwitch(agentInfo *protocol.AgentInfo, provider, model string) {
	if ch.appConfig == nil || agentInfo == nil {
		return
	}
	providerID := ch.providerIDForRuntime(provider)
	if !ch.appConfig.SetAgentRuntimeProvider(agentInfo.Name, string(agentInfo.Type), providerID, model) {
		return
	}
	if err := ch.appConfig.Save(); err != nil {
		log.Printf("persist agent provider switch: %v", err)
		return
	}
	if ch.providerCache != nil {
		ch.providerCache.Clear()
	}
}

func (ch *CommandHandler) providerIDForRuntime(provider string) string {
	if ch.appConfig == nil {
		return ""
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "hf" {
		provider = "huggingface"
	}
	for _, p := range ch.appConfig.ListProvidersSnapshot() {
		if p.Type == provider {
			return p.ID
		}
	}
	if provider == "ollama" {
		return "ollama-local"
	}
	return ""
}

// ── Channel management commands ──────────────────────────────────────────
