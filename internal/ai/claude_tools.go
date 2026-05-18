package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

const maxToolLoopIterations = 8

// claudeToolResponseContent supports text and tool_use blocks from Claude.
type claudeToolResponseContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type claudeToolResponse struct {
	ID         string                      `json:"id"`
	Role       string                      `json:"role"`
	Content    []claudeToolResponseContent `json:"content"`
	StopReason string                      `json:"stop_reason"`
}

// SupportsTools reports whether this provider can run tool-use loops.
func (c *ClaudeProvider) SupportsTools() bool {
	return true
}

// GenerateResponseWithTools runs a Claude messages loop with tool_use handling.
func (c *ClaudeProvider) GenerateResponseWithTools(
	ctx context.Context,
	prompt string,
	conversationHistory []protocol.Message,
	tools []ClaudeToolDefinition,
	onToolUse ToolUseCallback,
) (string, error) {
	if len(tools) == 0 {
		return c.GenerateResponse(ctx, prompt, conversationHistory)
	}
	if c.UseAIHub {
		return "", fmt.Errorf("MCP tools require direct Anthropic API (USE_AI_HUB=false)")
	}

	systemPrompt, userMessage := SplitSystemPrompt(prompt)

	messages := []ClaudeMessage{}
	for i, msg := range conversationHistory {
		if i >= 10 {
			break
		}
		messages = append(messages, ClaudeMessage{
			Role:    ChatRoleForHistory(msg),
			Content: msg.Content,
		})
	}
	messages = append(messages, ClaudeMessage{Role: "user", Content: userMessage})

	maxTokens := 1024
	if isDeepAnalysisPrompt(userMessage) {
		maxTokens = 4096
	}

	for iter := 0; iter < maxToolLoopIterations; iter++ {
		reqBody := map[string]any{
			"model":      c.Model,
			"max_tokens": maxTokens,
			"messages":   messages,
			"tools":      tools,
		}
		if systemPrompt != "" {
			reqBody["system"] = systemPrompt
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/messages", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var response claudeToolResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("decode response: %w", err)
		}

		if response.StopReason != "tool_use" {
			var parts []string
			for _, block := range response.Content {
				if block.Type == "text" && block.Text != "" {
					parts = append(parts, block.Text)
				}
			}
			if len(parts) == 0 {
				return "", fmt.Errorf("no text in Claude response")
			}
			return joinStrings(parts), nil
		}

		// Append assistant message with full content blocks
		assistantContent := make([]claudeToolResponseContent, len(response.Content))
		copy(assistantContent, response.Content)
		messages = append(messages, ClaudeMessage{
			Role:    "assistant",
			Content: assistantContent,
		})

		// Execute each tool_use and collect tool_result blocks
		var toolResults []map[string]any
		for _, block := range response.Content {
			if block.Type != "tool_use" {
				continue
			}
			resultText, err := onToolUse(ctx, ToolUseRequest{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
			if err != nil {
				resultText = fmt.Sprintf("Tool error: %v", err)
			}
			toolResults = append(toolResults, map[string]any{
				"type":        "tool_result",
				"tool_use_id": block.ID,
				"content":     resultText,
			})
		}

		if len(toolResults) == 0 {
			return "", fmt.Errorf("stop_reason tool_use but no tool_use blocks")
		}

		messages = append(messages, ClaudeMessage{
			Role:    "user",
			Content: toolResults,
		})
	}

	return "", fmt.Errorf("exceeded maximum tool loop iterations (%d)", maxToolLoopIterations)
}

func joinStrings(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "\n" + parts[i]
	}
	return out
}
