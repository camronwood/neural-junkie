package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/config"
)

func handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		redacted := appConfig.Redacted()
		json.NewEncoder(w).Encode(redacted.AI.Providers)

	case http.MethodPost:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if p.ID == "" || p.Type == "" {
			http.Error(w, "id and type are required", http.StatusBadRequest)
			return
		}
		syncCLIProviderModelToRuntime(&p)
		if err := appConfig.AddProvider(p); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		appConfig.Save()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProviderByID(w http.ResponseWriter, r *http.Request) {
	// Path: /api/providers/{id} or /api/providers/{id}/test
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/providers/"), "/")
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if action == "test" && r.Method == http.MethodPost {
		pcfg := appConfig.GetProvider(id)
		if pcfg == nil {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		provider, err := ai.ProviderFromConfig(pcfg)
		if err != nil {
			http.Error(w, "Failed to build provider: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		testResult := map[string]interface{}{"provider_id": id, "success": true}
		_, err = provider.GenerateResponse(ctx, "Say hello in one word.", nil)
		if err != nil {
			testResult["success"] = false
			testResult["error"] = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResult)
		return
	}

	switch r.Method {
	case http.MethodPut:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		var p config.ProviderConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		p.ID = id
		syncCLIProviderModelToRuntime(&p)
		if err := appConfig.UpdateProvider(p); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		appConfig.Save()
		globalProviderCache.Evict(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case http.MethodDelete:
		if _, ok := ensureMutationAccess(w, r, ""); !ok {
			return
		}
		if err := appConfig.RemoveProvider(id); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		appConfig.Save()
		globalProviderCache.Evict(id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLMStudioStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create LM Studio provider to test connection
	lmStudioProvider := ai.NewLMStudioProviderWithConfig("http://localhost:1234/v1", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := lmStudioProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"running":  err == nil,
		"endpoint": "http://localhost:1234/v1",
	}

	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLMStudioModels returns available LM Studio models

func handleLMStudioModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get endpoint from query parameter or use default
	endpoint := r.URL.Query().Get("endpoint")
	if endpoint == "" {
		endpoint = "http://localhost:1234/v1"
	}

	// Create LM Studio provider to get models
	lmStudioProvider := ai.NewLMStudioProviderWithConfig(endpoint, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := lmStudioProvider.GetAvailableModels(ctx)

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

// handleTestLMStudioConnection tests LM Studio connection

func handleTestLMStudioConnection(w http.ResponseWriter, r *http.Request) {
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
		request.Endpoint = "http://localhost:1234/v1"
	}

	// Create LM Studio provider and test connection
	lmStudioProvider := ai.NewLMStudioProviderWithConfig(request.Endpoint, request.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := lmStudioProvider.TestConnection(ctx)

	response := map[string]interface{}{
		"success":  err == nil,
		"endpoint": request.Endpoint,
		"model":    request.Model,
	}

	if err != nil {
		response["error"] = err.Error()
		response["message"] = "Failed to connect to LM Studio"
	} else {
		response["message"] = "Successfully connected to LM Studio"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHubDataRead returns bounded text from ~/.neural-junkie after the user grants access in the desktop app.
