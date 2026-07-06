package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/pathutil"
)

type browserWorkspaceRequest struct {
	WorkspaceID   string         `json:"workspace_id"`
	WorkspaceRoot string         `json:"workspace_root"`
	URL           string         `json:"url"`
	BaselinePath  string         `json:"baseline_path"`
	Viewport      map[string]any `json:"viewport"`
	FullPage      bool           `json:"full_page"`
	Threshold     float64        `json:"threshold"`
}

func resolveBrowserWorkspaceRoot(body map[string]any) (string, int, string) {
	if wsRoot := strings.TrimSpace(stringField(body, "workspace_root")); wsRoot != "" {
		return wsRoot, 0, ""
	}
	wsID := strings.TrimSpace(stringField(body, "workspace_id"))
	if wsID == "" {
		return "", http.StatusBadRequest, "workspace_id or workspace_root required"
	}
	if workspaceManager == nil {
		return "", http.StatusInternalServerError, "workspace manager unavailable"
	}
	workspace, ok := workspaceManager.GetWorkspace(wsID)
	if !ok {
		return "", http.StatusNotFound, "workspace not found"
	}
	return workspace.Path, 0, ""
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func handleBrowserAcceptBaseline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if appConfig == nil || appConfig.RouteOwnerPackID("/api/browser") == "" {
		http.Error(w, "Browser API requires the Web browser pack", http.StatusForbidden)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	wsRoot, code, msg := resolveBrowserWorkspaceRoot(body)
	if code != 0 {
		http.Error(w, msg, code)
		return
	}
	baselinePath := strings.TrimSpace(stringField(body, "baseline_path"))
	if baselinePath == "" {
		http.Error(w, "baseline_path required", http.StatusBadRequest)
		return
	}
	absBaseline := baselinePath
	if !filepath.IsAbs(absBaseline) {
		absBaseline = filepath.Join(wsRoot, baselinePath)
	}
	absBaseline, err := pathutil.WithinRoot(wsRoot, absBaseline)
	if err != nil {
		http.Error(w, "baseline path outside workspace", http.StatusForbidden)
		return
	}
	body["workspace_root"] = wsRoot
	if packSidecarMgr == nil {
		http.Error(w, "Pack sidecar manager unavailable", http.StatusServiceUnavailable)
		return
	}
	rec := newResponseRecorder()
	reqBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "/api/browser/screenshot", strings.NewReader(string(reqBody)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	packID := appConfig.RouteOwnerPackID("/api/browser")
	if err := packSidecarMgr.ProxyHTTP(rec, req, packID, "/api/browser/screenshot"); err != nil {
		http.Error(w, "Pack sidecar: "+err.Error(), http.StatusBadGateway)
		return
	}
	if rec.statusCode >= 400 {
		http.Error(w, rec.body.String(), rec.statusCode)
		return
	}
	var shot map[string]any
	if err := json.Unmarshal(rec.body.Bytes(), &shot); err != nil {
		http.Error(w, "decode screenshot response", http.StatusBadGateway)
		return
	}
	b64, _ := shot["png_b64"].(string)
	if strings.TrimSpace(b64) == "" {
		http.Error(w, "empty screenshot", http.StatusBadGateway)
		return
	}
	png, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		http.Error(w, "decode png", http.StatusBadGateway)
		return
	}
	if err := os.MkdirAll(filepath.Dir(absBaseline), 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(absBaseline, png, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"baseline_path": absBaseline,
		"bytes":         len(png),
	})
}

func handleBrowserVisualDiff(w http.ResponseWriter, r *http.Request) {
	if appConfig == nil || appConfig.RouteOwnerPackID("/api/browser") == "" {
		http.Error(w, "Browser API requires the Web browser pack", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	wsRoot, code, msg := resolveBrowserWorkspaceRoot(body)
	if code != 0 {
		http.Error(w, msg, code)
		return
	}
	body["workspace_root"] = wsRoot
	if packSidecarMgr == nil {
		http.Error(w, "Pack sidecar manager unavailable", http.StatusServiceUnavailable)
		return
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "/api/browser/visual-diff", strings.NewReader(string(raw)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	packID := appConfig.RouteOwnerPackID("/api/browser")
	if err := packSidecarMgr.ProxyHTTP(w, req, packID, "/api/browser/visual-diff"); err != nil {
		http.Error(w, "Pack sidecar: "+err.Error(), http.StatusBadGateway)
		return
	}
}

type responseRecorder struct {
	statusCode int
	body       bytes.Buffer
	header     http.Header
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{statusCode: http.StatusOK, header: make(http.Header)}
}

func (r *responseRecorder) Header() http.Header { return r.header }

func (r *responseRecorder) Write(b []byte) (int, error) { return r.body.Write(b) }

func (r *responseRecorder) WriteHeader(statusCode int) { r.statusCode = statusCode }
