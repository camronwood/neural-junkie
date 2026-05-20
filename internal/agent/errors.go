package agent

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/ai"
)

var ollamaModelNotFoundRE = regexp.MustCompile(`model ['"]?([^'"]+)['"]? not found`)

// ollamaMissingModel extracts a model tag from an Ollama "model not found" API error.
func ollamaMissingModel(err error) (model string, ok bool) {
	if err == nil {
		return "", false
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "not found") {
		return "", false
	}
	if !strings.Contains(lower, "ollama") && !strings.Contains(lower, "model '") {
		return "", false
	}
	if m := ollamaModelNotFoundRE.FindStringSubmatch(err.Error()); len(m) >= 2 {
		return strings.TrimSpace(m[1]), true
	}
	return "", strings.Contains(lower, "ollama") && strings.Contains(lower, "model")
}

func classifyUserFacingError(err error) (message, code string, retryable bool) {
	if err == nil {
		return "Sorry, I encountered an unexpected error.", "unknown", true
	}
	lower := strings.ToLower(err.Error())

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ai.ErrCLIProviderTimeout) || strings.Contains(lower, "timed out") {
		return "Sorry, the response timed out before completion. Please try again.", "timeout", true
	}
	if strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "resource_exhausted") || strings.Contains(lower, "capacity") {
		return "Sorry, the provider is rate-limited right now. Please try again in a moment.", "rate_limit", true
	}
	if strings.Contains(lower, "workspace trust") {
		return "I couldn't run that because workspace trust is required for this agent. Please trust the workspace and try again.", "workspace_trust", false
	}
	if strings.Contains(lower, "does not support tools") {
		return "This Ollama model does not support automatic tool calling. BiologyExpert will answer in plain chat; for sequence/fold tools use a tool-capable model (e.g. qwen2.5) or ask for analysis in natural language.", "provider_error", true
	}
	if errors.Is(err, ai.ErrOllamaNoContent) || strings.Contains(lower, "no content in response") {
		return "The model returned an empty reply. Try a shorter question or start a fresh DM thread; nj-bio sometimes fails on long system prompts.", "provider_error", true
	}
	if errors.Is(err, ai.ErrOllamaReasoningOnly) || strings.Contains(lower, "reasoning only") {
		return "The model produced reasoning text but no visible answer. Please try again.", "provider_error", true
	}
	if model, ok := ollamaMissingModel(err); ok {
		if model != "" {
			return "The Ollama model \"" + model + "\" is not installed. Open Model Library (⇧⌘M) → Hugging Face → Neural Junkie Bio 8B (GGUF) → download → Import to Ollama, then try again.", "provider_unavailable", false
		}
		return "The configured Ollama model is not installed. Open Model Library (⇧⌘M) to download or import the model, then try again.", "provider_unavailable", false
	}
	if strings.Contains(lower, "executable file not found") || strings.Contains(lower, "is not installed") && strings.Contains(lower, "cli") {
		return "I couldn't run the configured CLI agent because it is not available on this machine.", "provider_unavailable", false
	}
	return "Sorry, I encountered an error while generating a response. Please try again.", "provider_error", true
}
