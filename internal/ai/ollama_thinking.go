package ai

import (
	"errors"
	"strings"
	"time"
)

var (
	errOllamaNoContent      = errors.New("no content in response")
	errOllamaReasoningOnly  = errors.New("model returned reasoning only; try again or use a non-reasoning model")
)

// ollamaModelWantsThinking reports whether the model should use Ollama's think API.
func ollamaModelWantsThinking(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	// Reasoning-oriented Ollama tags (deepseek-r1, qwen3 thinking variants, etc.)
	if strings.Contains(m, "deepseek-r1") {
		return true
	}
	if strings.Contains(m, ":r1") || strings.HasSuffix(m, "-r1") {
		return true
	}
	if strings.Contains(m, "qwen3") && strings.Contains(m, "thinking") {
		return true
	}
	return false
}

func ollamaHTTPTimeout(model string) time.Duration {
	if ollamaModelWantsThinking(model) {
		return 300 * time.Second
	}
	return 120 * time.Second
}

func boolPtr(v bool) *bool {
	return &v
}

// ollamaFinalizeContent returns assistant reply text or an error when only reasoning arrived.
func ollamaFinalizeContent(content, thinking string) (string, error) {
	content = strings.TrimSpace(content)
	if content != "" {
		return content, nil
	}
	if strings.TrimSpace(thinking) != "" {
		return "", errOllamaReasoningOnly
	}
	return "", errOllamaNoContent
}
