package ai

import (
	"context"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// SystemPromptSeparator is embedded in prompts to split system instructions from user content.
// Providers that support a "system" role (Ollama, Claude, LM Studio) will split on this marker
// and send the first part as a system message and the rest as a user message.
// Providers that don't (mock, CLI) simply strip the marker and send as one user message.
const SystemPromptSeparator = "\n---SYSTEM_PROMPT_END---\n"

// SplitSystemPrompt splits a prompt at the SystemPromptSeparator.
// Returns (systemPrompt, userMessage). If no separator is found, systemPrompt is empty
// and the entire prompt is returned as userMessage.
func SplitSystemPrompt(prompt string) (string, string) {
	parts := strings.SplitN(prompt, SystemPromptSeparator, 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", prompt
}

// isDeepAnalysisPrompt checks if the user message suggests a detailed analysis request
// (code review, audit, explanation, etc.) that warrants a higher token limit.
func isDeepAnalysisPrompt(userMessage string) bool {
	lower := strings.ToLower(userMessage)
	deepKeywords := []string{
		"review", "audit", "analyze", "explain", "walk through",
		"deep dive", "code review", "security review", "examine",
		"investigate", "break down", "detailed",
	}
	for _, kw := range deepKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// AIProvider defines the interface for AI providers
type AIProvider interface {
	GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error)
	GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error)
	GetModel() string
}

// StreamToken represents a single token/chunk from a streaming AI response.
type StreamToken struct {
	Content  string
	Thinking string // reasoning delta (Ollama thinking models)
	Done     bool
	Error    error
}

// StreamingProvider is an optional interface that AIProviders can implement
// to support token-by-token response streaming. The agent checks for this
// at runtime and falls back to GenerateResponse if not available.
type StreamingProvider interface {
	GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error)
	SupportsStreaming() bool
}
