package agent

import (
	"context"
	"log"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/mcp"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

// domainToolFallbackModel returns the Ollama model used when the chat model lacks native tool calling.
func domainToolFallbackModel(agentType protocol.AgentType) string {
	if cfg := mcp.AppConfig(); cfg != nil {
		if m := cfg.ToolModelForAgent(string(agentType)); m != "" {
			return m
		}
	}
	switch agentType {
	case protocol.AgentTypeCAD:
		return config.CadOllamaToolModel
	case protocol.AgentTypeBiology:
		return config.BioOllamaToolModel
	default:
		return ai.OllamaBiologyFallbackModel
	}
}

func reactToolsEnabledForModel(model string) bool {
	cfg := mcp.AppConfig()
	if cfg == nil {
		return config.DefaultConfig().Ollama.ModelUsesReactTools(model)
	}
	return cfg.Ollama.ModelUsesReactTools(model)
}

// toolCapableProvider returns eff when it supports native tool calling; otherwise ReAct or Qwen swap.
func (a *Agent) toolCapableProvider(ctx context.Context, eff ai.AIProvider) ai.AIProvider {
	if tc, ok := eff.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
		return eff
	}
	chatModel := strings.TrimSpace(eff.GetModel())
	if reactToolsEnabledForModel(chatModel) {
		react := ai.NewReActToolProvider(eff)
		if react != nil && react.SupportsTools() {
			log.Printf("[%s] Primary model lacks native tools; using ReAct wrapper on %q", a.Info.Name, chatModel)
			a.RecordRoutingSnapshot(RoutingSnapshot{
				ToolModel: chatModel,
				Reason:    "react_tools",
				Source:    "rules",
			})
			return react
		}
	}
	if fb := a.ollamaToolSwapProvider(ctx, eff); fb != nil {
		return fb
	}
	return eff
}

// ollamaToolSwapProvider returns an Ollama provider with native tools when the chat model lacks them.
func (a *Agent) ollamaToolSwapProvider(ctx context.Context, eff ai.AIProvider) ai.AIProvider {
	fallbackModel := ai.ImplementationToolModelFromContext(ctx)
	if fallbackModel == "" {
		fallbackModel = domainToolFallbackModel(a.Info.Type)
	}
	if fb := ollamaFallbackProvider(eff, fallbackModel); fb != nil {
		if tc, ok := fb.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
			log.Printf("[%s] Primary model lacks tool calling; using %q for MCP tool loop", a.Info.Name, fallbackModel)
			a.RecordRoutingSnapshot(RoutingSnapshot{
				ToolModel: fallbackModel,
				Reason:    "tool_fallback",
				Source:    "rules",
			})
			return fb
		}
	}
	return nil
}
