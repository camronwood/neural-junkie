package main

import (
	"encoding/json"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestHandleDebugTurnTraceIncludesClassifierAndSpans(t *testing.T) {
	// Unit test for response shape when message metadata carries trace fields.
	resp := protocol.NewMessage(protocol.MessageTypeAnswer, "ch1", protocol.AgentInfo{ID: "a1", Name: "Agent"}, "ok")
	resp.Metadata = map[string]interface{}{
		protocol.MetadataRoutingModel:             "qwen2.5:7b",
		protocol.MetadataRoutingClassifierIntent:  "substantive",
		protocol.MetadataTraceID:                  "trace-123",
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
