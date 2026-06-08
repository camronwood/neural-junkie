package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/memory"
)

func handleMemoryRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/memory")
	path = strings.Trim(path, "/")
	switch path {
	case "stats":
		handleMemoryStats(w, r)
		return
	case "query":
		handleMemoryQuery(w, r)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func handleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	store := memory.GlobalStore()
	if store == nil {
		writeJSON(w, http.StatusOK, memory.Stats{BySourceType: map[string]int{}})
		return
	}
	st, err := store.Stats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func handleMemoryQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil || !appConfig.ConversationMemoryEnabled() {
		writeJSON(w, http.StatusOK, []memory.SearchResult{})
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	collabID := strings.TrimSpace(r.URL.Query().Get("collaboration_id"))
	if collabID == "" && channel != "" {
		collabID = memory.ResolveCollabID(channel)
	}
	limit := 5
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		limit = n
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	results, err := memory.QueryPreview(ctx, memory.PromptContext{
		Query:           q,
		Channel:         channel,
		CollaborationID: collabID,
	}, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []memory.SearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}
