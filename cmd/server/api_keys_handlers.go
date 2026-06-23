package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/hub/authstore"
)

var authKeyStore *authstore.Store

func initAuthStore() {
	store, err := authstore.Open("")
	if err != nil {
		log.Printf("Auth key store unavailable: %v", err)
		return
	}
	authKeyStore = store
	log.Println("API key store enabled (~/.neural-junkie/auth.db)")
}

func handleAPIKeysRoute(w http.ResponseWriter, r *http.Request) {
	if authKeyStore == nil {
		http.Error(w, "API keys unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		sess, ok := ensureMutationAccess(w, r, "")
		if !ok || !hub.RoleCanAdmin(sess.Role) {
			http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
			return
		}
		keys, err := authKeyStore.ListAPIKeys()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
	case http.MethodPost:
		sess, ok := ensureMutationAccess(w, r, "")
		if !ok || !hub.RoleCanAdmin(sess.Role) {
			http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
			return
		}
		var req struct {
			Name string `json:"name"`
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		raw, rec, err := authKeyStore.CreateAPIKey(req.Name, authstore.NormalizeRole(req.Role))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"api_key": raw,
			"record":  rec,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAPIKeyRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if authKeyStore == nil {
		http.Error(w, "API keys unavailable", http.StatusServiceUnavailable)
		return
	}
	sess, ok := ensureMutationAccess(w, r, "")
	if !ok || !hub.RoleCanAdmin(sess.Role) {
		http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/api-keys/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if err := authKeyStore.RevokeAPIKey(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sessionFromAPIKey(r *http.Request) *hub.HubSession {
	if authKeyStore == nil {
		return nil
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return nil
	}
	raw := strings.TrimSpace(auth[7:])
	if !strings.HasPrefix(raw, "nj_") {
		return nil
	}
	role, id, ok := authKeyStore.ValidateAPIKey(raw)
	if !ok {
		return nil
	}
	return &hub.HubSession{
		UserID:   "apikey:" + id,
		Username: "apikey",
		Role:     string(role),
	}
}
