package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/pathutil"
	"github.com/camronwood/neural-junkie/internal/scananalysis"
	"github.com/camronwood/neural-junkie/internal/secondaryanalysis"
)

const capSecondaryAnalysisAPI = "secondary-analysis-api"

var secondaryAnalysisMgr *secondaryanalysis.Manager

func initSecondaryAnalysisManager() {
	if secondaryAnalysisMgr != nil {
		return
	}
	secondaryAnalysisMgr = secondaryanalysis.NewManager("", func() secondaryanalysis.BiologySettings {
		if appConfig == nil {
			return secondaryanalysis.BiologySettings{}
		}
		b := appConfig.BiologyMCPSettings()
		return secondaryanalysis.BiologySettings{
			ToolsPath:        b.SecondaryAnalysisToolsPathOrDefault(),
			PythonExecutable: b.PythonExecutableOrDefault(),
			CumulativeQCDir:  b.CumulativeQCDirOrDefault(),
		}
	})
}

func secondaryAnalysisEnabled(w http.ResponseWriter) bool {
	if appConfig == nil || !appConfig.AnyPackCapability(capSecondaryAnalysisAPI) {
		http.Error(w, "Secondary analysis is not enabled (enable a custom pack with secondary-analysis-api)", http.StatusForbidden)
		return false
	}
	return true
}

func legacyHandleSecondaryAnalysisRoute(w http.ResponseWriter, r *http.Request) {
	if !secondaryAnalysisEnabled(w) {
		return
	}
	initSecondaryAnalysisManager()

	path := strings.TrimPrefix(r.URL.Path, "/api/secondary-analysis")
	path = strings.Trim(path, "/")

	if path == "12plex-qc" {
		handle12PlexQC(w, r)
		return
	}
	if path == "comparator-summary" {
		handleComparatorSummary(w, r)
		return
	}
	if path == "run" {
		handleSecondaryAnalysisRun(w, r)
		return
	}
	if strings.HasPrefix(path, "jobs/") {
		id := strings.TrimPrefix(path, "jobs/")
		handleSecondaryAnalysisJobByID(w, r, id)
		return
	}
	http.NotFound(w, r)
}

type twelvePlexQCRequest struct {
	WorkspaceID string `json:"workspace_id"`
	AnalysisDir string `json:"analysis_dir"`
	WriteReport bool   `json:"write_report"`
}

func handle12PlexQC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body twelvePlexQCRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	analysisDir, err := resolveAnalysisDirInWorkspace(body.WorkspaceID, body.AnalysisDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	loaded, _, err := scananalysis.LoadAnalysis(analysisDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	idx := loaded
	report, err := scananalysis.Run12PlexQC(analysisDir, idx, scananalysis.QCOptions{
		WriteReport: body.WriteReport,
		AnalysisDir: analysisDir,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func handleComparatorSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	workspaceID := r.URL.Query().Get("workspace")
	dir := strings.TrimSpace(r.URL.Query().Get("dir"))
	analysisDir, err := resolveAnalysisDirInWorkspace(workspaceID, dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sum, err := scananalysis.LoadComparatorSummary(analysisDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func handleSecondaryAnalysisRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body secondaryanalysis.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	workspace, exists := workspaceManager.GetWorkspace(body.WorkspaceID)
	if !exists {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}
	bio := appConfig.BiologyMCPSettings()
	if body.Config == nil {
		body.Config = map[string]any{}
	}
	body.Config["cumulative_qc_dir"] = secondaryanalysis.ResolveCumulativeQCDir(workspace.Path, bio.CumulativeQCDirOrDefault())
	body.Config["cumulative_spc_dir"] = secondaryanalysis.ResolveCumulativeSPCDir(workspace.Path, bio.CumulativeQCDirOrDefault())
	body.Config["workspace_root"] = workspace.Path
	resolveConfigPathsInWorkspace(body.Config, workspace.Path)

	job, err := secondaryAnalysisMgr.Start(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func handleSecondaryAnalysisJobByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		job, ok := secondaryAnalysisMgr.Get(id)
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodDelete:
		if err := secondaryAnalysisMgr.Cancel(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		job, _ := secondaryAnalysisMgr.Get(id)
		writeJSON(w, http.StatusOK, job)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func resolveAnalysisDirInWorkspace(workspaceID, relDir string) (string, error) {
	relDir = strings.TrimSpace(relDir)
	if workspaceID == "" {
		return "", errBadRequest("workspace_id is required")
	}
	workspace, exists := workspaceManager.GetWorkspace(workspaceID)
	if !exists {
		return "", errBadRequest("workspace not found")
	}
	full := filepath.Join(workspace.Path, relDir)
	abs, err := pathutil.WithinRoot(workspace.Path, full)
	if err != nil {
		return "", errBadRequest("path outside workspace")
	}
	resolved, err := scananalysis.ResolveAnalysisDir(abs)
	if err != nil {
		return "", errBadRequest(err.Error())
	}
	return resolved, nil
}

type errBadRequest string

func (e errBadRequest) Error() string { return string(e) }

func resolveConfigPathsInWorkspace(cfg map[string]any, workspaceRoot string) {
	if v, ok := cfg["analysis_dir"].(string); ok && v != "" {
		cfg["analysis_dir"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["summary_dir"].(string); ok && v != "" {
		cfg["summary_dir"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["experiment_details_csv"].(string); ok && v != "" {
		cfg["experiment_details_csv"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["out_dir"].(string); ok && v != "" {
		cfg["out_dir"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["out_path"].(string); ok && v != "" {
		cfg["out_path"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["cumulative_qc_dir"].(string); ok && v != "" && !filepath.IsAbs(v) {
		cfg["cumulative_qc_dir"] = absInWorkspace(workspaceRoot, v)
	}
	if v, ok := cfg["cumulative_spc_dir"].(string); ok && v != "" && !filepath.IsAbs(v) {
		cfg["cumulative_spc_dir"] = absInWorkspace(workspaceRoot, v)
	}
	if exclude, ok := cfg["exclude_plates"].([]any); ok {
		for i, p := range exclude {
			if s, ok := p.(string); ok && s != "" {
				exclude[i] = absInWorkspace(workspaceRoot, s)
			}
		}
		cfg["exclude_plates"] = exclude
	}
	if plates, ok := cfg["plates"].([]any); ok {
		for i, p := range plates {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if path, ok := m["path"].(string); ok && path != "" {
				m["path"] = absInWorkspace(workspaceRoot, path)
			}
			plates[i] = m
		}
		cfg["plates"] = plates
	}
	if paths, ok := cfg["plates"].([]string); ok {
		for i, p := range paths {
			paths[i] = absInWorkspace(workspaceRoot, p)
		}
		cfg["plates"] = paths
	}
}

func absInWorkspace(workspaceRoot, rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return workspaceRoot
	}
	full := filepath.Join(workspaceRoot, rel)
	if abs, err := pathutil.WithinRoot(workspaceRoot, full); err == nil {
		return abs
	}
	return full
}
