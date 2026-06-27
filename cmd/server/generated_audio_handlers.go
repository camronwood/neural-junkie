package main

import (
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleGeneratedAudio(w http.ResponseWriter, r *http.Request) {
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
	mime, data, ok := hub.GeneratedAudioBytesFromMessage(msg)
	if !ok || len(data) == 0 {
		http.Error(w, "generated audio not found", http.StatusNotFound)
		return
	}
	switch strings.ToLower(mime) {
	case "audio/mpeg", "audio/mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	case "audio/flac":
		w.Header().Set("Content-Type", "audio/flac")
	case "audio/ogg":
		w.Header().Set("Content-Type", "audio/ogg")
	default:
		w.Header().Set("Content-Type", "audio/wav")
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(data)
}

func handleMusicRoute(w http.ResponseWriter, r *http.Request) {
	if appConfig != nil && appConfig.RouteOwnerPackID("/api/music") != "" {
		handlePackExtensionRoute(w, r)
		return
	}
	http.Error(w, "Music API requires the Music creation pack", http.StatusForbidden)
}
