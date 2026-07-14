package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

func handleHfSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = "hosted"
	}
	limit := queryIntDefault(r.URL.Query().Get("limit"), 24)
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	result, err := hfhub.SearchModels(ctx, query, mode, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleHfFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	repoID := strings.TrimSpace(r.URL.Query().Get("repo_id"))
	if repoID == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	token := hfhub.TokenFromConfig(appConfig)
	var files []hfhub.CatalogFile
	var err error
	switch kind {
	case "adapter":
		files, err = hfhub.ListRepoAdapterFiles(ctx, repoID, token)
	default:
		files, err = hfhub.ListRepoGGUF(ctx, repoID, token)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"repo_id": repoID, "kind": kind, "files": files})
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
	if err := hfMgr.EnsureDownloadStarted(token, req.RepoID, req.Filename); err != nil {
		line, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(line))
		flusher.Flush()
		return
	}
	err := hfMgr.WatchDownload(r.Context(), req.RepoID, req.Filename, func(p hfhub.DownloadProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	})
	if err != nil && err != context.Canceled {
		line, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(line))
		flusher.Flush()
		return
	}
	if err == nil {
		fmt.Fprintf(w, "data: {\"status\":\"success\"}\n\n")
		flusher.Flush()
	}
}

func handleHfDownloadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	repoID := strings.TrimSpace(r.URL.Query().Get("repo_id"))
	filename := strings.TrimSpace(r.URL.Query().Get("filename"))
	if repoID == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if ready, err := hfMgr.FileReady(repoID, filename); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else if ready {
		json.NewEncoder(w).Encode(hfhub.DownloadProgress{Status: "success", RepoID: repoID, Filename: filename, Percent: 100})
		return
	}
	if p, ok := hfMgr.DownloadStatus(repoID, filename); ok {
		json.NewEncoder(w).Encode(p)
		return
	}
	json.NewEncoder(w).Encode(hfhub.DownloadProgress{Status: "idle", RepoID: repoID, Filename: filename})
}

func handleHfDownloadsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if hfMgr == nil {
		http.Error(w, "HF manager not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"downloads": hfMgr.ActiveDownloads()})
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
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	files, err := hfMgr.ListLocalFiltered(kind)
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
		RepoID        string `json:"repo_id"`
		Filename      string `json:"filename"`
		OllamaTag     string `json:"ollama_tag"`
		BaseOllamaTag string `json:"base_ollama_tag"`
		Kind          string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RepoID) == "" {
		http.Error(w, "repo_id is required", http.StatusBadRequest)
		return
	}
	entry, _ := hfhub.FindCatalogEntry(req.RepoID)
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" && entry != nil && hfhub.IsAdapterEntry(entry) {
		kind = "adapter"
	}
	fn := strings.TrimSpace(req.Filename)
	if fn == "" && entry != nil {
		if primary := hfhub.PrimaryCatalogFile(entry); primary != nil {
			fn = primary.Filename
		}
	}
	path, err := hfMgr.LocalPath(req.RepoID, fn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag := strings.TrimSpace(req.OllamaTag)
	if tag == "" {
		if entry != nil {
			if t := strings.TrimSpace(entry.DefaultOllamaTag); t != "" {
				tag = t
			}
		}
	}
	if tag == "" {
		if entry != nil && hfhub.IsAdapterEntry(entry) {
			tag = hfhub.DefaultAdapterOllamaTag(entry, fn)
		} else {
			tag = hfhub.DefaultOllamaTag(req.RepoID, fn)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	if kind == "adapter" {
		if !requireLoRACapability(w, capLoRACompose) {
			return
		}
		baseTag := strings.TrimSpace(req.BaseOllamaTag)
		if baseTag == "" && entry != nil {
			baseTag = entry.BaseOllamaTag
		}
		if baseTag == "" {
			baseTag = hfhub.DefaultLoRABaseTag
		}
		if err := hfhub.ImportAdapterToOllama(ctx, baseTag, path, tag); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		mmprojPath := ""
		if entry != nil {
			if mm := hfhub.MmprojCatalogFile(entry); mm != nil {
				if p, err := hfMgr.LocalPath(req.RepoID, mm.Filename); err == nil {
					mmprojPath = p
				} else {
					http.Error(w, fmt.Sprintf("mmproj %s not downloaded — download the model (includes projector) first: %v", mm.Filename, err), http.StatusBadRequest)
					return
				}
			}
		}
		if err := hfhub.ImportToOllama(ctx, path, tag, mmprojPath, req.RepoID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "imported", "ollama_tag": tag, "kind": kind})
}
