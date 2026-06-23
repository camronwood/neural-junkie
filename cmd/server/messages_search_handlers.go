package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/agent"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func handleMessagesSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "q required", http.StatusBadRequest)
		return
	}
	if channel != "" && !ensureChannelReadAccess(w, r, channel) {
		return
	}

	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	var before int64
	if v := strings.TrimSpace(r.URL.Query().Get("before")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			before = n
		}
	}

	messages, err := chatHub.SearchMessages(hub.MessageSearchOptions{
		Channel: channel,
		Query:   q,
		Limit:   limit,
		Before:  before,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	out := make([]*protocol.Message, 0, len(messages))
	for _, m := range messages {
		cp, cerr := protocol.CloneMessage(m)
		if cerr != nil || cp == nil {
			continue
		}
		protocol.RedactImageBinaryMetadata(cp)
		agent.RedactSidecarSecrets(cp)
		out = append(out, cp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
