package trace

import "context"

type ctxKey struct{}

// WithRecorder attaches a trace recorder to ctx.
func WithRecorder(ctx context.Context, r *Recorder) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, r)
}

// FromContext returns the recorder for this turn, if any.
func FromContext(ctx context.Context) *Recorder {
	if ctx == nil {
		return nil
	}
	r, _ := ctx.Value(ctxKey{}).(*Recorder)
	return r
}

// StartSpan starts a span using the recorder from ctx, if present.
func StartSpan(ctx context.Context, name string, attrs map[string]any) *SpanHandle {
	if r := FromContext(ctx); r != nil {
		return r.StartSpan(name, attrs)
	}
	return &SpanHandle{}
}
