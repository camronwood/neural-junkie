package agent

import (
	"context"
	"log"

	"github.com/camronwood/neural-junkie/internal/ai"
)

// toolCapableProvider returns eff when it supports native tool calling; otherwise an Ollama
// fallback on the same endpoint (implementation tool model or biology default).
func (a *Agent) toolCapableProvider(ctx context.Context, eff ai.AIProvider) ai.AIProvider {
	if tc, ok := eff.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
		return eff
	}
	fallbackModel := ai.ImplementationToolModelFromContext(ctx)
	if fallbackModel == "" {
		fallbackModel = ai.OllamaBiologyFallbackModel
	}
	if fb := ollamaFallbackProvider(eff, fallbackModel); fb != nil {
		if tc, ok := fb.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
			log.Printf("[%s] Primary model lacks tool calling; using %q for MCP tool loop", a.Info.Name, fallbackModel)
			return fb
		}
	}
	return eff
}
