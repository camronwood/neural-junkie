package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ErrNativeToolsUnsupported is returned when an OpenAI-compat endpoint rejects tool calling.
var ErrNativeToolsUnsupported = errors.New("native tools unsupported")

// SupportsTools reports whether this OpenAI-compat provider can run tool-use loops.
func (p *OpenAICompatProvider) SupportsTools() bool {
	return p != nil && !p.nativeToolsUnsupported
}

// NativeToolsMarkedUnsupported reports whether native tool calling was rejected at runtime.
func (p *OpenAICompatProvider) NativeToolsMarkedUnsupported() bool {
	return p != nil && p.nativeToolsUnsupported
}

// MarkNativeToolsUnsupported records that this endpoint/model rejected tool calls.
func (p *OpenAICompatProvider) MarkNativeToolsUnsupported() {
	if p != nil {
		p.nativeToolsUnsupported = true
	}
}

func claudeToolsToOpenAI(tools []ClaudeToolDefinition) []OpenAITool {
	out := make([]OpenAITool, 0, len(tools))
	for _, t := range tools {
		params := t.InputSchema
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return out
}

// GenerateResponseWithTools runs an OpenAI Chat Completions loop with tool calling.
func (p *OpenAICompatProvider) GenerateResponseWithTools(
	ctx context.Context,
	prompt string,
	conversationHistory []protocol.Message,
	tools []ClaudeToolDefinition,
	onToolUse ToolUseCallback,
) (string, error) {
	if len(tools) == 0 {
		return p.GenerateResponse(ctx, prompt, conversationHistory)
	}
	if p.nativeToolsUnsupported {
		return "", ErrNativeToolsUnsupported
	}

	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := buildOpenAIChatMessages(systemPrompt, userMessage, conversationHistory, nil)
	openAITools := claudeToolsToOpenAI(tools)

	maxIter := ToolLoopMaxIterationsFromContext(ctx)
	for iter := 0; iter < maxIter; iter++ {
		reqBody := OpenAICompatibleRequest{
			Model:      p.Model,
			Messages:   messages,
			Stream:     false,
			Tools:      openAITools,
			ToolChoice: "auto",
		}

		body, status, err := p.doChatCompletions(ctx, reqBody)
		if err != nil {
			return "", err
		}
		if status != http.StatusOK {
			if openAIToolsUnsupported(status, body) {
				p.MarkNativeToolsUnsupported()
				log.Printf("OpenAI-compat model %q does not support native tool calling", p.Model)
				return "", ErrNativeToolsUnsupported
			}
			return "", fmt.Errorf("OpenAI-compat API status %d: %s", status, string(body))
		}

		var response OpenAICompatibleResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("decode openai response: %w", err)
		}
		if response.Error != nil {
			return "", fmt.Errorf("OpenAI-compat API error: %s", response.Error.Message)
		}
		if len(response.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		recordOpenAICompatUsage(&p.usage, response.Usage.PromptTokens, response.Usage.CompletionTokens)

		choice := response.Choices[0]
		msg := choice.Message
		if len(msg.ToolCalls) == 0 {
			text := strings.TrimSpace(openAIMessageTextContent(msg.Content))
			if text == "" {
				return "", fmt.Errorf("no content in response")
			}
			return text, nil
		}

		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			input := tc.Function.Arguments
			if len(input) == 0 {
				input = json.RawMessage(`{}`)
			}
			emitToolStep(ctx, ToolStepEvent{
				Kind: "start", Name: name, Iteration: iter + 1, MaxIterations: maxIter,
			})
			result, err := onToolUse(ctx, ToolUseRequest{ID: tc.ID, Name: name, Input: input})
			if err != nil {
				result = "ERROR: " + err.Error()
				emitToolStep(ctx, ToolStepEvent{
					Kind: "error", Name: name, Iteration: iter + 1, MaxIterations: maxIter, Preview: result,
				})
			} else {
				preview := result
				if len(preview) > 200 {
					preview = preview[:200] + "…"
				}
				emitToolStep(ctx, ToolStepEvent{
					Kind: "result", Name: name, Iteration: iter + 1, MaxIterations: maxIter, Preview: preview,
				})
			}
			toolCallID := tc.ID
			if toolCallID == "" {
				toolCallID = name
			}
			messages = append(messages, OpenAICompatibleMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    result,
			})
		}
	}

	return "", fmt.Errorf("openai tool loop exceeded %d iterations", maxIter)
}

func (p *OpenAICompatProvider) doChatCompletions(ctx context.Context, reqBody OpenAICompatibleRequest) ([]byte, int, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
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
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func openAIToolsUnsupported(status int, body []byte) bool {
	if status != http.StatusBadRequest && status != http.StatusUnprocessableEntity {
		return false
	}
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "does not support tools") ||
		strings.Contains(lower, "does not support tool") ||
		strings.Contains(lower, "tool calling") && strings.Contains(lower, "not supported") ||
		strings.Contains(lower, "unknown field") && strings.Contains(lower, "tools") ||
		strings.Contains(lower, "invalid") && strings.Contains(lower, "tools")
}

// Ensure OpenAICompatProvider implements ToolCapableProvider.
var _ ToolCapableProvider = (*OpenAICompatProvider)(nil)
