package agent

import (
	"context"
	"log"

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

// toolCapableProvider returns eff when it supports native tool calling; otherwise an Ollama
// fallback on the same endpoint (implementation tool model, domain pack default, or biology default).
func (a *Agent) toolCapableProvider(ctx context.Context, eff ai.AIProvider) ai.AIProvider {
	if tc, ok := eff.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
		return eff
	}
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
	return eff
}
