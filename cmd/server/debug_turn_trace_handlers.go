package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
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

	decision := routing.ClassifyKnowledgeRoute(query)
	trace := map[string]interface{}{
		"message_id": messageID,
		"channel":    channel,
		"query":      query,
		"routing":    decision,
	}

	var target *protocol.Message
	for _, m := range msgs {
		if m != nil && m.ID == messageID {
			target = m
			break
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
		trace["provider"] = meta
		if meta.KnowledgeRoute != "" {
			trace["routing"] = map[string]interface{}{
				"target": meta.KnowledgeRoute,
				"reason": meta.KnowledgeReason,
			}
		} else if meta.Model != "" || meta.ToolModel != "" {
			trace["routing"] = map[string]interface{}{
				"chat_model": meta.Model,
				"tool_model": meta.ToolModel,
				"reason":     meta.Reason,
				"source":     meta.Source,
			}
		}
		if target.Metadata != nil {
			if steps, ok := target.Metadata["tool_steps"]; ok {
				trace["tool_steps"] = steps
			}
			if rt, ok := target.Metadata["reasoning_text"].(string); ok && rt != "" {
				trace["reasoning_text"] = rt
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}
