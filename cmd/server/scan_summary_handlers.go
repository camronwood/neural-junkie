package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/scansummary"
)

func handleScanSummaryWellImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil || !appConfig.AnyPackCapability("scan-summary-api") {
		http.Error(w, "Life sciences pack is not enabled", http.StatusForbidden)
		return
	}

	workspaceID := r.URL.Query().Get("workspace")
	dir := strings.TrimSpace(r.URL.Query().Get("dir"))
	well := strings.TrimSpace(r.URL.Query().Get("well"))
	if workspaceID == "" || well == "" {
		http.Error(w, "workspace and well parameters required", http.StatusBadRequest)
		return
	}
	if !scansummary.ValidateWellID(well) {
		http.Error(w, "invalid well id", http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	wellPath := filepath.Join(dir, well)
	fullPath := filepath.Join(workspace.Path, wellPath)
	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !scansummary.IsTIFF(content) {
		http.Error(w, "well file is not TIFF", http.StatusUnsupportedMediaType)
		return
	}

	pngBytes, err := scansummary.DecodeWellTIFFToPNG(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mime":            "image/png",
		"content_base64": base64.StdEncoding.EncodeToString(pngBytes),
	})
}
