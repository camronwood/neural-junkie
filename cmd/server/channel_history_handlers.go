package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleChannelExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "markdown"
	}
	msgs := chatHub.ExportChannelMessages(channel)
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"channel":  channel,
			"count":    len(msgs),
			"messages": msgs,
		})
	case "markdown", "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+channel+`-history.md"`)
		_, _ = w.Write([]byte(hub.FormatChannelExportMarkdown(channel, msgs)))
	default:
		http.Error(w, "format must be markdown or json", http.StatusBadRequest)
	}
}

func handleChannelDurable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Channel string `json:"channel"`
		Durable bool   `json:"durable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	channel := strings.TrimSpace(body.Channel)
	if channel == "" {
		http.Error(w, "channel required", http.StatusBadRequest)
		return
	}
	appConfig.SetChannelDurable(channel, body.Durable)
	if body.Durable {
		chatHub.MarkChannelDurable(channel)
	} else {
		chatHub.UnmarkChannelDurable(channel)
	}
	if err := appConfig.Save(); err != nil {
		http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"channel": channel,
		"durable": body.Durable,
	})
}

func handleChannelDurableGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel == "" {
		http.Error(w, "channel query parameter required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"channel": channel,
		"durable": appConfig.IsChannelDurable(channel),
	})
}
