package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

var hubSessions = hub.NewSessionManager()

func handleAuthSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !hub.RequireHubAccess(w, r) {
			return
		}
		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Username) == "" {
			req.Username = "Anonymous"
		}
		sess := hubSessions.CreateSession(req.Username)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sess)
	case http.MethodGet:
		sess := hub.SessionFromRequest(r, hubSessions)
		if sess == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sess)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ensureMutationAccess checks loopback/token, session, and optional channel ACL.
func ensureMutationAccess(w http.ResponseWriter, r *http.Request, channel string) (*hub.HubSession, bool) {
	if !hub.RequireHubAccess(w, r) {
		return nil, false
	}
	sess, ok := hub.RequireSessionForMutation(w, r, hubSessions)
	if !ok {
		return nil, false
	}
	if hub.EnforceChannelACL(r) && strings.TrimSpace(channel) != "" && chatHub != nil {
		if !chatHub.RequireChannelAccess(w, sess.Username, channel) {
			return nil, false
		}
	}
	return sess, true
}

// ensureChannelReadAccess applies ACL for message/history reads when a session is sent.
func ensureChannelReadAccess(w http.ResponseWriter, r *http.Request, channel string) bool {
	if !hub.EnforceChannelACL(r) || chatHub == nil {
		return true
	}
	sess := hub.SessionFromRequest(r, hubSessions)
	if sess == nil {
		http.Error(w, "Unauthorized: X-NJ-Session required", http.StatusUnauthorized)
		return false
	}
	return chatHub.RequireChannelAccess(w, sess.Username, channel)
}
