package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/camronwood/neural-junkie/internal/inference"
)

func initInferenceStats() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("⚠️  inference stats: no home dir: %v", err)
		return
	}
	path := filepath.Join(home, ".neural-junkie", "inference-stats.json")
	store, err := inference.NewStatsStore(path)
	if err != nil {
		log.Printf("⚠️  inference stats init failed: %v", err)
		return
	}
	inference.SetDefaultStore(store)
	log.Printf("📊 Inference usage stats: %s", path)
}

func handleInferenceUsage(w http.ResponseWriter, r *http.Request) {
	store := inference.DefaultStore()
	if store == nil {
		http.Error(w, "inference usage store not initialized", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(store.Summary()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodDelete:
		if err := store.Reset(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
