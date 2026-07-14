package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/config"
)

func handleDevComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
		return
	}
	var req struct {
		Prefix   string `json:"prefix"`
		Suffix   string `json:"suffix"`
		Language string `json:"language"`
		Path     string `json:"path"`
		Context  string `json:"context"`
		Model    string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prefix) == "" {
		http.Error(w, "prefix required", http.StatusBadRequest)
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" && appConfig != nil {
		model = config.DevOllamaCodeModel
	}
	if model == "" {
		model = config.DevOllamaCodeModel
	}
	endpoint := "http://localhost:11434"
	if appConfig != nil {
		for _, p := range appConfig.AI.Providers {
			if p.Type == "ollama" && strings.TrimSpace(p.Endpoint) != "" {
				endpoint = strings.TrimRight(p.Endpoint, "/")
				break
			}
		}
	}
	prompt := req.Prefix
	if ctx := strings.TrimSpace(req.Context); ctx != "" {
		lang := strings.TrimSpace(req.Language)
		if lang == "" {
			lang = "text"
		}
		path := strings.TrimSpace(req.Path)
		if path == "" {
			path = "file"
		}
		if len(ctx) > 8000 {
			ctx = ctx[:8000] + "\n…"
		}
		prompt = "File: " + path + "\n```" + lang + "\n" + ctx + "\n```\n\nComplete at cursor:\n" + req.Prefix
	} else if req.Suffix != "" {
		prompt += "<|fim_suffix|>" + req.Suffix + "<|fim_prefix|>" + req.Prefix + "<|fim_middle|>"
	}
	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 64,
			"temperature": 0.2,
		},
	}
	raw, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/generate", bytes.NewReader(raw))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		http.Error(w, string(data), resp.StatusCode)
		return
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		http.Error(w, "invalid ollama response", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"completion": strings.TrimSpace(out.Response)})
}
