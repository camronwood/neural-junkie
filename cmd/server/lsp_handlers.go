package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/lsp"
)

func handleLSPGoDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireSoftwareDevPack(w) {
		return
	}
	wsID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	if wsID == "" {
		http.Error(w, "workspace required", http.StatusBadRequest)
		return
	}
	ws, ok := workspaceManager.GetWorkspace(wsID)
	if !ok {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	items, err := lsp.GoDiagnostics(r.Context(), ws.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []lsp.Diagnostic{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"diagnostics": items})
}
