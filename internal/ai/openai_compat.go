package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// OpenAICompatProvider implements AIProvider and StreamingProvider for any
// service that speaks the OpenAI chat completions API (Amazon Bedrock gateway,
// Azure OpenAI, Together AI, Groq, Fireworks, LM Studio, etc.).
type OpenAICompatProvider struct {
	Endpoint   string
	APIKey     string
	Model      string
	Headers    map[string]string
	httpClient *http.Client
}

func NewOpenAICompatProvider(endpoint, apiKey, model string, headers map[string]string) *OpenAICompatProvider {
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1"
	}
	if model == "" {
		model = "default"
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	return &OpenAICompatProvider{
		Endpoint: strings.TrimRight(endpoint, "/"),
		APIKey:   apiKey,
		Model:    model,
		Headers:  headers,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SetHTTPClient overrides the default HTTP client (e.g. LM Studio reuse).
func (p *OpenAICompatProvider) SetHTTPClient(c *http.Client) {
	if c != nil {
		p.httpClient = c
	}
}

func (p *OpenAICompatProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := buildOpenAIChatMessages(systemPrompt, userMessage, conversationHistory, nil)

	reqBody := OpenAICompatibleRequest{
		Model:    p.Model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAICompatibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
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

func (p *OpenAICompatProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	if len(imageData) == 0 {
		return "", fmt.Errorf("empty image")
	}
	return p.GenerateMultimodal(ctx, prompt, []protocol.UserImagePart{{MIME: imageType, Data: imageData}}, conversationHistory)
}

func (p *OpenAICompatProvider) GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := buildOpenAIChatMessages(systemPrompt, userMessage, conversationHistory, images)

	reqBody := OpenAICompatibleRequest{
		Model:    p.Model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAICompatibleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
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

func (p *OpenAICompatProvider) GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := buildOpenAIChatMessages(systemPrompt, userMessage, conversationHistory, images)

	reqBody := OpenAICompatibleRequest{
		Model:    p.Model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
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
			if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "stop" {
				ch <- StreamToken{Done: true}
				return
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

func (p *OpenAICompatProvider) GetModel() string { return p.Model }

func (p *OpenAICompatProvider) SupportsStreaming() bool { return true }

func (p *OpenAICompatProvider) GenerateResponseStream(ctx context.Context, prompt string, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	return p.GenerateMultimodalStream(ctx, prompt, nil, conversationHistory)
}

func (p *OpenAICompatProvider) GetEndpoint() string  { return p.Endpoint }
func (p *OpenAICompatProvider) SetModel(model string) { p.Model = model }
func (p *OpenAICompatProvider) SetEndpoint(endpoint string) {
	p.Endpoint = strings.TrimRight(endpoint, "/")
}

func (p *OpenAICompatProvider) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Endpoint+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

func (p *OpenAICompatProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Endpoint+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	models := make([]string, len(response.Data))
	for i, m := range response.Data {
		models[i] = m.ID
	}
	return models, nil
}
