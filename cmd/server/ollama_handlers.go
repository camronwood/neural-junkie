package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	ollamaManager "github.com/camronwood/neural-junkie/internal/ollama"
)

func handleOllamaInstallStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := ollamaMgr.DetectInstallation()
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	running := ollamaMgr.IsServerRunning(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"installed":              status.Installed,
		"bundled":                status.Bundled,
		"version":                status.Version,
		"effective_version":      status.EffectiveVersion,
		"path":                   status.Path,
		"running":                running,
		"auto_install_supported": ollamaManager.AutoInstallSupported(),
		"install_platforms":      []string{"darwin", "linux", "windows"},
		"recommended_version":    status.RecommendedVersion,
		"min_version":            status.MinVersion,
		"update_available":       status.UpdateAvailable,
		"meets_minimum":          status.MeetsMinimum,
		"update_supported":       status.UpdateSupported,
	})
}

func handleOllamaInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	err := ollamaMgr.InstallOllama(r.Context(), func(msg string) {
		fmt.Fprintf(w, "data: %s\n\n", ollamaManager.SSESafeLine(msg))
		flusher.Flush()
	})
	if err != nil {
		fmt.Fprintf(w, "data: ERROR: %s\n\n", ollamaManager.SSESafeLine(err.Error()))
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: DONE\n\n")
	flusher.Flush()
}

func handleOllamaUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	err := ollamaMgr.UpdateOllama(r.Context(), func(msg string) {
		fmt.Fprintf(w, "data: %s\n\n", ollamaManager.SSESafeLine(msg))
		flusher.Flush()
	})
	if err != nil {
		fmt.Fprintf(w, "data: ERROR: %s\n\n", ollamaManager.SSESafeLine(err.Error()))
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: DONE\n\n")
	flusher.Flush()
}

func handleOllamaStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := ollamaMgr.StartServer(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func handleOllamaStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := ollamaMgr.StopServer(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
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

	err := ollamaMgr.PullModel(r.Context(), req.Model, func(p ollamaManager.PullProgress) {
		data, _ := json.Marshal(p)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	})
	if err != nil {
		line, mErr := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
		if mErr != nil {
			line = []byte(`{"status":"error","error":"pull failed"}`)
		}
		fmt.Fprintf(w, "data: %s\n\n", string(line))
		flusher.Flush()
		return
	}
	fmt.Fprintf(w, "data: {\"status\":\"success\"}\n\n")
	flusher.Flush()
}

func handleOllamaCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	models, err := ollamaManager.Library()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		log.Printf("ollama catalog encode: %v", err)
	}
}

func handleOllamaLibrarySearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	page := queryIntDefault(r.URL.Query().Get("page"), 1)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result, err := ollamaManager.SearchRegistry(ctx, query, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleOllamaLibraryTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result, err := ollamaManager.ListRegistryTags(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func queryIntDefault(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func handleOllamaDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Model) == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if err := ollamaMgr.DeleteModel(ctx, req.Model); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "model": req.Model})
}

func handleOllamaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create Ollama provider to test connection
	ollamaProvider := ai.NewOllamaProviderWithConfig("http://localhost:11434", "llama3.1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ollamaProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"running":  err == nil,
		"endpoint": "http://localhost:11434",
	}

	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleOllamaModels returns available Ollama models
func handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" && appConfig != nil {
		endpoint = appConfig.FirstOllamaEndpoint()
	}
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	mgr := ollamaManager.NewManager(endpoint)
	models, err := mgr.ListModels(ctx)

	response := map[string]interface{}{
		"models":   models,
		"endpoint": endpoint,
	}

	if err != nil {
		response["error"] = err.Error()
		response["models"] = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestOllamaConnection tests Ollama connection
func handleTestOllamaConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.Endpoint == "" {
		request.Endpoint = "http://localhost:11434"
	}
	if request.Model == "" {
		request.Model = "llama3.1"
	}

	// Create Ollama provider and test connection
	ollamaProvider := ai.NewOllamaProviderWithConfig(request.Endpoint, request.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := ollamaProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"success":  err == nil,
		"endpoint": request.Endpoint,
		"model":    request.Model,
	}

	if err != nil {
		response["error"] = err.Error()
		response["message"] = "Failed to connect to Ollama"
	} else {
		response["message"] = "Successfully connected to Ollama"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLMStudioStatus checks if LM Studio is running
