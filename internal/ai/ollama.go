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

// OllamaProvider implements AI responses using Ollama local LLM
type OllamaProvider struct {
	Endpoint   string
	Model      string
	httpClient *http.Client
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// OllamaMessage represents a message in Ollama API format
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model   string        `json:"model"`
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// NewOllamaProvider creates a new Ollama AI provider
func NewOllamaProvider() (*OllamaProvider, error) {
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.1"
	}

	return &OllamaProvider{
		Endpoint: endpoint,
		Model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Ollama can be slower than cloud APIs
		},
	}, nil
}

// NewOllamaProviderWithConfig creates a new Ollama AI provider with custom configuration
func NewOllamaProviderWithConfig(endpoint, model string) *OllamaProvider {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.1"
	}

	return &OllamaProvider{
		Endpoint: endpoint,
		Model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateResponse generates a response using Ollama API
func (o *OllamaProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	// Split system prompt from user message for better model adherence
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	// Build messages array
	messages := []OllamaMessage{}

	// Add system message if present (Ollama supports "system" role)
	if systemPrompt != "" {
		messages = append(messages, OllamaMessage{
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
		messages = append(messages, OllamaMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, OllamaMessage{
		Role:    "user",
		Content: userMessage,
	})

	request := OllamaRequest{
		Model:    o.Model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("Ollama API error: %s", response.Error)
	}

	if response.Message.Content == "" {
		return "", fmt.Errorf("no content in response")
	}

	return response.Message.Content, nil
}

// GenerateVisionResponse generates a response using Ollama API with image input
// Note: Most Ollama models don't support vision, so this returns an error
func (o *OllamaProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	// For now, Ollama doesn't support vision in most models
	// This could be extended to support vision-capable models like llava
	return "", fmt.Errorf("vision not supported by Ollama provider (model: %s)", o.Model)
}

// GetModel returns the model name
func (o *OllamaProvider) GetModel() string {
	return o.Model
}

// GetEndpoint returns the Ollama endpoint
func (o *OllamaProvider) GetEndpoint() string {
	return o.Endpoint
}

// TestConnection tests the connection to Ollama server
func (o *OllamaProvider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.Endpoint+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama server returned status %d", resp.StatusCode)
	}

	return nil
}

// GetAvailableModels returns a list of available models from Ollama
func (o *OllamaProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.Endpoint+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	var response struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	models := make([]string, len(response.Models))
	for i, model := range response.Models {
		models[i] = model.Name
	}

	return models, nil
}

// SetModel changes the model for this provider
func (o *OllamaProvider) SetModel(model string) {
	o.Model = model
}

// SetEndpoint changes the endpoint for this provider
func (o *OllamaProvider) SetEndpoint(endpoint string) {
	o.Endpoint = endpoint
}
