package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or multimodal []part
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

	text := strings.TrimSpace(openAIMessageTextContent(response.Choices[0].Message.Content))
	if text == "" {
		return "", fmt.Errorf("no content in response")
	}

	return text, nil
}

// GenerateVisionResponse uses OpenAI-compatible multimodal chat (vision models in LM Studio).
func (l *LMStudioProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	if len(imageData) == 0 {
		return "", fmt.Errorf("empty image")
	}
	return l.GenerateMultimodal(ctx, prompt, []protocol.UserImagePart{{MIME: imageType, Data: imageData}}, conversationHistory)
}

func (l *LMStudioProvider) openAICompatFor(model string) *OpenAICompatProvider {
	return NewOpenAICompatProvider(l.Endpoint, "", model, nil)
}

func (l *LMStudioProvider) resolveChatModel(ctx context.Context) (string, error) {
	if l.Model != "" {
		return l.Model, nil
	}
	availableModels, err := l.GetAvailableModels(ctx)
	if err == nil && len(availableModels) > 0 {
		return availableModels[0], nil
	}
	return "", fmt.Errorf("no model specified and unable to fetch available models: %w", err)
}

// GenerateMultimodal implements MultimodalProvider for LM Studio's OpenAI-compatible API.
func (l *LMStudioProvider) GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error) {
	model, err := l.resolveChatModel(ctx)
	if err != nil {
		return "", err
	}
	tmp := l.openAICompatFor(model)
	tmp.SetHTTPClient(l.httpClient)
	return tmp.GenerateMultimodal(ctx, prompt, images, conversationHistory)
}

// GenerateMultimodalStream implements MultimodalProvider streaming for LM Studio.
func (l *LMStudioProvider) GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	model, err := l.resolveChatModel(ctx)
	if err != nil {
		return nil, err
	}
	tmp := l.openAICompatFor(model)
	tmp.SetHTTPClient(l.httpClient)
	return tmp.GenerateMultimodalStream(ctx, prompt, images, conversationHistory)
}

// GetModel returns the model name
func (l *LMStudioProvider) GetModel() string {
	return l.Model
}

// openAIStreamChunk represents one SSE chunk from an OpenAI-compatible stream.
type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// SupportsStreaming returns true -- LM Studio supports OpenAI-compatible SSE.
func (l *LMStudioProvider) SupportsStreaming() bool { return true }

// GenerateResponseStream returns a channel of StreamTokens from LM Studio's
// OpenAI-compatible SSE streaming response.
func (l *LMStudioProvider) GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	messages := []OpenAICompatibleMessage{}
	if systemPrompt != "" {
		messages = append(messages, OpenAICompatibleMessage{Role: "system", Content: systemPrompt})
	}

	historyLimit := 10
	if len(conversationHistory) > historyLimit {
		conversationHistory = conversationHistory[len(conversationHistory)-historyLimit:]
	}
	for _, msg := range conversationHistory {
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		messages = append(messages, OpenAICompatibleMessage{Role: role, Content: msg.Content})
	}
	messages = append(messages, OpenAICompatibleMessage{Role: "user", Content: userMessage})

	model := l.Model
	if model == "" {
		availableModels, err := l.GetAvailableModels(ctx)
		if err == nil && len(availableModels) > 0 {
			model = availableModels[0]
		} else {
			return nil, fmt.Errorf("no model specified and unable to fetch available models: %w", err)
		}
	}

	request := OpenAICompatibleRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", l.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LM Studio API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamToken, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if ctx.Err() != nil {
				ch <- StreamToken{Error: ctx.Err(), Done: true}
				return
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamToken{Done: true}
				return
			}
			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- StreamToken{Content: chunk.Choices[0].Delta.Content}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamToken{Error: fmt.Errorf("scanner error: %w", err), Done: true}
			return
		}
		ch <- StreamToken{Done: true}
	}()

	return ch, nil
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
