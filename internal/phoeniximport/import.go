package phoeniximport

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultTimeout = 8 * time.Minute

// Settings configures native Phoenix TIM API access.
type Settings struct {
	Environment      string
	CredentialsPath  string // optional; default bbio credentials-{env}.json store
	AuthConfigPath   string // optional Auth0 app creds file (client_id for refresh)
}

// AnalysisSummary is a row for the import picker UI.
type AnalysisSummary struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ImportRequest describes a Phoenix TIM import into a workspace folder.
type ImportRequest struct {
	WorkspaceRoot string
	OutputDir     string // relative to workspace root
	AnalysisID    string
	ScanResultsID string // optional override
	Settings      Settings
}

// ImportResult is returned after a successful layout build.
type ImportResult struct {
	AnalysisDir     string   `json:"analysis_dir"`
	ScanExportDir   string   `json:"scan_export_dir,omitempty"`
	ScanResultsID   string   `json:"scan_results_id,omitempty"`
	FilesWritten    []string `json:"files_written,omitempty"`
	AttachmentNotes []string `json:"attachment_notes,omitempty"`
}

// Status reports TIM auth availability.
type Status struct {
	Environment     string `json:"environment"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	Authenticated   bool   `json:"authenticated"`
	LoggedIn        bool   `json:"logged_in"` // same as Authenticated (UI compat)
	Identity        string `json:"identity,omitempty"`
	Hint            string `json:"hint,omitempty"`
}

// ImportAnalysis downloads analysis + optional scan attachments via TIM API and lays out NJ paths.
func ImportAnalysis(ctx context.Context, req ImportRequest) (*ImportResult, error) {
	req.AnalysisID = strings.TrimSpace(req.AnalysisID)
	if req.AnalysisID == "" {
		return nil, fmt.Errorf("analysis_id required")
	}
	if strings.TrimSpace(req.WorkspaceRoot) == "" {
		return nil, fmt.Errorf("workspace root required")
	}
	outRel := strings.TrimSpace(req.OutputDir)
	if outRel == "" {
		outRel = "phoenix-" + sanitizeDirName(req.AnalysisID)
	}
	outRel = strings.Trim(outRel, `/\`)
	if outRel == "" || outRel == "." {
		return nil, fmt.Errorf("invalid output_dir")
	}

	root, err := filepath.Abs(req.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	outAbs, err := safeJoin(root, outRel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(outAbs, "reports"), 0o755); err != nil {
		return nil, err
	}

	client, err := newTimClient(ctx, req.Settings)
	if err != nil {
		return nil, err
	}
	docRaw, err := client.getDocument(ctx, "analyses", req.AnalysisID)
	if err != nil {
		return nil, fmt.Errorf("get analysis: %w", err)
	}
	scanID := strings.TrimSpace(req.ScanResultsID)
	if scanID == "" {
		scanID = extractScanResultsID(docRaw)
	}

	result := &ImportResult{
		AnalysisDir:   outRel,
		ScanResultsID: scanID,
	}

	attNames, err := client.listAttachments(ctx, "analyses", req.AnalysisID)
	if err != nil {
		return nil, fmt.Errorf("list analysis attachments: %w", err)
	}

	for _, name := range attNames {
		lower := strings.ToLower(name)
		switch {
		case lower == "results.json":
			dest := filepath.Join(outAbs, "reports", "results.json")
			if err := client.downloadAttachment(ctx, "analyses", req.AnalysisID, name, dest); err != nil {
				return nil, err
			}
			result.FilesWritten = append(result.FilesWritten, filepath.Join(outRel, "reports", "results.json"))
		case lower == "process_report.txt":
			dest := filepath.Join(outAbs, "reports", "process_report.txt")
			if err := client.downloadAttachment(ctx, "analyses", req.AnalysisID, name, dest); err != nil {
				result.AttachmentNotes = append(result.AttachmentNotes, "process_report.txt: "+err.Error())
			} else {
				result.FilesWritten = append(result.FilesWritten, filepath.Join(outRel, "reports", "process_report.txt"))
			}
		case lower == "summary.zip":
			tmpZip := filepath.Join(outAbs, ".summary.zip")
			if err := client.downloadAttachment(ctx, "analyses", req.AnalysisID, name, tmpZip); err != nil {
				result.AttachmentNotes = append(result.AttachmentNotes, "summary.zip: "+err.Error())
			} else if err := unzipSafe(tmpZip, outAbs); err != nil {
				result.AttachmentNotes = append(result.AttachmentNotes, "summary.zip extract: "+err.Error())
			} else {
				result.FilesWritten = append(result.FilesWritten, filepath.Join(outRel, "(summary.zip contents)"))
			}
			_ = os.Remove(tmpZip)
		}
	}

	if _, err := os.Stat(filepath.Join(outAbs, "reports", "results.json")); err != nil {
		if len(attNames) == 0 {
			return nil, fmt.Errorf("analysis has no attachments in TIM (pick a COMPLETE analysis with results.json)")
		}
		return nil, fmt.Errorf("analysis export missing reports/results.json (attachments: %v)", attNames)
	}

	if scanID != "" {
		scanDir := filepath.Join(outAbs, "scan-export")
		if err := os.MkdirAll(scanDir, 0o755); err != nil {
			return nil, err
		}
		scanAtts, err := client.listAttachments(ctx, "scanResults", scanID)
		if err != nil {
			result.AttachmentNotes = append(result.AttachmentNotes, "scan attachments: "+err.Error())
		} else {
			scanName := pickScanZipAttachment(scanAtts)
			if scanName == "" {
				result.AttachmentNotes = append(result.AttachmentNotes, "no scan results zip attachment found")
			} else {
				tmpScan := filepath.Join(outAbs, ".scan.zip")
				if err := client.downloadAttachment(ctx, "scanResults", scanID, scanName, tmpScan); err != nil {
					result.AttachmentNotes = append(result.AttachmentNotes, "scan download: "+err.Error())
				} else if err := unzipSafe(tmpScan, scanDir); err != nil {
					result.AttachmentNotes = append(result.AttachmentNotes, "scan extract: "+err.Error())
				} else {
					result.ScanExportDir = filepath.Join(outRel, "scan-export")
					result.FilesWritten = append(result.FilesWritten, result.ScanExportDir)
				}
				_ = os.Remove(tmpScan)
			}
		}
	}

	return result, nil
}

// ListAnalyses returns recent COMPLETE analyses from TIM (newest first).
func ListAnalyses(ctx context.Context, settings Settings, limit int) ([]AnalysisSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	client, err := newTimClient(ctx, settings)
	if err != nil {
		return nil, err
	}
	raw, err := client.listDocuments(ctx, "analyses", listDocumentsOptions{
		limit: limit,
		sort:  "-createdOn",
		query: map[string]any{"status": "COMPLETE"},
	})
	if err != nil {
		return nil, err
	}
	items := parseDocumentList(raw)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

// ListScanResults returns recent scanResults that have a downloadable results attachment.
func ListScanResults(ctx context.Context, settings Settings, limit int) ([]AnalysisSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	client, err := newTimClient(ctx, settings)
	if err != nil {
		return nil, err
	}
	// Many COMPLETE scanResults rows exist without synced S3 attachments; probe until we fill the page.
	raw, err := client.listDocuments(ctx, "scanResults", listDocumentsOptions{
		limit: 150,
		sort:  "-createdOn",
		query: map[string]any{"status": "COMPLETE"},
	})
	if err != nil {
		return nil, err
	}
	candidates := parseDocumentList(raw)
	var out []AnalysisSummary
	for _, item := range candidates {
		attNames, err := client.listAttachments(ctx, "scanResults", item.ID)
		if err != nil || pickScanZipAttachment(attNames) == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ImportScanRequest downloads a scanResults attachment bundle into a workspace folder.
type ImportScanRequest struct {
	WorkspaceRoot string
	OutputDir     string
	ScanResultsID string
	Settings      Settings
}

// ImportScan downloads and extracts linked scanResults zip attachments.
func ImportScan(ctx context.Context, req ImportScanRequest) (*ImportResult, error) {
	scanID := strings.TrimSpace(req.ScanResultsID)
	if scanID == "" {
		return nil, fmt.Errorf("scan_results_id required")
	}
	if strings.TrimSpace(req.WorkspaceRoot) == "" {
		return nil, fmt.Errorf("workspace root required")
	}
	outRel := strings.TrimSpace(req.OutputDir)
	if outRel == "" {
		outRel = "phoenix-scan-" + sanitizeDirName(scanID)
	}
	outRel = strings.Trim(outRel, `/\`)

	root, err := filepath.Abs(req.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	outAbs, err := safeJoin(root, outRel)
	if err != nil {
		return nil, err
	}
	scanDir := filepath.Join(outAbs, "scan-export")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		return nil, err
	}

	client, err := newTimClient(ctx, req.Settings)
	if err != nil {
		return nil, err
	}
	result := &ImportResult{
		AnalysisDir:   outRel,
		ScanExportDir: filepath.Join(outRel, "scan-export"),
		ScanResultsID: scanID,
	}

	scanAtts, err := client.listAttachments(ctx, "scanResults", scanID)
	if err != nil {
		return nil, fmt.Errorf("list scan attachments: %w", err)
	}
	scanName := pickScanZipAttachment(scanAtts)
	if scanName == "" {
		if len(scanAtts) == 0 {
			return nil, fmt.Errorf("scan result has no attachments in TIM (results zip not synced to cloud yet)")
		}
		return nil, fmt.Errorf("no scan results zip attachment found (attachments: %v)", scanAtts)
	}
	tmpScan := filepath.Join(outAbs, ".scan.zip")
	if err := client.downloadAttachment(ctx, "scanResults", scanID, scanName, tmpScan); err != nil {
		return nil, err
	}
	if err := unzipSafe(tmpScan, scanDir); err != nil {
		return nil, err
	}
	_ = os.Remove(tmpScan)
	result.FilesWritten = append(result.FilesWritten, result.ScanExportDir)
	return result, nil
}

// CheckStatus verifies stored credentials and token validity.
func CheckStatus(ctx context.Context, settings Settings) Status {
	st := Status{
		Environment:     settings.EnvironmentOrDefault(),
		CredentialsPath: resolveCredentialsPath(settings),
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	client, err := newTimClient(ctx, settings)
	if err != nil {
		st.Hint = err.Error()
		if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "read credentials") {
			st.Hint = "Sign in with the Phoenix toolbar chip (device code login)."
		}
		return st
	}
	st.Authenticated = true
	st.LoggedIn = true
	st.CredentialsPath = client.credentialsPath
	st.Identity = strings.TrimSpace(client.whoami())
	return st
}

func (s Settings) EnvironmentOrDefault() string {
	if e := strings.TrimSpace(s.Environment); e != "" {
		return e
	}
	return "staging"
}

func extractScanResultsID(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return findStringField(v, "scanResultsId", "scanResultId", "scan_results_id", "scanResultsID", "scanResults")
}

func findStringField(v any, keys ...string) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	for _, val := range m {
		if s := findStringField(val, keys...); s != "" {
			return s
		}
	}
	return ""
}

func pickScanZipAttachment(names []string) string {
	for _, n := range names {
		if strings.EqualFold(n, "results") || strings.EqualFold(n, "results.zip") {
			return n
		}
	}
	for _, n := range names {
		lower := strings.ToLower(n)
		if strings.Contains(lower, "result") && (strings.HasSuffix(lower, ".zip") || lower == "results") {
			return n
		}
	}
	return ""
}

func parseDocumentList(raw json.RawMessage) []AnalysisSummary {
	var arr []any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return summariesFromArray(arr)
	}
	var wrap map[string]any
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	for _, key := range []string{"data", "items", "results", "documents"} {
		if v, ok := wrap[key]; ok {
			if arr, ok := v.([]any); ok {
				return summariesFromArray(arr)
			}
		}
	}
	return summariesFromArray([]any{wrap})
}

func summariesFromArray(arr []any) []AnalysisSummary {
	var out []AnalysisSummary
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := stringField(m, "_id", "id")
		if id == "" {
			continue
		}
		label := stringField(m, "name", "title", "label", "analysisName", "plateBarcode", "barcode")
		if label == "" {
			label = id
		}
		out = append(out, AnalysisSummary{ID: id, Label: label})
	}
	return out
}

func stringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case map[string]any:
			if s, ok := v["$oid"].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func sanitizeDirName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, s)
	s = strings.Trim(s, "-")
	if len(s) > 48 {
		s = s[:48]
	}
	if s == "" {
		return "import"
	}
	return s
}

func safeJoin(root, rel string) (string, error) {
	clean := filepath.Clean(strings.ReplaceAll(rel, `\`, `/`))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside workspace: %q", rel)
	}
	abs := filepath.Join(root, clean)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside workspace: %q", rel)
	}
	return targetAbs, nil
}

func unzipSafe(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		name := filepath.Join(destDir, f.Name)
		abs, err := filepath.Abs(name)
		if err != nil {
			return err
		}
		if abs != destAbs && !strings.HasPrefix(abs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("zip slip: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode()&0o777|0o600)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeOut := out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOut != nil {
			return closeOut
		}
	}
	return nil
}

func parseAttachmentNames(raw []byte) []string {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	var objs []map[string]any
	if err := json.Unmarshal(raw, &objs); err == nil {
		var names []string
		for _, o := range objs {
			if n := stringField(o, "name", "filename", "key"); n != "" {
				names = append(names, n)
			}
		}
		return names
	}
	var wrap map[string]any
	if err := json.Unmarshal(raw, &wrap); err == nil {
		for _, key := range []string{"attachments", "data", "items"} {
			if v, ok := wrap[key]; ok {
				b, _ := json.Marshal(v)
				return parseAttachmentNames(b)
			}
		}
	}
	return nil
}
