package ai

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MockProvider provides mock AI responses for testing
type MockProvider struct {
	Model string
}

// NewMockProvider creates a new mock AI provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		Model: "mock-model",
	}
}

// GenerateResponse generates a mock response
func (m *MockProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	// Strip system prompt separator if present (mock doesn't use it)
	_, userMessage := SplitSystemPrompt(prompt)
	_ = userMessage
	return "This is a mock response for testing purposes.", nil
}

// GenerateVisionResponse generates a mock vision response
func (m *MockProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return "This is a mock vision response for design analysis. In a real implementation, this would analyze the uploaded image and generate CSS/HTML.", nil
}

// GetModel returns the model name
func (m *MockProvider) GetModel() string {
	return m.Model
}
