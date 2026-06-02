package main

import (
	"encoding/json"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleUserRules(w http.ResponseWriter, r *http.Request) {
	if !hub.RequireHubAccess(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		sess := hub.SessionFromRequest(r, hubSessions)
		markdown := ""
		if chatHub != nil {
			markdown = chatHub.GetUserRulesMarkdown(userRulesUsername(sess))
		}
		writeJSON(w, http.StatusOK, map[string]string{"markdown": markdown})
	case http.MethodPut:
		sess, ok := hub.RequireSessionForMutation(w, r, hubSessions)
		if !ok {
			return
		}
		if chatHub == nil {
			http.Error(w, "hub unavailable", http.StatusServiceUnavailable)
			return
		}
		var body struct {
			Markdown string `json:"markdown"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := chatHub.SetUserRulesMarkdown(userRulesUsername(sess), body.Markdown); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func userRulesUsername(sess *hub.HubSession) string {
	if sess == nil {
		return ""
	}
	return sess.Username
}
