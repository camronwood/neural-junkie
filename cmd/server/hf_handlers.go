package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/hfhub"
)

func isAllowedRuntimeProvider(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "claude", "ollama", "lmstudio", "huggingface", "hf":
		return true
	default:
		return false
	}
}

var hfMgr *hfhub.Manager

func initHFManager() error {
	cacheDir := ""
	if appConfig != nil {
		cacheDir = appConfig.HF.CacheDir
	}
	var err error
	hfMgr, err = hfhub.NewManager(cacheDir)
	return err
}

func handleHfStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hfhub.BuildStatus(appConfig, hfMgr))
}

func handleHfCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	models, err := hfhub.Library()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func handleHfTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	token := ai.ResolveHFToken(req.APIKey)
	if token == "" {
		http.Error(w, "HF token required (api_key or HF_TOKEN)", http.StatusBadRequest)
		return
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = "meta-llama/Meta-Llama-3-8B-Instruct"
	}
	prov := ai.NewHuggingFaceProvider("", token, model)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	_, err := prov.GenerateResponse(ctx, "Say hello in one word.", nil)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Connected to Hugging Face Inference"})
}

func handleHfDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	var req struct {
		RepoID   string `json:"repo_id"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RepoID) == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	token := hfhub.TokenFromConfig(appConfig)
	err := hfMgr.Download(r.Context(), req.RepoID, req.Filename, token, func(p hfhub.DownloadProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	})
	if err != nil {
		line, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(line))
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: {\"status\":\"success\"}\n\n")
	flusher.Flush()
}

func handleHfLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	files, err := hfMgr.ListLocal()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"files": files})
}

func handleHfDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	var req struct {
		RepoID   string `json:"repo_id"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RepoID) == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}
	if err := hfMgr.Delete(req.RepoID, req.Filename); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func handleHfImportOllama(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	var req struct {
		RepoID    string `json:"repo_id"`
		Filename  string `json:"filename"`
		OllamaTag string `json:"ollama_tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RepoID) == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}
	path, err := hfMgr.LocalPath(req.RepoID, req.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag := strings.TrimSpace(req.OllamaTag)
	if tag == "" {
		entry, _ := hfhub.FindCatalogEntry(req.RepoID)
		fn := req.Filename
		if entry != nil && fn == "" && len(entry.Files) > 0 {
			fn = entry.Files[0].Filename
		}
		tag = hfhub.DefaultOllamaTag(req.RepoID, fn)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	if err := hfhub.ImportToOllama(ctx, path, tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "imported", "ollama_tag": tag})
}
