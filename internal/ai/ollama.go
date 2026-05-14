package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
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
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // base64-encoded raw image bytes (vision models)
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

// GenerateVisionResponse uses vision-capable Ollama models (e.g. llava).
func (o *OllamaProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	if len(imageData) == 0 {
		return "", fmt.Errorf("empty image")
	}
	return o.GenerateMultimodal(ctx, prompt, []protocol.UserImagePart{{MIME: imageType, Data: imageData}}, conversationHistory)
}

// GenerateMultimodal sends images on the final user turn (Ollama /api/chat).
func (o *OllamaProvider) GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	messages := []OllamaMessage{}
	if systemPrompt != "" {
		messages = append(messages, OllamaMessage{Role: "system", Content: systemPrompt})
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
		messages = append(messages, OllamaMessage{Role: role, Content: msg.Content})
	}
	var imgB64 []string
	for _, im := range images {
		if len(im.Data) == 0 {
			continue
		}
		imgB64 = append(imgB64, base64.StdEncoding.EncodeToString(im.Data))
	}
	last := OllamaMessage{Role: "user", Content: userMessage}
	if len(imgB64) > 0 {
		last.Images = imgB64
	}
	messages = append(messages, last)

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

// GenerateMultimodalStream runs a non-streaming multimodal request and emits the full reply as one chunk.
func (o *OllamaProvider) GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	ch := make(chan StreamToken, 2)
	go func() {
		defer close(ch)
		text, err := o.GenerateMultimodal(ctx, prompt, images, conversationHistory)
		if err != nil {
			ch <- StreamToken{Error: err, Done: true}
			return
		}
		if text != "" {
			ch <- StreamToken{Content: text}
		}
		ch <- StreamToken{Done: true}
	}()
	return ch, nil
}

// GetModel returns the model name
func (o *OllamaProvider) GetModel() string {
	return o.Model
}

// SupportsStreaming returns true -- Ollama natively streams NDJSON.
func (o *OllamaProvider) SupportsStreaming() bool { return true }

// GenerateResponseStream returns a channel of StreamTokens, each carrying
// a text chunk from Ollama's NDJSON streaming response.
func (o *OllamaProvider) GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	messages := []OllamaMessage{}
	if systemPrompt != "" {
		messages = append(messages, OllamaMessage{Role: "system", Content: systemPrompt})
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
		messages = append(messages, OllamaMessage{Role: role, Content: msg.Content})
	}
	messages = append(messages, OllamaMessage{Role: "user", Content: userMessage})

	request := OllamaRequest{
		Model:    o.Model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/chat", bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("Ollama API request failed with status %d: %s", resp.StatusCode, string(body))
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
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var chunk OllamaResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				ch <- StreamToken{Error: fmt.Errorf("failed to decode chunk: %w", err), Done: true}
				return
			}
			if chunk.Error != "" {
				ch <- StreamToken{Error: fmt.Errorf("Ollama error: %s", chunk.Error), Done: true}
				return
			}
			if chunk.Message.Content != "" {
				ch <- StreamToken{Content: chunk.Message.Content}
			}
			if chunk.Done {
				ch <- StreamToken{Done: true}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamToken{Error: fmt.Errorf("scanner error: %w", err), Done: true}
		}
	}()

	return ch, nil
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
