package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
	tracelib "github.com/camronwood/neural-junkie/internal/trace"
)

func handleDebugTurnTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	messageID := strings.TrimSpace(r.URL.Query().Get("message_id"))
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	var msgs []*protocol.Message
	if channel != "" {
		if loaded, err := chatHub.GetMessages(channel, 200); err == nil {
			msgs = loaded
		}
	}

	if query == "" && messageID != "" {
		for _, m := range msgs {
			if m != nil && m.ID == messageID {
				query = m.Content
				break
			}
		}
	}

	trace := map[string]interface{}{
		"message_id": messageID,
		"channel":    channel,
		"query":      query,
	}

	var target *protocol.Message
	for _, m := range msgs {
		if m != nil && m.ID == messageID {
			target = m
			break
		}
	}
	if target != nil && !turnTraceHasRoutingMeta(target) {
		for _, m := range msgs {
			if m != nil && m.ReplyTo == target.ID {
				target = m
				trace["reply_message_id"] = m.ID
				break
			}
		}
	}
	if target == nil && messageID != "" {
		for _, m := range msgs {
			if m != nil && m.ReplyTo == messageID {
				target = m
				trace["reply_message_id"] = m.ID
				break
			}
		}
	}

	if target != nil {
		meta := protocol.ExtractRoutingMeta(target)
		trace["routing"] = map[string]interface{}{
			"model":       meta.Model,
			"tool_model":  meta.ToolModel,
			"provider_id": meta.ProviderID,
			"domain":      meta.Domain,
			"cost_tier":   meta.CostTier,
			"reason":      meta.Reason,
			"source":      meta.Source,
			"classifier": map[string]interface{}{
				"intent":     meta.ClassifierIntent,
				"tool_need":  meta.ClassifierToolNeed,
				"confidence": meta.ClassifierConfidence,
				"lora_tag":   meta.ClassifierLoRATag,
			},
			"attempts":         meta.Attempts,
			"failure_evidence": meta.FailureEvidence,
		}
		trace["retrieval"] = map[string]interface{}{
			"mode":     meta.KnowledgeRoute,
			"reason":   meta.KnowledgeReason,
			"targets":  meta.KnowledgeTargets,
			"executed": meta.KnowledgeExecuted,
		}
		if target.Metadata != nil {
			if n, ok := turnTraceMetadataCount(target.Metadata, "injected_memory_count"); ok {
				trace["retrieval"].(map[string]interface{})["memory_count"] = n
			}
			if n, ok := turnTraceMetadataCount(target.Metadata, "injected_codebase_count"); ok {
				trace["retrieval"].(map[string]interface{})["codebase_count"] = n
			}
		}
		if meta.KnowledgeRoute == "" && query != "" {
			plan := routing.PlanKnowledgeRoute(query)
			trace["plan"] = plan
			trace["retrieval"] = map[string]interface{}{
				"mode":    string(plan.Primary()),
				"reason":  plan.Reason,
				"targets": plan.Targets,
			}
		}
		trace["governance"] = turnTraceGovernance(meta, target, msgs)
		if target.Metadata != nil {
			if steps, ok := target.Metadata["tool_steps"]; ok {
				trace["tool_steps"] = steps
			}
			if raw, ok := target.Metadata[protocol.MetadataInferenceUsage]; ok {
				trace["inference_usage"] = raw
			}
			if rt, ok := target.Metadata["reasoning_text"].(string); ok && rt != "" {
				trace["reasoning_text"] = rt
			}
			if traceID, ok := target.Metadata[protocol.MetadataTraceID].(string); ok && traceID != "" {
				trace["trace_id"] = traceID
				if spans := preferredTurnTraceSpans(traceID, target.Metadata); spans != nil {
					trace["spans"] = spans
				}
			}
		}
		if runID := target.GetCollaborationID(); runID != "" {
			ids := map[string]interface{}{
				"run_id":  runID,
				"task_id": target.GetTaskID(),
			}
			if target.Metadata != nil {
				ids["attempt_id"] = target.Metadata["orchestration_attempt_id"]
			}
			trace["orchestration_ids"] = ids
			if snapshot, err := chatHub.GetOrchestrationSnapshot(r.Context(), runID, 0); err == nil {
				trace["orchestration"] = snapshot
			}
		}
		if spans, ok := trace["spans"]; ok {
			if selection := turnTraceContextSelection(spans); selection != nil {
				trace["context_selection"] = selection
			}
		}
		compress := protocol.ExtractCompressMeta(target)
		if compress.Strategy != "" || compress.BytesIn > 0 || compress.BytesOut > 0 {
			trace["compress"] = map[string]interface{}{
				"strategy":  compress.Strategy,
				"bytes_in":  compress.BytesIn,
				"bytes_out": compress.BytesOut,
			}
		}
	} else if query != "" {
		plan := routing.PlanKnowledgeRoute(query)
		trace["plan"] = plan
		trace["retrieval"] = map[string]interface{}{
			"mode":    string(plan.Primary()),
			"reason":  plan.Reason,
			"targets": plan.Targets,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}

func turnTraceMetadataCount(metadata map[string]interface{}, key string) (int, bool) {
	value, exists := metadata[key]
	if !exists {
		return 0, false
	}
	switch n := value.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func turnTraceContextSelection(raw interface{}) map[string]interface{} {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var spans []tracelib.Span
	if err := json.Unmarshal(encoded, &spans); err != nil {
		return nil
	}
	for i := len(spans) - 1; i >= 0; i-- {
		if spans[i].Name == "context_select" {
			return spans[i].Attrs
		}
	}
	return nil
}

func preferredTurnTraceSpans(traceID string, metadata map[string]interface{}) interface{} {
	if loaded, err := tracelib.Load(traceID); err == nil && len(loaded.Spans) > 0 {
		return loaded.Spans
	}
	if metadata != nil {
		return metadata[protocol.MetadataTraceSpans]
	}
	return nil
}

func turnTraceHasRoutingMeta(msg *protocol.Message) bool {
	if msg == nil {
		return false
	}
	meta := protocol.ExtractRoutingMeta(msg)
	return meta.Model != "" || meta.Reason != "" || meta.KnowledgeRoute != ""
}

func turnTraceGovernance(meta protocol.RoutingMeta, target *protocol.Message, msgs []*protocol.Message) map[string]interface{} {
	composerMode := meta.ComposerMode
	contextScope := meta.ContextScope
	implSession := meta.ImplSession

	if composerMode == "" || contextScope == "" {
		if input := turnTraceInputMessage(target, msgs); input != nil {
			caps := protocol.ResolveTurnCapabilities(input)
			if composerMode == "" {
				composerMode = caps.ComposerMode
			}
			if contextScope == "" {
				contextScope = caps.ContextTier
			}
			if !implSession {
				implSession = caps.CanRunImplSession
			}
		}
	}

	return map[string]interface{}{
		"composer_mode": composerMode,
		"context_scope": contextScope,
		"impl_session":  implSession,
	}
}

func turnTraceInputMessage(target *protocol.Message, msgs []*protocol.Message) *protocol.Message {
	if target == nil {
		return nil
	}
	if target.ReplyTo != "" {
		for _, m := range msgs {
			if m != nil && m.ID == target.ReplyTo {
				return m
			}
		}
	}
	for i, m := range msgs {
		if m != target {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			prev := msgs[j]
			if prev == nil {
				continue
			}
			if protocol.IsUserLikeSender(prev.From) {
				return prev
			}
		}
		break
	}
	return nil
}
