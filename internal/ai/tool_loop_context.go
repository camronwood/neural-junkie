package ai

import (
	"context"
	"strings"
)

// defaultToolLoopMaxIterations is the chat-path tool-use cap when no override is set.
// Inspect/debug turns routinely need more than a handful of read_file/list_dir steps.
const defaultToolLoopMaxIterations = 24

type toolLoopMaxIterationsKey struct{}

// WithToolLoopMaxIterations overrides the tool-use loop iteration cap for this context.
func WithToolLoopMaxIterations(ctx context.Context, max int) context.Context {
	if max <= 0 {
		return ctx
	}
	return context.WithValue(ctx, toolLoopMaxIterationsKey{}, max)
}

// EnsureToolLoopMaxIterations sets the cap only when the context has no explicit override.
func EnsureToolLoopMaxIterations(ctx context.Context, max int) context.Context {
	if ctx != nil {
		if v, ok := ctx.Value(toolLoopMaxIterationsKey{}).(int); ok && v > 0 {
			return ctx
		}
	}
	return WithToolLoopMaxIterations(ctx, max)
}

// ToolLoopMaxIterationsFromContext returns the configured cap or defaultToolLoopMaxIterations.
func ToolLoopMaxIterationsFromContext(ctx context.Context) int {
	if ctx == nil {
		return defaultToolLoopMaxIterations
	}
	if v, ok := ctx.Value(toolLoopMaxIterationsKey{}).(int); ok && v > 0 {
		return v
	}
	return defaultToolLoopMaxIterations
}

type implementationToolModelKey struct{}

// WithImplementationToolModel sets the Ollama model tag for tool loops during implementation sessions.
func WithImplementationToolModel(ctx context.Context, model string) context.Context {
	model = strings.TrimSpace(model)
	if model == "" {
		return ctx
	}
	return context.WithValue(ctx, implementationToolModelKey{}, model)
}

// ImplementationToolModelFromContext returns a non-empty tool-loop model override when set.
func ImplementationToolModelFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(implementationToolModelKey{}).(string)
	return strings.TrimSpace(v)
}
