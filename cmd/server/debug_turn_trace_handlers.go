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
	if query == "" && messageID != "" && channel != "" {
		msgs, err := chatHub.GetMessages(channel, 200)
		if err == nil {
			for _, m := range msgs {
				if m != nil && m.ID == messageID {
					query = m.Content
					break
				}
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
	if messageID != "" && channel != "" {
		msgs, err := chatHub.GetMessages(channel, 200)
		if err == nil {
			for _, m := range msgs {
				if m != nil && m.ID == messageID {
					meta := protocol.ExtractRoutingMeta(m)
					trace["provider"] = meta
					if meta.KnowledgeRoute == "" {
						trace["routing"] = decision
					} else {
						trace["routing"] = map[string]interface{}{
							"target": meta.KnowledgeRoute,
							"reason": meta.KnowledgeReason,
						}
					}
					break
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}
