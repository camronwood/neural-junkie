package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/camronwood/neural-junkie/internal/hardware"
)

func handleSystemMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpoint := "http://localhost:11434"
	if appConfig != nil {
		if ep := appConfig.FirstOllamaEndpoint(); ep != "" {
			endpoint = ep
		}
	}

	snap, err := hardware.BuildSystemMemorySnapshot(endpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		log.Printf("system memory encode: %v", err)
	}
}
