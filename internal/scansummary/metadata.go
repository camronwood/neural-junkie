package scansummary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const MetadataFileName = "imageMetadata.json"

var wellIDRe = regexp.MustCompile(`^[A-H](?:[1-9]|1[0-2])$`)

// Spot is one detected spot on a well image.
type Spot struct {
	Analyte string `json:"analyte"`
	Row     string `json:"row"`
	Column  string `json:"column"`
	XPx     int    `json:"x_px"`
	YPx     int    `json:"y_px"`
}

// WellMeta is per-well metadata from imageMetadata.json.
type WellMeta struct {
	ImageName        string  `json:"imageName"`
	Time             string  `json:"time,omitempty"`
	FovSizeXUm       float64 `json:"fovSizeXUm,omitempty"`
	FovSizeYUm       float64 `json:"fovSizeYUm,omitempty"`
	ZStagePositionUm float64 `json:"zStagePositionUm,omitempty"`
	XStagePositionUm float64 `json:"xStagePositionUm,omitempty"`
	YStagePositionUm float64 `json:"yStagePositionUm,omitempty"`
	Spots            []Spot  `json:"spots"`
}

// Document is the parsed imageMetadata.json root.
type Document struct {
	Metadata []WellMeta `json:"metadata"`
}

// LoadMetadata reads and parses imageMetadata.json under dir.
func LoadMetadata(dir string) (*Document, error) {
	path := filepath.Join(dir, MetadataFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", MetadataFileName, err)
	}
	if len(doc.Metadata) == 0 {
		return nil, fmt.Errorf("%s: empty metadata", MetadataFileName)
	}
	return &doc, nil
}

// ResolveSummaryDir returns the summary directory for path (dir or path to imageMetadata.json).
func ResolveSummaryDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	base := filepath.Base(path)
	if base == MetadataFileName {
		return filepath.Dir(path), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a scan summary directory: %s", path)
	}
	meta := filepath.Join(path, MetadataFileName)
	if _, err := os.Stat(meta); err != nil {
		return "", fmt.Errorf("missing %s in %s", MetadataFileName, path)
	}
	return path, nil
}

// SummaryStats holds QC aggregates for summarize_scan_summary.
type SummaryStats struct {
	RunLabel       string
	WellCount      int
	SpotsPerWell   int
	AnalyteCounts  map[string]int
	WellsMissing   []string
	UnexpectedSpot map[string]int
}

// BuildSummaryStats computes QC stats from a loaded document.
func BuildSummaryStats(dir string, doc *Document) SummaryStats {
	runLabel := strings.TrimSuffix(filepath.Base(dir), "-summary")
	analyteCounts := make(map[string]int)
	spotsPerWell := 0
	byWell := make(map[string]WellMeta)
	expected := make(map[string]struct{})
	for _, w := range doc.Metadata {
		byWell[w.ImageName] = w
		if spotsPerWell == 0 && len(w.Spots) > 0 {
			spotsPerWell = len(w.Spots)
		}
		for _, s := range w.Spots {
			analyteCounts[s.Analyte]++
			expected[s.Analyte] = struct{}{}
		}
	}
	var wellsMissing []string
	for _, w := range doc.Metadata {
		if len(w.Spots) == 0 {
			wellsMissing = append(wellsMissing, w.ImageName)
		}
	}
	sort.Strings(wellsMissing)
	return SummaryStats{
		RunLabel:      runLabel,
		WellCount:     len(doc.Metadata),
		SpotsPerWell:  spotsPerWell,
		AnalyteCounts: analyteCounts,
		WellsMissing:  wellsMissing,
	}
}

// FormatSummaryMarkdown renders stats for MCP / chat output.
func FormatSummaryMarkdown(stats SummaryStats) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Scan summary: %s\n\n", stats.RunLabel))
	b.WriteString(fmt.Sprintf("- **Wells in metadata:** %d\n", stats.WellCount))
	if stats.SpotsPerWell > 0 {
		b.WriteString(fmt.Sprintf("- **Spots per well (first non-empty):** %d\n", stats.SpotsPerWell))
	}
	if len(stats.WellsMissing) > 0 {
		b.WriteString(fmt.Sprintf("- **Wells with no spots:** %d (%s)\n", len(stats.WellsMissing), strings.Join(stats.WellsMissing, ", ")))
	}
	b.WriteString("\n### Analyte spot counts (all wells)\n\n")
	keys := make([]string, 0, len(stats.AnalyteCounts))
	for k := range stats.AnalyteCounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- %s: %d\n", k, stats.AnalyteCounts[k]))
	}
	b.WriteString("\nOpen the folder in Neural Junkie file explorer (Life sciences pack) to view well images with spot overlays.\n")
	return b.String()
}

// ValidateWellID returns true for A1–H12 style well names.
func ValidateWellID(well string) bool {
	return wellIDRe.MatchString(strings.TrimSpace(well))
}
