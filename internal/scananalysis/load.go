package scananalysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	ResultsFileName       = "results.json"
	ReportsDirName        = "reports"
	ProcessReportFileName = "process_report.txt"
)

var nanRe = regexp.MustCompile(`\bNaN\b`)

// LoadSource indicates which file format supplied analysis data.
type LoadSource string

const (
	SourceJSON LoadSource = "json"
	SourceCSV  LoadSource = "csv"
)

// NormalizeJSONNaN replaces non-standard JSON NaN tokens with null.
func NormalizeJSONNaN(raw []byte) []byte {
	return nanRe.ReplaceAll(raw, []byte("null"))
}

// HasResultsJSON reports whether reports/results.json exists under analysisDir.
func HasResultsJSON(analysisDir string) bool {
	_, err := os.Stat(ResultsPath(analysisDir))
	return err == nil
}

// FindSummaryCSVFiles lists reports/*_summary_report.csv under an analysis directory.
func FindSummaryCSVFiles(analysisDir string) ([]string, error) {
	reportsDir := filepath.Join(analysisDir, ReportsDirName)
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if IsSummaryCSVPath(e.Name()) {
			out = append(out, filepath.Join(reportsDir, e.Name()))
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no *_summary_report.csv files in %s", reportsDir)
	}
	return out, nil
}

// IsAnalysisExport reports whether path resolves to a scan analysis export (JSON or CSV).
func IsAnalysisExport(path string) bool {
	_, err := ResolveAnalysisDir(path)
	return err == nil
}

// ResolveAnalysisDir returns the analysis root directory for path (dir, results.json, or summary CSV).
func ResolveAnalysisDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	base := filepath.Base(path)
	if base == ResultsFileName {
		parent := filepath.Base(filepath.Dir(path))
		if parent == ReportsDirName {
			return filepath.Dir(filepath.Dir(path)), nil
		}
		return filepath.Dir(path), nil
	}
	if IsSummaryCSVPath(path) {
		parent := filepath.Base(filepath.Dir(path))
		if parent == ReportsDirName {
			return filepath.Dir(filepath.Dir(path)), nil
		}
		return filepath.Dir(path), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a scan analysis path: %s", path)
	}
	if HasResultsJSON(path) {
		return path, nil
	}
	if csvs, err := FindSummaryCSVFiles(path); err == nil && len(csvs) > 0 {
		return path, nil
	}
	return "", fmt.Errorf("missing %s/%s or reports/*_summary_report.csv in %s", ReportsDirName, ResultsFileName, path)
}

// ResultsPath returns the path to results.json under an analysis directory.
func ResultsPath(analysisDir string) string {
	return filepath.Join(analysisDir, ReportsDirName, ResultsFileName)
}

// ProcessReportPath returns the path to process_report.txt under an analysis directory.
func ProcessReportPath(analysisDir string) string {
	return filepath.Join(analysisDir, ReportsDirName, ProcessReportFileName)
}

// ProcessReportExcerpt returns the first maxLines of process_report.txt when present.
func ProcessReportExcerpt(analysisDir string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	raw, err := os.ReadFile(ProcessReportPath(analysisDir))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// LoadResults reads and parses reports/results.json under analysisDir.
func LoadResults(analysisDir string) (*IndexedDocument, error) {
	dir, err := ResolveAnalysisDir(analysisDir)
	if err != nil {
		return nil, err
	}
	path := ResultsPath(dir)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseResults(raw)
}

// LoadAnalysis loads analysis data from results.json when present, otherwise from summary CSV files.
func LoadAnalysis(path string) (*IndexedDocument, LoadSource, error) {
	dir, err := ResolveAnalysisDir(path)
	if err != nil {
		return nil, "", err
	}
	if HasResultsJSON(dir) {
		doc, err := LoadResults(dir)
		return doc, SourceJSON, err
	}
	csvFiles, err := FindSummaryCSVFiles(dir)
	if err != nil {
		return nil, "", err
	}
	if IsSummaryCSVPath(path) {
		csvPath := path
		if !filepath.IsAbs(csvPath) {
			csvPath = filepath.Join(dir, ReportsDirName, filepath.Base(path))
		}
		csvFiles = []string{csvPath}
	}
	docs := make([]*Document, 0, len(csvFiles))
	for _, csvPath := range csvFiles {
		analyte, ok := AnalyteFromSummaryCSVPath(csvPath)
		if !ok {
			continue
		}
		raw, err := os.ReadFile(csvPath)
		if err != nil {
			return nil, "", err
		}
		doc, err := ParseSummaryCSV(string(raw), analyte)
		if err != nil {
			return nil, "", err
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil, "", fmt.Errorf("failed to parse any summary report CSV files under %s", dir)
	}
	return BuildIndexes(MergeDocuments(docs)), SourceCSV, nil
}

// ParseResults parses normalized results.json bytes.
func ParseResults(raw []byte) (*IndexedDocument, error) {
	normalized := NormalizeJSONNaN(raw)
	var doc Document
	if err := json.Unmarshal(normalized, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ResultsFileName, err)
	}
	if doc.Experiment.ProductName == "" && len(doc.Validation) == 0 {
		return nil, fmt.Errorf("%s: empty or invalid analysis document", ResultsFileName)
	}
	return BuildIndexes(&doc), nil
}
