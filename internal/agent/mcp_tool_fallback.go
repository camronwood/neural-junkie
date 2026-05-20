package agent

import (
	"log"

	"github.com/camronwood/neural-junkie/internal/ai"
)

// toolCapableProvider returns eff when it supports native tool calling; otherwise an Ollama
// fallback (qwen2.5:7b) on the same endpoint so biology MCP tools still run.
func (a *Agent) toolCapableProvider(eff ai.AIProvider) ai.AIProvider {
	if tc, ok := eff.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
		return eff
	}
	if fb := ollamaFallbackProvider(eff, ai.OllamaBiologyFallbackModel); fb != nil {
		if tc, ok := fb.(ai.ToolCapableProvider); ok && tc.SupportsTools() {
			log.Printf("[%s] Primary model lacks tool calling; using %q for MCP tool loop", a.Info.Name, ai.OllamaBiologyFallbackModel)
			return fb
		}
	}
	return eff
}
