package ai

import (
	"context"
	"encoding/json"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// ClaudeToolDefinition is an Anthropic Messages API tool definition.
type ClaudeToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolUseRequest is passed to the tool execution callback when Claude requests a tool.
type ToolUseRequest struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolUseCallback executes a tool and returns the result text for the model.
type ToolUseCallback func(ctx context.Context, req ToolUseRequest) (string, error)

// ToolCapableProvider supports Claude tool-use loops.
type ToolCapableProvider interface {
	AIProvider
	SupportsTools() bool
	GenerateResponseWithTools(
		ctx context.Context,
		prompt string,
		conversationHistory []protocol.Message,
		tools []ClaudeToolDefinition,
		onToolUse ToolUseCallback,
	) (string, error)
}
