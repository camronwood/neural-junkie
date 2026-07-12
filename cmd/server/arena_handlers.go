package main

import (
	"encoding/json"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleArenaMatchRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if chatHub == nil || !chatHub.ArenaAvailable() {
		http.Error(w, "Model Arena requires the Model Arena pack", http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case "/api/arena/match/step":
		var req hub.ArenaMatchStepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		out, err := chatHub.ArenaRunMatchStep(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeArenaJSON(w, out)
	case "/api/arena/match/run":
		var req hub.ArenaMatchRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		out, err := chatHub.ArenaRunMatchAuto(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeArenaJSON(w, out)
	default:
		http.NotFound(w, r)
	}
}

func writeArenaJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
