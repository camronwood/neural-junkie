package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusOK    = "ok"
	StatusError = "error"
)

// Span is one timed operation within a trace.
type Span struct {
	ID       string         `json:"id"`
	ParentID string         `json:"parent_id,omitempty"`
	Name     string         `json:"name"`
	StartMS  int64          `json:"start_ms"`
	EndMS    int64          `json:"end_ms"`
	Status   string         `json:"status"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

// Trace is a tree of spans for one agent turn.
type Trace struct {
	TraceID   string `json:"trace_id"`
	TurnID    string `json:"turn_id"`
	ChannelID string `json:"channel_id"`
	AgentID   string `json:"agent_id"`
	Spans     []Span `json:"spans"`
}

type activeSpan struct {
	span Span
}

// Recorder collects spans for a single turn.
type Recorder struct {
	mu     sync.Mutex
	trace  Trace
	stack  []*activeSpan
	closed bool
}

// NewRecorder starts a trace for a turn.
func NewRecorder(turnID, channelID, agentID string) *Recorder {
	return &Recorder{
		trace: Trace{
			TraceID:   uuid.New().String(),
			TurnID:    turnID,
			ChannelID: channelID,
			AgentID:   agentID,
		},
	}
}

// StartSpan opens a child span. End with (*SpanHandle).End or EndError.
func (r *Recorder) StartSpan(name string, attrs map[string]any) *SpanHandle {
	if r == nil {
		return &SpanHandle{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return &SpanHandle{}
	}
	now := time.Now().UnixMilli()
	span := Span{
		ID:      uuid.New().String(),
		Name:    name,
		StartMS: now,
		Status:  StatusOK,
		Attrs:   copyAttrs(attrs),
	}
	if len(r.stack) > 0 {
		span.ParentID = r.stack[len(r.stack)-1].span.ID
	}
	as := &activeSpan{span: span}
	r.stack = append(r.stack, as)
	return &SpanHandle{recorder: r, span: as}
}

// End closes the root turn span recorder.
func (r *Recorder) Close() Trace {
	if r == nil {
		return Trace{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	for len(r.stack) > 0 {
		r.popSpan(StatusOK)
	}
	return r.trace
}

func (r *Recorder) popSpan(status string) {
	if len(r.stack) == 0 {
		return
	}
	as := r.stack[len(r.stack)-1]
	r.stack = r.stack[:len(r.stack)-1]
	as.span.EndMS = time.Now().UnixMilli()
	as.span.Status = status
	r.trace.Spans = append(r.trace.Spans, as.span)
}

// Snapshot returns a copy of the trace so far.
func (r *Recorder) Snapshot() Trace {
	if r == nil {
		return Trace{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.trace
	out.Spans = append([]Span(nil), r.trace.Spans...)
	for _, as := range r.stack {
		s := as.span
		if s.EndMS == 0 {
			s.EndMS = time.Now().UnixMilli()
		}
		out.Spans = append(out.Spans, s)
	}
	return out
}

// SpanHandle references an in-flight span.
type SpanHandle struct {
	recorder *Recorder
	span     *activeSpan
	ended    bool
}

// Annotate merges attributes into a span, including after it has ended.
// This supports observations (for example prompt compression) that become
// available after the operation which selected the context.
func (h *SpanHandle) Annotate(attrs map[string]any) {
	if h == nil || h.recorder == nil || h.span == nil || len(attrs) == 0 {
		return
	}
	h.recorder.mu.Lock()
	defer h.recorder.mu.Unlock()
	for k, v := range attrs {
		if h.span.span.Attrs == nil {
			h.span.span.Attrs = map[string]any{}
		}
		h.span.span.Attrs[k] = v
	}
	for i := range h.recorder.trace.Spans {
		if h.recorder.trace.Spans[i].ID == h.span.span.ID {
			h.recorder.trace.Spans[i].Attrs = copyAttrs(h.span.span.Attrs)
			break
		}
	}
}

// End marks the span successful.
func (h *SpanHandle) End(attrs map[string]any) {
	h.end(StatusOK, attrs)
}

// EndError marks the span failed.
func (h *SpanHandle) EndError(err error, attrs map[string]any) {
	if attrs == nil {
		attrs = map[string]any{}
	}
	if err != nil {
		attrs["error"] = err.Error()
	}
	h.end(StatusError, attrs)
}

func (h *SpanHandle) end(status string, attrs map[string]any) {
	if h == nil || h.recorder == nil || h.span == nil || h.ended {
		return
	}
	h.ended = true
	h.recorder.mu.Lock()
	defer h.recorder.mu.Unlock()
	if len(h.recorder.stack) == 0 || h.recorder.stack[len(h.recorder.stack)-1] != h.span {
		return
	}
	for k, v := range attrs {
		if h.span.span.Attrs == nil {
			h.span.span.Attrs = map[string]any{}
		}
		h.span.span.Attrs[k] = v
	}
	h.recorder.popSpan(status)
}

func copyAttrs(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

// DefaultDir returns ~/.neural-junkie/traces
func DefaultDir() string {
	if configured := os.Getenv("NEURAL_JUNKIE_TRACE_DIR"); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "traces"
	}
	return filepath.Join(home, ".neural-junkie", "traces")
}

// Persist writes a trace JSON file.
func Persist(t Trace) error {
	if t.TraceID == "" {
		return nil
	}
	dir := DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, t.TraceID+".json")
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a persisted trace by ID.
func Load(traceID string) (Trace, error) {
	path := filepath.Join(DefaultDir(), traceID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Trace{}, err
	}
	var t Trace
	if err := json.Unmarshal(data, &t); err != nil {
		return Trace{}, err
	}
	return t, nil
}
