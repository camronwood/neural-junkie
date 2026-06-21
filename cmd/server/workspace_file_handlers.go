package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/scansummary"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.URL.Query().Get("workspace")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	rel := strings.TrimPrefix(path, "/")
	if workspace.Kind == workspacebackend.KindSSH || workspace.Kind == workspacebackend.KindDevcontainer {
		entries, code, msg := backendListDir(r.Context(), workspaceID, rel)
		if code != 0 {
			http.Error(w, msg, code)
			return
		}
		var files []map[string]interface{}
		for _, entry := range entries {
			files = append(files, map[string]interface{}{
				"name":     entry.Name,
				"path":     entry.Path,
				"is_dir":   entry.IsDir,
				"size":     entry.Size,
				"mod_time": entry.ModTime,
			})
		}
		_ = json.NewEncoder(w).Encode(files)
		return
	}

	if info, err := os.Stat(workspace.Path); err != nil || !info.IsDir() {
		http.Error(w, "Workspace path is unavailable", http.StatusNotFound)
		return
	}

	fullPath := filepath.Join(workspace.Path, path)

	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Calculate the relative path from workspace root
		entryPath := filepath.Join(path, entry.Name())
		// Clean up the path to use forward slashes
		entryPath = strings.TrimPrefix(entryPath, "/")
		if entryPath == "" {
			entryPath = entry.Name()
		}

		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"path":     entryPath,
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mod_time": info.ModTime(),
		})
	}

	json.NewEncoder(w).Encode(files)
}

func isWorkspaceImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".ico", ".svg":
		return true
	default:
		return false
	}
}

func handleFileContent(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		workspaceID := r.URL.Query().Get("workspace")
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "path parameter required", http.StatusBadRequest)
			return
		}

		workspace, exists := workspaceManager.GetWorkspace(workspaceID)
		if !exists {
			http.Error(w, "Workspace not found", http.StatusNotFound)
			return
		}

		relPath := strings.TrimPrefix(path, "/")
		if isRemoteWorkspace(workspace) {
			content, code, msg := backendReadFile(r.Context(), workspaceID, relPath)
			if code != 0 {
				http.Error(w, msg, code)
				return
			}
			if scansummary.ValidateWellID(filepath.Base(path)) && scansummary.IsTIFF(content) {
				http.Error(w, "well TIFF: open via scan summary viewer (Life sciences pack)", http.StatusUnsupportedMediaType)
				return
			}
			if r.URL.Query().Get("binary") == "1" || isWorkspaceImageFile(path) {
				mimeType := mime.TypeByExtension(filepath.Ext(path))
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"mime":           mimeType,
					"content_base64": base64.StdEncoding.EncodeToString(content),
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"content": string(content)})
			return
		}

		fullPath := filepath.Join(workspace.Path, path)

		absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
		if err != nil {
			http.Error(w, "Path outside workspace", http.StatusForbidden)
			return
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if scansummary.ValidateWellID(filepath.Base(path)) && scansummary.IsTIFF(content) {
			http.Error(w, "well TIFF: open via scan summary viewer (Life sciences pack)", http.StatusUnsupportedMediaType)
			return
		}

		if r.URL.Query().Get("binary") == "1" || isWorkspaceImageFile(path) {
			mimeType := mime.TypeByExtension(filepath.Ext(path))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"mime":           mimeType,
				"content_base64": base64.StdEncoding.EncodeToString(content),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": string(content),
		})
	case "POST":
		if !hub.RequireHubAccess(w, r) {
			return
		}
		var req struct {
			WorkspaceID string `json:"workspace_id"`
			Path        string `json:"path"`
			Content     string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		workspace, exists := workspaceManager.GetWorkspace(req.WorkspaceID)
		if !exists {
			http.Error(w, "Workspace not found", http.StatusNotFound)
			return
		}

		if isRemoteWorkspace(workspace) {
			code, msg := backendWriteFile(r.Context(), req.WorkspaceID, req.Path, []byte(req.Content))
			if code != 0 {
				http.Error(w, msg, code)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		fullPath := filepath.Join(workspace.Path, req.Path)

		absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
		if err != nil {
			http.Error(w, "Path outside workspace", http.StatusForbidden)
			return
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWorkspacePreview serves workspace files for the HTML browser workbench (text/html and assets).
func handleWorkspacePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace"))
	relPath := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("path")), "/")
	if workspaceID == "" || relPath == "" {
		http.Error(w, "workspace and path required", http.StatusBadRequest)
		return
	}
	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	if isRemoteWorkspace(workspace) {
		content, code, msg := backendReadFile(r.Context(), workspaceID, relPath)
		if code != 0 {
			http.Error(w, msg, code)
			return
		}
		writeWorkspacePreview(w, r, workspaceID, relPath, content)
		return
	}
	fullPath := filepath.Join(workspace.Path, relPath)
	absPath, err := pathutil.WithinRoot(workspace.Path, fullPath)
	if err != nil {
		http.Error(w, "Path outside workspace", http.StatusForbidden)
		return
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeWorkspacePreview(w, r, workspaceID, relPath, content)
}

func writeWorkspacePreview(w http.ResponseWriter, r *http.Request, workspaceID, relPath string, content []byte) {
	ext := strings.ToLower(filepath.Ext(relPath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if ext == ".html" || ext == ".htm" {
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir == "." {
			dir = ""
		}
		basePath := dir
		if basePath != "" && !strings.HasSuffix(basePath, "/") {
			basePath += "/"
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseHref := fmt.Sprintf("%s://%s/api/workspace-preview?workspace=%s&path=%s",
			scheme, r.Host, url.QueryEscape(workspaceID), url.PathEscape(basePath))
		html := string(content)
		baseTag := fmt.Sprintf(`<base href="%s">`, baseHref)
		lower := strings.ToLower(html)
		if idx := strings.Index(lower, "<head"); idx >= 0 {
			if end := strings.Index(lower[idx:], ">"); end >= 0 {
				insertAt := idx + end + 1
				html = html[:insertAt] + baseTag + html[insertAt:]
			} else {
				html = baseTag + html
			}
		} else {
			html = baseTag + html
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
		return
	}
	w.Header().Set("Content-Type", mimeType)
	_, _ = w.Write(content)
}
