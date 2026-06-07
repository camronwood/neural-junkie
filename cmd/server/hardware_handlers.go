package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hardware"
)

func handleSystemHardware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snap, err := hardware.BuildSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snap); err != nil {
		log.Printf("system hardware encode: %v", err)
	}
}

func handleOllamaLibraryLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	row, err := hardware.LookupModel(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if row == nil {
		w.Write([]byte("null"))
		return
	}
	if err := json.NewEncoder(w).Encode(row); err != nil {
		log.Printf("ollama library lookup encode: %v", err)
	}
}
