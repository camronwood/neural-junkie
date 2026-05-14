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
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func (c *ClaudeProvider) appendClaudeHistory(messages *[]ClaudeMessage, conversationHistory []protocol.Message, max int) {
	for _, msg := range conversationHistory {
		if len(*messages) >= max {
			break
		}
		role := "user"
		if msg.From.Type != protocol.AgentTypeGeneral {
			role = "assistant"
		}
		*messages = append(*messages, ClaudeMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
}

// GenerateMultimodal sends the current user turn with one or more images (Claude Messages API).
func (c *ClaudeProvider) GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := []ClaudeMessage{}
	c.appendClaudeHistory(&messages, conversationHistory, 10)

	var blocks []ClaudeContentBlock
	if strings.TrimSpace(userMessage) != "" {
		blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: userMessage})
	}
	for _, im := range images {
		mime := im.MIME
		if mime == "" {
			mime = "image/png"
		}
		b64 := base64.StdEncoding.EncodeToString(im.Data)
		blocks = append(blocks, ClaudeContentBlock{
			Type: "image",
			Source: &ClaudeImageSource{
				Type:      "base64",
				MediaType: mime,
				Data:      b64,
			},
		})
	}
	if len(blocks) == 0 {
		return "", fmt.Errorf("no text or images for multimodal request")
	}
	messages = append(messages, ClaudeMessage{Role: "user", Content: blocks})

	maxTokens := 1024
	if isDeepAnalysisPrompt(userMessage) {
		maxTokens = 4096
	}
	if len(images) > 0 {
		maxTokens = intMax(maxTokens, 2000)
	}

	request := ClaudeRequest{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    systemPrompt,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.UseAIHub {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return response.Content[0].Text, nil
}

// GenerateMultimodalStream streams a multimodal Claude response (same SSE path as text streaming).
func (c *ClaudeProvider) GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error) {
	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := []ClaudeMessage{}
	c.appendClaudeHistory(&messages, conversationHistory, 10)

	var blocks []ClaudeContentBlock
	if strings.TrimSpace(userMessage) != "" {
		blocks = append(blocks, ClaudeContentBlock{Type: "text", Text: userMessage})
	}
	for _, im := range images {
		mime := im.MIME
		if mime == "" {
			mime = "image/png"
		}
		b64 := base64.StdEncoding.EncodeToString(im.Data)
		blocks = append(blocks, ClaudeContentBlock{
			Type: "image",
			Source: &ClaudeImageSource{
				Type:      "base64",
				MediaType: mime,
				Data:      b64,
			},
		})
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no text or images for multimodal request")
	}
	messages = append(messages, ClaudeMessage{Role: "user", Content: blocks})

	maxTokens := 1024
	if isDeepAnalysisPrompt(userMessage) {
		maxTokens = 4096
	}
	if len(images) > 0 {
		maxTokens = intMax(maxTokens, 2000)
	}

	request := ClaudeRequest{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages:  messages,
		System:    systemPrompt,
		Stream:    true,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.UseAIHub {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
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

			var event claudeSSEEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					ch <- StreamToken{Content: event.Delta.Text}
				}
			case "message_stop":
				ch <- StreamToken{Done: true}
				return
			case "error":
				ch <- StreamToken{Error: fmt.Errorf("Claude stream error"), Done: true}
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

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
