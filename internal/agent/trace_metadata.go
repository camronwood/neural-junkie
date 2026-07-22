package agent

import (
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/trace"
)

// ApplyTraceMetadataToResponse stamps the trace ID and an in-flight span
// snapshot on the response. The pipeline persists the authoritative trace only
// after delivery and the root span have closed.
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
}
