package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/phoeniximport"
)

const capPhoenixImport = "phoenix-import"
const capCustomerPack = "customer-pack"

func phoenixImportEnabled(w http.ResponseWriter) bool {
	if appConfig == nil {
		http.Error(w, "config not loaded", http.StatusInternalServerError)
		return false
	}
	if !appConfig.AnyPackCapability(capPhoenixImport) && !appConfig.AnyPackCapability(capCustomerPack) {
		http.Error(w, "Phoenix import requires an enabled customer pack", http.StatusForbidden)
		return false
	}
	return true
}

func phoenixSettings() phoeniximport.Settings {
	p := appConfig.PhoenixSettings()
	return phoeniximport.Settings{
		Environment:     p.EnvironmentOrDefault(),
		CredentialsPath: p.CredentialsPath,
		AuthConfigPath:  p.AuthConfigPath,
	}
}

func legacyHandlePhoenixRoute(w http.ResponseWriter, r *http.Request) {
	if !phoenixImportEnabled(w) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/phoenix")
	path = strings.Trim(path, "/")
	switch {
	case path == "status":
		handlePhoenixStatus(w, r)
	case path == "analyses":
		handlePhoenixAnalyses(w, r)
	case path == "scan-results":
		handlePhoenixScanResults(w, r)
	case path == "import":
		handlePhoenixImport(w, r)
	case path == "import-scan":
		handlePhoenixImportScan(w, r)
	case path == "login/start":
		handlePhoenixLoginStart(w, r)
	case path == "login/poll":
		handlePhoenixLoginPoll(w, r)
	case path == "logout":
		handlePhoenixLogout(w, r)
	default:
		http.NotFound(w, r)
	}
}

func handlePhoenixStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	st := phoeniximport.CheckStatus(ctx, phoenixSettings())
	writeJSON(w, http.StatusOK, st)
}

func handlePhoenixAnalyses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	items, err := phoeniximport.ListAnalyses(ctx, phoenixSettings(), 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyses": items})
}

func handlePhoenixScanResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	items, err := phoeniximport.ListScanResults(ctx, phoenixSettings(), 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scan_results": items})
}

type phoenixImportRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	AnalysisID    string `json:"analysis_id"`
	ScanResultsID string `json:"scan_results_id,omitempty"`
	OutputDir     string `json:"output_dir,omitempty"`
}

func handlePhoenixImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body phoenixImportRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	workspace, exists := workspaceManager.GetWorkspace(body.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Minute)
	defer cancel()
	result, err := phoeniximport.ImportAnalysis(ctx, phoeniximport.ImportRequest{
		WorkspaceRoot: workspace.Path,
		OutputDir:     body.OutputDir,
		AnalysisID:    body.AnalysisID,
		ScanResultsID: body.ScanResultsID,
		Settings:      phoenixSettings(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type phoenixImportScanRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	ScanResultsID string `json:"scan_results_id"`
	OutputDir     string `json:"output_dir,omitempty"`
}

func handlePhoenixImportScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body phoenixImportScanRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	workspace, exists := workspaceManager.GetWorkspace(body.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Minute)
	defer cancel()
	result, err := phoeniximport.ImportScan(ctx, phoeniximport.ImportScanRequest{
		WorkspaceRoot: workspace.Path,
		OutputDir:     body.OutputDir,
		ScanResultsID: body.ScanResultsID,
		Settings:      phoenixSettings(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handlePhoenixLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	start, err := phoeniximport.StartDeviceLogin(ctx, phoenixSettings())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, start)
}

func handlePhoenixLoginPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	result, err := phoeniximport.PollDeviceLogin(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handlePhoenixLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := phoeniximport.Logout(phoenixSettings()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
