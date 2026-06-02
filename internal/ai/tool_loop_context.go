package ai

import (
	"context"
	"strings"
)

const defaultToolLoopMaxIterations = 8

type toolLoopMaxIterationsKey struct{}

// WithToolLoopMaxIterations overrides the tool-use loop iteration cap for this context.
func WithToolLoopMaxIterations(ctx context.Context, max int) context.Context {
	if max <= 0 {
		return ctx
	}
	return context.WithValue(ctx, toolLoopMaxIterationsKey{}, max)
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
