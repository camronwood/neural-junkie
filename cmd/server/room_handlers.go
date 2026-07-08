package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func requireRoomChatCapability(w http.ResponseWriter) bool {
	if config.AppConfig() == nil || !config.AppConfig().HasPackCapability("room-chat") {
		http.Error(w, "Not found", http.StatusNotFound)
		return false
	}
	return true
}

// handleRoomCreate starts an ephemeral room on the current hub.
func handleRoomCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireRoomChatCapability(w) {
		return
	}
	sess, ok := ensureMutationAccess(w, r, "")
	if !ok {
		return
	}
	if chatHub == nil {
		http.Error(w, "Hub unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Name       string `json:"name"`
		TTLHours   int    `json:"ttl_hours"`
		MaxMembers int    `json:"max_members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	opts := hub.DefaultRoomOptions()
	if strings.TrimSpace(req.Name) != "" {
		opts.Name = req.Name
	}
	if req.TTLHours > 0 {
		opts.TTL = time.Duration(req.TTLHours) * time.Hour
	}
	if req.MaxMembers > 0 {
		opts.MaxMembers = req.MaxMembers
	}

	room, err := chatHub.CreateRoom(sess.Username, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Ensure host is included in membership.
	_, _ = chatHub.JoinRoom(room.JoinCode, sess.Username)

	chName := hub.RoomGeneralChannel(room.ID)
	ch := chatHub.CreateChannelWithType(chName, "Room chat", "", protocol.ChannelTypeRoom, sess.Username)
	ch.RoomID = room.ID
	chatHub.SyncRoomChannelMembers(room.ID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"room":    room,
		"channel": ch,
	})
}

// handleRoomJoin allows a LAN guest to join a room by join code. This endpoint is intentionally
// accessible without a hub token; the join code is the trust ceremony.
func handleRoomJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireRoomChatCapability(w) {
		return
	}
	if chatHub == nil {
		http.Error(w, "Hub unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		JoinCode string `json:"join_code"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Username) == "" {
		req.Username = "Anonymous"
	}

	room, err := chatHub.JoinRoom(req.JoinCode, req.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chatHub.SyncRoomChannelMembers(room.ID)

	// Issue a normal hub session; channel ACL restricts room access.
	sess := hubSessions.CreateSession(req.Username, "member")

	baseURL := "http://" + strings.TrimSpace(r.Host)
	if strings.HasPrefix(strings.ToLower(r.Host), "http://") || strings.HasPrefix(strings.ToLower(r.Host), "https://") {
		baseURL = strings.TrimSpace(r.Host)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	resp := map[string]interface{}{
		"room":         room,
		"session":      sess,
		"hub_url":      baseURL,
		"hub_token":    hub.HubAccessToken(),
		"room_channel": hub.RoomGeneralChannel(room.ID),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleRoomLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireRoomChatCapability(w) {
		return
	}
	sess, ok := ensureMutationAccess(w, r, "")
	if !ok {
		return
	}
	if chatHub == nil {
		http.Error(w, "Hub unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.RoomID) == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}
	chatHub.LeaveRoom(req.RoomID, sess.Username)
	chatHub.SyncRoomChannelMembers(req.RoomID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleRoomEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireRoomChatCapability(w) {
		return
	}
	sess, ok := ensureMutationAccess(w, r, "")
	if !ok {
		return
	}
	if chatHub == nil {
		http.Error(w, "Hub unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	roomID := strings.TrimSpace(req.RoomID)
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	room, _ := chatHub.GetRoom(roomID)
	if err := chatHub.EndRoom(roomID, sess.Username); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if room != nil {
		for _, chName := range room.Channels {
			_ = chatHub.DeleteChannel(chName)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleRoomActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireRoomChatCapability(w) {
		return
	}
	sess := hub.SessionFromRequest(r, hubSessions)
	if sess == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !hub.RequireHubAccess(w, r) {
		return
	}
	if chatHub == nil {
		http.Error(w, "Hub unavailable", http.StatusServiceUnavailable)
		return
	}
	rooms := chatHub.ListActiveRooms(sess.Username)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"rooms": rooms,
	})
}

// handleRoomSubRoute serves:
// - GET /api/room/{room_id}
// - GET /api/room/{room_id}/presence
func handleRoomSubRoute(w http.ResponseWriter, r *http.Request) {
	if !requireRoomChatCapability(w) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/room/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	roomID := strings.TrimSpace(parts[0])
	if roomID == "" {
		http.NotFound(w, r)
		return
	}

	// Presence
	if len(parts) == 2 && parts[1] == "presence" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !hub.RequireHubAccess(w, r) {
			return
		}
		sess := hub.SessionFromRequest(r, hubSessions)
		if sess == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		room, ok := chatHub.GetRoom(roomID)
		if !ok || room == nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		// ACL: user must be able to access the general room channel.
		if !chatHub.RequireChannelAccess(w, sess.Username, hub.RoomGeneralChannel(roomID)) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"room_id": roomID,
			"members": room.Members,
		})
		return
	}

	// Metadata
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !hub.RequireHubAccess(w, r) {
		return
	}
	sess := hub.SessionFromRequest(r, hubSessions)
	if sess == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	room, ok := chatHub.GetRoom(roomID)
	if !ok || room == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	if !chatHub.RequireChannelAccess(w, sess.Username, hub.RoomGeneralChannel(roomID)) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(room)
}

