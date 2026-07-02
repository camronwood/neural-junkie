package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ErrReActToolLoopExceeded is returned when the ReAct loop hits the iteration cap.
var ErrReActToolLoopExceeded = errors.New("react tool loop exceeded max iterations")

// ReActToolProvider runs tool loops via prompt + text parsing when native tool calling is unavailable.
type ReActToolProvider struct {
	Inner AIProvider
}

// NewReActToolProvider wraps a chat provider with ReAct-style tool invocation.
func NewReActToolProvider(inner AIProvider) *ReActToolProvider {
	if inner == nil {
		return nil
	}
	return &ReActToolProvider{Inner: inner}
}

func (r *ReActToolProvider) SupportsTools() bool {
	return r != nil && r.Inner != nil
}

func (r *ReActToolProvider) GetModel() string {
	if r == nil || r.Inner == nil {
		return ""
	}
	return r.Inner.GetModel()
}

func (r *ReActToolProvider) GenerateResponse(ctx context.Context, prompt string, conversationHistory []protocol.Message) (string, error) {
	return r.Inner.GenerateResponse(ctx, prompt, conversationHistory)
}

func (r *ReActToolProvider) GenerateVisionResponse(ctx context.Context, prompt string, imageData []byte, imageType string, conversationHistory []protocol.Message) (string, error) {
	return r.Inner.GenerateVisionResponse(ctx, prompt, imageData, imageType, conversationHistory)
}

// GenerateResponseWithTools runs a ReAct loop: model emits <tool_call> JSON, hub executes, observation fed back.
func (r *ReActToolProvider) GenerateResponseWithTools(
	ctx context.Context,
	prompt string,
	conversationHistory []protocol.Message,
	tools []ClaudeToolDefinition,
	onToolUse ToolUseCallback,
) (string, error) {
	if r == nil || r.Inner == nil {
		return "", fmt.Errorf("react tool provider: nil inner")
	}
	if len(tools) == 0 {
		return r.Inner.GenerateResponse(ctx, prompt, conversationHistory)
	}

	reactPrompt := prompt + reactToolSystemSuffix(tools)
	loopSuffix := ""
	maxIter := ToolLoopMaxIterationsFromContext(ctx)

	for iter := 0; iter < maxIter; iter++ {
		fullPrompt := reactPrompt + loopSuffix
		text, err := r.Inner.GenerateResponse(ctx, fullPrompt, conversationHistory)
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		name, input, ok := ParseToolCallFromText(text)
		if !ok {
			return StripToolCallFromText(text), nil
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

		loopSuffix += "\n\n" + formatReActObservation(name, result) +
			"\n\nContinue: call another tool with <tool_call> or reply without a tool block."
	}

	return "", fmt.Errorf("%w (%d)", ErrReActToolLoopExceeded, maxIter)
}

func formatReActObservation(toolName, result string) string {
	return "=== TOOL RESULT (" + strings.TrimSpace(toolName) + ") ===\n" + strings.TrimSpace(result)
}

func reactToolSystemSuffix(tools []ClaudeToolDefinition) string {
	var b strings.Builder
	b.WriteString("\n\n---REACT_TOOLS---\n")
	b.WriteString("You can call tools to gather data or act on the workspace.\n")
	b.WriteString("To call a tool, emit exactly one block on its own line:\n")
	b.WriteString(`<tool_call>{"name":"TOOL_NAME","arguments":{...}}</tool_call>`)
	b.WriteString("\nUse \"arguments\" for parameters. When you have enough information, reply normally with no <tool_call> block.\n\n")
	b.WriteString("Available tools:\n")
	for _, t := range tools {
		b.WriteString("- ")
		b.WriteString(t.Name)
		if desc := strings.TrimSpace(t.Description); desc != "" {
			b.WriteString(": ")
			b.WriteString(desc)
		}
		b.WriteByte('\n')
		if len(t.InputSchema) > 0 {
			schema := strings.TrimSpace(string(t.InputSchema))
			if len(schema) > 400 {
				schema = schema[:400] + "…"
			}
			b.WriteString("  schema: ")
			b.WriteString(schema)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// IsReActToolLoopError reports whether err should trigger Qwen swap fallback.
func IsReActToolLoopError(err error) bool {
	return errors.Is(err, ErrReActToolLoopExceeded)
}

// ReActToolLoopError unwraps iteration cap errors for tests.
func ReActToolLoopError(max int) error {
	return fmt.Errorf("%w (%d)", ErrReActToolLoopExceeded, max)
}

// Ensure ReActToolProvider implements ToolCapableProvider.
var _ ToolCapableProvider = (*ReActToolProvider)(nil)
