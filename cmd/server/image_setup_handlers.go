package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func handleImageGenStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := ai.ImageGenConfigFromEnv()
	ollamaRunning := false
	modelPulled := false
	if cfg.Provider == "ollama" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if ollamaMgr != nil {
			ollamaRunning = ollamaMgr.IsServerRunning(ctx)
			if ollamaRunning {
				modelPulled, _ = ollamaMgr.HasModel(ctx, cfg.Model)
			}
		}
	}
	st := ai.BuildImageGenStatus(cfg, ollamaRunning, modelPulled)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}
