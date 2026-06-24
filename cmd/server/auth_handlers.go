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
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Username) == "" {
			req.Username = "Anonymous"
		}
		role := resolveSessionRole(r, req.Role)
		sess := hubSessions.CreateSession(req.Username, role)
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
	if apiSess := sessionFromAPIKey(r); apiSess != nil {
		if !hub.RoleCanMutate(apiSess.Role) {
			http.Error(w, "Forbidden: viewer role cannot mutate", http.StatusForbidden)
			return nil, false
		}
		if hub.EnforceChannelACL(r) && strings.TrimSpace(channel) != "" && chatHub != nil {
			if !chatHub.RequireChannelAccess(w, apiSess.Username, channel) {
				return nil, false
			}
		}
		return apiSess, true
	}
	sess, ok := hub.RequireSessionForMutation(w, r, hubSessions)
	if !ok {
		return nil, false
	}
	if !hub.RoleCanMutate(sess.Role) {
		http.Error(w, "Forbidden: viewer role cannot mutate", http.StatusForbidden)
		return nil, false
	}
	if hub.EnforceChannelACL(r) && strings.TrimSpace(channel) != "" && chatHub != nil {
		if !chatHub.RequireChannelAccess(w, sess.Username, channel) {
			return nil, false
		}
	}
	return sess, true
}

// resolveSessionRole caps client-supplied roles; admin requires bootstrap or an existing admin caller.
func resolveSessionRole(r *http.Request, requested string) string {
	want := hub.NormalizeRole(requested)
	if want != "admin" {
		if want == "" {
			return "member"
		}
		return want
	}
	if hub.ValidBootstrapToken(r) || callerIsAdmin(r) {
		return "admin"
	}
	return "member"
}

func callerIsAdmin(r *http.Request) bool {
	if apiSess := sessionFromAPIKey(r); apiSess != nil && hub.RoleCanAdmin(apiSess.Role) {
		return true
	}
	if sess := hub.SessionFromRequest(r, hubSessions); sess != nil && hub.RoleCanAdmin(sess.Role) {
		return true
	}
	return false
}

// ensureChannelReadAccess applies hub access and channel ACL for message/history reads.
func ensureChannelReadAccess(w http.ResponseWriter, r *http.Request, channel string) bool {
	if !hub.RequireHubAccess(w, r) {
		return false
	}
	apiSess := sessionFromAPIKey(r)
	sess := hub.SessionFromRequest(r, hubSessions)
	hasIdentity := apiSess != nil || sess != nil
	if hub.AuthRequired() || hasIdentity {
		if !hasIdentity {
			http.Error(w, "Unauthorized: X-NJ-Session required", http.StatusUnauthorized)
			return false
		}
		username := ""
		if sess != nil {
			username = sess.Username
		} else if apiSess != nil {
			username = apiSess.Username
		}
		if chatHub != nil && strings.TrimSpace(channel) != "" {
			return chatHub.RequireChannelAccess(w, username, channel)
		}
		return true
	}
	if hub.RelaxedLocal() {
		return true
	}
	http.Error(w, "Unauthorized: X-NJ-Session required", http.StatusUnauthorized)
	return false
}
