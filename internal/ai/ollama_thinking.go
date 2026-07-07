package ai

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrOllamaNoContent is returned when Ollama completes without assistant text.
	ErrOllamaNoContent = errors.New("no content in response")
	// ErrOllamaReasoningOnly is returned when only thinking/reasoning text arrived.
	ErrOllamaReasoningOnly = errors.New("model returned reasoning only; try again or use a non-reasoning model")
)

// errOllamaNoContent and errOllamaReasoningOnly are aliases kept for internal use.
var (
	errOllamaNoContent     = ErrOllamaNoContent
	errOllamaReasoningOnly = ErrOllamaReasoningOnly
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
		return 360 * time.Second
	}
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(m, "qwen3.5:27b") ||
		strings.Contains(m, "qwen3.5:122b") ||
		strings.Contains(m, ":35b") ||
		strings.Contains(m, ":27b") ||
		strings.Contains(m, ":122b") {
		return 600 * time.Second
	}
	if ollamaModelNeedsCollabTimeout(m) {
		// Match agent collab discussion deadline (480s) with headroom for HTTP round-trip.
		return 540 * time.Second
	}
	return 120 * time.Second
}

// ollamaModelNeedsCollabTimeout reports models that routinely exceed 120s on collab planning/execution.
func ollamaModelNeedsCollabTimeout(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	prefixes := []string{
		"qwen3.5:", "qwen2.5-coder:", "gemma3:", "devstral:", "codestral:", "deepseek-coder:", "ornith", "nj-ornith:",
	}
	for _, p := range prefixes {
		if strings.Contains(m, p) {
			return true
		}
	}
	return false
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
