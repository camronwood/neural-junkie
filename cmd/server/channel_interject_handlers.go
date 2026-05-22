package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handleChannelInterjectRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "interject" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	channel := parts[0]
	heldBy := ""
	var body struct {
		HeldBy string `json:"held_by"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
		heldBy = strings.TrimSpace(body.HeldBy)
	}
	if err := chatHub.InterjectChannel(channel, heldBy); err != nil {
		if _, ok := err.(interface{ Error() string }); ok && strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"channel": channel,
		"held":    true,
	})
}
