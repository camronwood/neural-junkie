package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
)

func handleArenaBenchmarkRoute(w http.ResponseWriter, r *http.Request) {
	if chatHub == nil || !chatHub.ArenaAvailable() {
		http.Error(w, "Model Arena requires the Model Arena pack", http.StatusForbidden)
		return
	}
	path := r.URL.Path
	switch {
	case path == "/api/arena/benchmark/suites" && r.Method == http.MethodGet:
		writeArenaJSON(w, hub.LoadArenaBenchmarkSuites())
	case path == "/api/arena/benchmark/history" && r.Method == http.MethodGet:
		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				limit = n
			}
		}
		jobs, err := chatHub.ListArenaBenchmarkHistory(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeArenaJSON(w, map[string]any{"runs": jobs})
	case path == "/api/arena/benchmark/start" && r.Method == http.MethodPost:
		var req hub.ArenaBenchmarkStartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		job, err := chatHub.StartArenaBenchmark(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeArenaJSON(w, job)
	case strings.HasPrefix(path, "/api/arena/benchmark/") && strings.HasSuffix(path, "/cancel") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/api/arena/benchmark/"), "/cancel")
		id = strings.Trim(id, "/")
		job, err := chatHub.CancelArenaBenchmark(id)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeArenaJSON(w, job)
	case strings.HasPrefix(path, "/api/arena/benchmark/") && r.Method == http.MethodGet:
		id := strings.TrimPrefix(path, "/api/arena/benchmark/")
		id = strings.Trim(id, "/")
		if id == "" || id == "suites" || id == "history" || id == "start" {
			http.NotFound(w, r)
			return
		}
		job, err := chatHub.GetArenaBenchmark(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeArenaJSON(w, job)
	default:
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.NotFound(w, r)
	}
}
