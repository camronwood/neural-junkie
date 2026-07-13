package trace

import (
	"context"
	"testing"
)

func TestRecorderSpanNesting(t *testing.T) {
	r := NewRecorder("turn-1", "ch-1", "agent-1")
	root := r.StartSpan("turn", nil)
	child := r.StartSpan("intent_classify", map[string]any{"intent": "substantive"})
	child.End(nil)
	root.End(nil)
	tr := r.Close()
	if len(tr.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(tr.Spans))
	}
	if tr.Spans[0].Name != "intent_classify" {
		t.Fatalf("first span name=%q (child ends first)", tr.Spans[0].Name)
	}
	if tr.Spans[1].Name != "turn" {
		t.Fatalf("second span name=%q", tr.Spans[1].Name)
	}
	if tr.Spans[0].ParentID != tr.Spans[1].ID {
		t.Fatalf("child parent mismatch")
	}
	if tr.Spans[0].Attrs["intent"] != "substantive" {
		t.Fatalf("attrs not preserved")
	}
}

func TestContextPropagation(t *testing.T) {
	r := NewRecorder("turn-2", "ch", "ag")
	ctx := WithRecorder(context.Background(), r)
	h := StartSpan(ctx, "knowledge_plan", nil)
	h.End(nil)
	tr := r.Close()
	if len(tr.Spans) != 1 || tr.Spans[0].Name != "knowledge_plan" {
		t.Fatalf("unexpected spans: %+v", tr.Spans)
	}
}

func TestSpanErrorStatus(t *testing.T) {
	r := NewRecorder("turn-3", "ch", "ag")
	h := r.StartSpan("generate", nil)
	h.EndError(context.Canceled, nil)
	tr := r.Close()
	if tr.Spans[0].Status != StatusError {
		t.Fatalf("status=%q", tr.Spans[0].Status)
	}
}
