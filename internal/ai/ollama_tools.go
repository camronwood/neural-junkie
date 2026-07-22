package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// SupportsTools reports whether this Ollama provider can run tool-use loops.
func (o *OllamaProvider) SupportsTools() bool {
	o.ensureNativeToolsKnown()
	return o.nativeToolsSupported
}

func claudeToolsToOllama(tools []ClaudeToolDefinition) []OllamaTool {
	out := make([]OllamaTool, 0, len(tools))
	for _, t := range tools {
		params := t.InputSchema
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, OllamaTool{
			Type: "function",
			Function: OllamaToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return out
}

// GenerateResponseWithTools runs an Ollama chat loop with tool calling.
func (o *OllamaProvider) GenerateResponseWithTools(
	ctx context.Context,
	prompt string,
	conversationHistory []protocol.Message,
	tools []ClaudeToolDefinition,
	onToolUse ToolUseCallback,
) (string, error) {
	if len(tools) == 0 {
		return o.GenerateResponse(ctx, prompt, conversationHistory)
	}
	o.ensureNativeToolsKnown()
	if !o.nativeToolsSupported {
		return o.GenerateResponse(ctx, prompt, conversationHistory)
	}

	systemPrompt, userMessage := SplitSystemPrompt(prompt)
	messages := o.buildChatMessages(systemPrompt, userMessage, conversationHistory)
	ollamaTools := claudeToolsToOllama(tools)

	maxIter := ToolLoopMaxIterationsFromContext(ctx)
	for iter := 0; iter < maxIter; iter++ {
		reqBody := o.newChatRequest(messages, false)
		reqBody.Tools = ollamaTools

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("marshal ollama request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/chat", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := o.httpClient.Do(httpReq)
		if err != nil {
			return "", fmt.Errorf("ollama chat: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if resp.StatusCode != http.StatusOK {
			if ollamaToolsUnsupported(resp.StatusCode, body) {
				o.MarkNativeToolsUnsupported()
				log.Printf("Ollama model %q does not support native tool calling; using plain chat", o.Model)
				return o.GenerateResponse(ctx, prompt, conversationHistory)
			}
			if ollamaToolCallMalformed(resp.StatusCode, body) {
				o.MarkNativeToolsUnsupported()
				log.Printf("Ollama model %q returned malformed tool-call XML; falling back from native tools", o.Model)
				return "", fmt.Errorf("%w: %s", ErrNativeToolsUnsupported, truncateOllamaErrBody(body))
			}
			return "", fmt.Errorf("Ollama API status %d: %s", resp.StatusCode, string(body))
		}

		var chatResp OllamaResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return "", fmt.Errorf("decode ollama response: %w", err)
		}
		if chatResp.Error != "" {
			if ollamaToolCallMalformed(http.StatusOK, []byte(chatResp.Error)) {
				o.MarkNativeToolsUnsupported()
				log.Printf("Ollama model %q returned malformed tool-call XML; falling back from native tools", o.Model)
				return "", fmt.Errorf("%w: %s", ErrNativeToolsUnsupported, chatResp.Error)
			}
			return "", fmt.Errorf("Ollama API error: %s", chatResp.Error)
		}

		msg := chatResp.Message
		if len(msg.ToolCalls) == 0 {
			return ollamaFinalizeContent(msg.Content, msg.Thinking)
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
			result, err := onToolUse(ctx, ToolUseRequest{Name: name, Input: input})
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
			messages = append(messages, OllamaMessage{
				Role:     "tool",
				ToolName: name,
				Content:  result,
			})
		}
	}

	return "", fmt.Errorf("ollama tool loop exceeded %d iterations", maxIter)
}

func ollamaToolsUnsupported(status int, body []byte) bool {
	if status != http.StatusBadRequest {
		return false
	}
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "does not support tools") ||
		strings.Contains(lower, "does not support tool")
}

// ollamaToolCallMalformed detects Ollama 500s (and similar) where the model emitted
// broken tool-call XML such as "<function> closed by </parameter>".
func ollamaToolCallMalformed(status int, body []byte) bool {
	lower := strings.ToLower(string(body))
	if !strings.Contains(lower, "xml syntax") &&
		!strings.Contains(lower, "closed by") &&
		!strings.Contains(lower, "malformed") {
		return false
	}
	return strings.Contains(lower, "function") ||
		strings.Contains(lower, "parameter") ||
		strings.Contains(lower, "tool") ||
		status == http.StatusInternalServerError
}

func truncateOllamaErrBody(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 240 {
		return s[:240] + "…"
	}
	return s
}
