package agent

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/trace"
)

// ApplyTraceMetadataToResponse stamps trace_id, spans, and tool_steps on the response.
func (a *Agent) ApplyTraceMetadataToResponse(msg *protocol.Message, recorder *trace.Recorder, toolSteps []map[string]interface{}) {
	if a == nil || msg == nil {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	if len(toolSteps) > 0 {
		msg.Metadata["tool_steps"] = toolSteps
	}
	if recorder == nil {
		return
	}
	tr := recorder.Snapshot()
	if tr.TraceID != "" {
		msg.Metadata[protocol.MetadataTraceID] = tr.TraceID
	}
	if len(tr.Spans) > 0 {
		msg.Metadata[protocol.MetadataTraceSpans] = tr.Spans
	}
	_ = trace.Persist(tr)
}
