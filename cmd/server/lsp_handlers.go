package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/lsp"
)

func handleLSPDiagnostics(w http.ResponseWriter, r *http.Request, lang string) {
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
	var items []lsp.Diagnostic
	var err error
	switch lang {
	case "go":
		items, err = lsp.GoDiagnostics(r.Context(), ws.Path)
	case "rust":
		items, err = lsp.RustDiagnostics(r.Context(), ws.Path)
	case "python":
		items, err = lsp.PythonDiagnostics(r.Context(), ws.Path)
	default:
		http.Error(w, "unsupported language", http.StatusBadRequest)
		return
	}
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

func handleLSPGoDiagnostics(w http.ResponseWriter, r *http.Request) {
	handleLSPDiagnostics(w, r, "go")
}

func handleLSPRustDiagnostics(w http.ResponseWriter, r *http.Request) {
	handleLSPDiagnostics(w, r, "rust")
}

func handleLSPPythonDiagnostics(w http.ResponseWriter, r *http.Request) {
	handleLSPDiagnostics(w, r, "python")
}
