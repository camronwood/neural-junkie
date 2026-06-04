package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cadlib "github.com/camronwood/neural-junkie/internal/cad"
	"github.com/camronwood/neural-junkie/internal/pathutil"
)

type cadRenderRequest struct {
	Workspace  string            `json:"workspace"`
	Path       string            `json:"path"`
	ProjectID  string            `json:"project_id"`
	Params     map[string]string `json:"params"`
	OutputPath string            `json:"output_path"`
}

type cadVersionRequest struct {
	Workspace string            `json:"workspace"`
	ProjectID string            `json:"project_id"`
	Path      string            `json:"path"`
	Label     string            `json:"label"`
	Params    map[string]string `json:"params"`
}

type cadRestoreRequest struct {
	Workspace string `json:"workspace"`
	ProjectID string `json:"project_id"`
	VersionID string `json:"version_id"`
}

func cadPackEnabled() bool {
	return appConfig != nil && appConfig.AnyPackCapability("cad-api")
}

func handleCADRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	var req cadRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	scadPath, stlPath, err := resolveCADPaths(req.Workspace, req.Path, req.ProjectID, req.OutputPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	settings := appConfig.CadMCPSettings()
	timeout := time.Duration(settings.RenderTimeoutOrDefault()) * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), timeout+5*time.Second)
	defer cancel()
	if err := cadlib.RenderSCADToSTL(ctx, scadPath, stlPath, cadlib.RenderOptions{
		OpenSCADPath: settings.OpenSCADPathOrDefault(),
		Timeout:      timeout,
		Params:       req.Params,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stlBytes, err := os.ReadFile(stlPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"mime":            "model/stl",
		"content_base64":  base64.StdEncoding.EncodeToString(stlBytes),
		"scad_path":       scadPath,
		"stl_path":        stlPath,
		"params":          cadlib.ParseParams(mustReadSCAD(scadPath)),
	})
}

func handleCADMesh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	workspaceID := r.URL.Query().Get("workspace")
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	_, stlPath, err := resolveCADPaths(workspaceID, path, projectID, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stlBytes, err := os.ReadFile(stlPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"mime":           "model/stl",
		"content_base64": base64.StdEncoding.EncodeToString(stlBytes),
		"stl_path":       stlPath,
	})
}

func handleCADParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	workspaceID := r.URL.Query().Get("workspace")
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	scadPath, _, err := resolveCADPaths(workspaceID, path, projectID, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	content, err := cadlib.ReadSCADFile(scadPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"path":   scadPath,
		"params": cadlib.ParseParams(content),
	})
}

func handleCADVersions(w http.ResponseWriter, r *http.Request) {
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
		if projectID == "" {
			projectID = "default"
		}
		paths, err := cadProjectPaths(projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		versions, err := cadlib.ListVersions(paths)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"versions": versions})
	case http.MethodPost:
		var req cadVersionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		projectID := strings.TrimSpace(req.ProjectID)
		if projectID == "" {
			projectID = "default"
		}
		paths, err := cadProjectPaths(projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scadPath, _, err := resolveCADPaths(req.Workspace, req.Path, projectID, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content, err := cadlib.ReadSCADFile(scadPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		meta, err := cadlib.SaveVersion(paths, req.Label, req.Params, content, paths.STLPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCADVersionRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	var req cadRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = "default"
	}
	paths, err := cadProjectPaths(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content, err := cadlib.RestoreVersion(paths, req.VersionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scad_path": paths.SCADPath,
		"content":   content,
	})
}

func handleCADTestOpenSCAD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !cadPackEnabled() {
		http.Error(w, "CAD pack is not enabled", http.StatusForbidden)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	settings := appConfig.CadMCPSettings()
	out, err := cadlib.TestOpenSCAD(r.Context(), body.Path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": err.Error(),
			"path":    settings.OpenSCADPathOrDefault(),
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"message": out,
	})
}

func cadProjectPaths(projectID string) (cadlib.ProjectPaths, error) {
	settings := appConfig.CadMCPSettings()
	return cadlib.ProjectDir(settings.ArtifactsDirOrDefault(), projectID)
}

func resolveCADPaths(workspaceID, relPath, projectID, outputPath string) (scadPath, stlPath string, err error) {
	relPath = strings.TrimSpace(relPath)
	projectID = strings.TrimSpace(projectID)
	if relPath != "" && filepath.IsAbs(relPath) {
		scadPath = relPath
	} else if relPath != "" && workspaceID != "" {
		workspace, ok := workspaceManager.GetWorkspace(workspaceID)
		if !ok {
			return "", "", errCAD("workspace not found")
		}
		full := filepath.Join(workspace.Path, relPath)
		abs, err := pathutil.WithinRoot(workspace.Path, full)
		if err != nil {
			return "", "", errCAD("path outside workspace")
		}
		scadPath = abs
	} else if projectID != "" || relPath == "" {
		if projectID == "" {
			projectID = "default"
		}
		paths, err := cadProjectPaths(projectID)
		if err != nil {
			return "", "", err
		}
		scadPath = paths.SCADPath
		if outputPath == "" {
			stlPath = paths.STLPath
		}
	} else {
		return "", "", errCAD("workspace and path or project_id required")
	}
	if outputPath != "" {
		if filepath.IsAbs(outputPath) {
			stlPath = outputPath
		} else if workspaceID != "" {
			workspace, ok := workspaceManager.GetWorkspace(workspaceID)
			if ok {
				full := filepath.Join(workspace.Path, outputPath)
				if abs, e := pathutil.WithinRoot(workspace.Path, full); e == nil {
					stlPath = abs
				}
			}
		}
	}
	if stlPath == "" {
		dir := filepath.Dir(scadPath)
		stlPath = filepath.Join(dir, "preview.stl")
	}
	return scadPath, stlPath, nil
}

func errCAD(msg string) error {
	return &cadPathError{msg: msg}
}

type cadPathError struct{ msg string }

func (e *cadPathError) Error() string { return e.msg }

func mustReadSCAD(path string) string {
	s, _ := cadlib.ReadSCADFile(path)
	return s
}
