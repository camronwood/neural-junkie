package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// LMStudioProvider implements AI responses using LM Studio local LLM server
// LM Studio provides an OpenAI-compatible API endpoint
type LMStudioProvider struct {
	Endpoint   string
	Model      string
	httpClient *http.Client
}

// OpenAICompatibleRequest represents a request to OpenAI-compatible API (used by LM Studio)
type OpenAICompatibleRequest struct {
	Model    string                    `json:"model"`
	Messages []OpenAICompatibleMessage `json:"messages"`
	Stream   bool                      `json:"stream,omitempty"`
}

// OpenAICompatibleMessage represents a message in OpenAI API format
type OpenAICompatibleMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAICompatibleResponse represents a response from OpenAI-compatible API
type OpenAICompatibleResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int                     `json:"index"`
		Message      OpenAICompatibleMessage `json:"message"`
		FinishReason string                  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ModelsResponse represents the response from /v1/models endpoint
type ModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// NewLMStudioProvider creates a new LM Studio AI provider
func NewLMStudioProvider() (*LMStudioProvider, error) {
	endpoint := os.Getenv("LM_STUDIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1"
	}

	model := os.Getenv("LM_STUDIO_MODEL")
	if model == "" {
		model = "" // Will be determined from available models or use first available
	}

	return &LMStudioProvider{
		Endpoint: endpoint,
		Model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LM Studio can be slower than cloud APIs
		},
	}, nil
}

// NewLMStudioProviderWithConfig creates a new LM Studio AI provider with custom configuration
func NewLMStudioProviderWithConfig(endpoint, model string) *LMStudioProvider {
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1"
	}

	return &LMStudioProvider{
		Endpoint: endpoint,
		Model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateResponse generates a response using LM Studio's OpenAI-compatible API
func (l *LMStudioProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	// Split system prompt from user message for better model adherence
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	// Build messages array
	messages := []OpenAICompatibleMessage{}

	// Add system message if present (OpenAI-compatible API supports "system" role)
	if systemPrompt != "" {
		messages = append(messages, OpenAICompatibleMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Add conversation history (limit to last 10 messages to avoid token limits)
	historyLimit := 10
	if len(conversationHistory) > historyLimit {
		conversationHistory = conversationHistory[len(conversationHistory)-historyLimit:]
	}

	for _, msg := range conversationHistory {
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		messages = append(messages, OpenAICompatibleMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, OpenAICompatibleMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Determine model to use
	model := l.Model
	if model == "" {
		// Try to get the first available model
		availableModels, err := l.GetAvailableModels(ctx)
		if err == nil && len(availableModels) > 0 {
			model = availableModels[0]
		} else {
			return "", fmt.Errorf("no model specified and unable to fetch available models: %w", err)
		}
	}

	request := OpenAICompatibleRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LM Studio API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAICompatibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("LM Studio API error: %s (type: %s, code: %s)", response.Error.Message, response.Error.Type, response.Error.Code)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	if response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no content in response")
	}

	return response.Choices[0].Message.Content, nil
}

// GenerateVisionResponse generates a response using LM Studio API with image input
// Note: Most LM Studio models don't support vision, so this returns an error
func (l *LMStudioProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	// For now, LM Studio doesn't support vision in most models
	// This could be extended to support vision-capable models
	return "", fmt.Errorf("vision not supported by LM Studio provider (model: %s)", l.Model)
}

// GetModel returns the model name
func (l *LMStudioProvider) GetModel() string {
	return l.Model
}

// GetEndpoint returns the LM Studio endpoint
func (l *LMStudioProvider) GetEndpoint() string {
	return l.Endpoint
}

// TestConnection tests the connection to LM Studio server
func (l *LMStudioProvider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", l.Endpoint+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to LM Studio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LM Studio server returned status %d", resp.StatusCode)
	}

	return nil
}

// GetAvailableModels returns a list of available models from LM Studio
func (l *LMStudioProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", l.Endpoint+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get models: status %d, body: %s", resp.StatusCode, string(body))
	}

	var response ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	models := make([]string, len(response.Data))
	for i, model := range response.Data {
		models[i] = model.ID
	}

	return models, nil
}

// SetModel changes the model for this provider
func (l *LMStudioProvider) SetModel(model string) {
	l.Model = model
}

// SetEndpoint changes the endpoint for this provider
func (l *LMStudioProvider) SetEndpoint(endpoint string) {
	l.Endpoint = endpoint
}
