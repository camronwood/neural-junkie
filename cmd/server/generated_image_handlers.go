package main

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

// handleGeneratedImage serves a generated image by message id when history API redacts inline data.
func handleGeneratedImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	messageID := strings.TrimSpace(r.URL.Query().Get("message_id"))
	if channel == "" || messageID == "" {
		http.Error(w, "channel and message_id required", http.StatusBadRequest)
		return
	}
	if !ensureChannelReadAccess(w, r, channel) {
		return
	}

	msg, err := chatHub.GetChannelMessageByID(channel, messageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	mime, data, ok := hub.GeneratedImageBytesFromMessage(msg)
	if !ok || len(data) == 0 {
		http.Error(w, "generated image not found", http.StatusNotFound)
		return
	}
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/jpg":
		w.Header().Set("Content-Type", "image/jpeg")
	case "image/gif":
		w.Header().Set("Content-Type", "image/gif")
	case "image/webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "image/png")
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(data)
}
