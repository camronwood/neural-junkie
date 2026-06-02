package ai

import "context"

// ToolStepEvent describes progress during a tool-use loop.
type ToolStepEvent struct {
	Kind          string // start | result | error
	Name          string
	Iteration     int
	MaxIterations int
	Preview       string
}

type toolStepObserverKey struct{}

// WithToolStepObserver attaches a callback invoked for each tool step in a tool loop.
func WithToolStepObserver(ctx context.Context, fn func(ToolStepEvent)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, toolStepObserverKey{}, fn)
}

// ToolStepObserverFromContext returns the observer if present.
func ToolStepObserverFromContext(ctx context.Context) func(ToolStepEvent) {
	if ctx == nil {
		return nil
	}
	fn, _ := ctx.Value(toolStepObserverKey{}).(func(ToolStepEvent))
	return fn
}

func emitToolStep(ctx context.Context, ev ToolStepEvent) {
	if fn := ToolStepObserverFromContext(ctx); fn != nil {
		fn(ev)
	}
}
