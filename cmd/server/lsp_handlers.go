package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/lsp"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func handleLSPDiagnostics(w http.ResponseWriter, r *http.Request, lang string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireIDEPack(w) {
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
	root := ws.Path
	var items []lsp.Diagnostic
	var err error
	var backend workspacebackend.Backend
	if workspaceBackendResolver != nil {
		if b, berr := workspaceBackendResolver.ForWorkspace(wsID); berr == nil {
			backend = b
			root = b.Root()
		}
	}
	if backend != nil && backend.Kind() != workspacebackend.KindLocal {
		items, err = lsp.DiagnosticsViaBackend(r.Context(), backend, lang)
	} else {
		switch lang {
		case "go":
			items, err = lsp.GoDiagnostics(r.Context(), root)
		case "rust":
			items, err = lsp.RustDiagnostics(r.Context(), root)
		case "python":
			items, err = lsp.PythonDiagnostics(r.Context(), root)
		default:
			http.Error(w, "unsupported language", http.StatusBadRequest)
			return
		}
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
