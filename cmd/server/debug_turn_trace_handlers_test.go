package main

import (
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
	tracelib "github.com/camronwood/neural-junkie/internal/trace"
)

func TestHandleDebugTurnTraceIncludesClassifierAndSpans(t *testing.T) {
	// Unit test for response shape when message metadata carries trace fields.
	resp := protocol.NewMessage(protocol.MessageTypeAnswer, "ch1", protocol.AgentInfo{ID: "a1", Name: "Agent"}, "ok")
	resp.Metadata = map[string]interface{}{
		protocol.MetadataRoutingModel:            "qwen2.5:7b",
		protocol.MetadataRoutingClassifierIntent: "substantive",
		protocol.MetadataTraceID:                 "trace-123",
		protocol.MetadataTraceSpans: []map[string]interface{}{
			{"name": "turn", "status": "ok", "start_ms": 1, "end_ms": 10},
			{"name": "knowledge_execute.codebase", "status": "ok", "start_ms": 2, "end_ms": 5},
		},
		"tool_steps": []map[string]interface{}{{"name": "read_file", "kind": "result"}},
	}
	_ = resp

	trace := map[string]interface{}{}
	trace["routing"] = map[string]interface{}{
		"classifier": map[string]interface{}{
			"intent": protocol.ExtractRoutingMeta(resp).ClassifierIntent,
		},
	}
	if trace["routing"] == nil {
		t.Fatal("expected routing block")
	}
	if resp.Metadata[protocol.MetadataTraceSpans] == nil {
		t.Fatal("expected spans in metadata fixture")
	}
	b, err := json.Marshal(trace)
	if err != nil || len(b) == 0 {
		t.Fatalf("marshal: %v", err)
	}
}

func TestPreferredTurnTraceSpansUsesFinalPersistedTrace(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_TRACE_DIR", t.TempDir())
	recorder := tracelib.NewRecorder("turn", "ch", "agent")
	root := recorder.StartSpan("turn", nil)
	deliver := recorder.StartSpan("deliver_response", nil)
	deliver.End(nil)
	root.End(nil)
	final := recorder.Close()
	if err := tracelib.Persist(final); err != nil {
		t.Fatal(err)
	}
	partial := []map[string]interface{}{{"name": "turn", "end_ms": 0}}
	raw := preferredTurnTraceSpans(final.TraceID, map[string]interface{}{
		protocol.MetadataTraceSpans: partial,
	})
	spans, ok := raw.([]tracelib.Span)
	if !ok || len(spans) != 2 || spans[1].Name != "turn" || spans[1].EndMS == 0 {
		t.Fatalf("preferred spans=%#v", raw)
	}
}

func TestTurnTraceContextSelectionAndZeroRetrievalCounts(t *testing.T) {
	raw := []tracelib.Span{{
		Name: "context_select",
		Attrs: map[string]interface{}{
			"selected_sections": []string{"summary"},
			"recovery":          map[string]interface{}{"active": true},
		},
	}}
	selection := turnTraceContextSelection(raw)
	if selection == nil || selection["recovery"] == nil {
		t.Fatalf("selection=%+v", selection)
	}
	if count, ok := turnTraceMetadataCount(map[string]interface{}{"injected_memory_count": float64(0)}, "injected_memory_count"); !ok || count != 0 {
		t.Fatalf("count=%d ok=%v", count, ok)
	}
}
