package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/plans"
)

func handlePlans(w http.ResponseWriter, r *http.Request) {
	if !hub.RequireHubAccess(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := plans.Active().List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": list})
}

func handlePlansSubRoute(w http.ResponseWriter, r *http.Request) {
	if !hub.RequireHubAccess(w, r) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/plans/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rec, err := plans.Active().Get(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, rec)
	case http.MethodPut:
		if _, ok := hub.RequireSessionForMutation(w, r, hubSessions); !ok {
			return
		}
		var body struct {
			Markdown string `json:"markdown"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		rec, err := plans.Active().Put(id, body.Markdown)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, rec)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
