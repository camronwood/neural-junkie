package scananalysis

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AnalysisStats holds QC aggregates for summarize_scan_analysis.
type AnalysisStats struct {
	RunLabel            string
	ProductName         string
	PlateBarcode        string
	AnalyteCount        int
	DilutionFactor      float64
	UnknownWithinLOQ    int
	UnknownOutsideLOQ   int
	StandardsWithinLOQ  int
	StandardsOutsideLOQ int
	Analytes            []string
	DataSource          string
	LinkedScanDir       string
	ProcessReportExcerpt string
}

// BuildAnalysisStats computes QC stats from an indexed document.
func BuildAnalysisStats(analysisDir string, doc *IndexedDocument) AnalysisStats {
	runLabel := strings.TrimSuffix(filepath.Base(analysisDir), "-summary")
	stats := AnalysisStats{
		RunLabel:       runLabel,
		ProductName:    doc.Experiment.ProductName,
		PlateBarcode:   doc.Experiment.PlateBarcode,
		AnalyteCount:   len(doc.Analytes),
		DilutionFactor: doc.Experiment.DilutionFactor,
		Analytes:       doc.Analytes,
		LinkedScanDir:  ResolveLinkedScanDir(analysisDir),
	}

	for _, rows := range doc.UnknownReport {
		for _, row := range rows {
			if row.WithinLimitsOfQuantification {
				stats.UnknownWithinLOQ++
			} else {
				stats.UnknownOutsideLOQ++
			}
		}
	}
	for _, rows := range doc.StandardReport {
		for _, row := range rows {
			if row.WithinLimitsOfQuantificationV2 {
				stats.StandardsWithinLOQ++
			} else {
				stats.StandardsOutsideLOQ++
			}
		}
	}
	return stats
}

// FormatAnalysisMarkdown renders stats for MCP / chat output.
func FormatAnalysisMarkdown(stats AnalysisStats) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Scan analysis: %s\n\n", stats.RunLabel))
	if stats.ProductName != "" {
		b.WriteString(fmt.Sprintf("- **Product:** %s\n", stats.ProductName))
	}
	if stats.PlateBarcode != "" {
		b.WriteString(fmt.Sprintf("- **Plate barcode:** %s\n", stats.PlateBarcode))
	}
	b.WriteString(fmt.Sprintf("- **Analytes:** %d\n", stats.AnalyteCount))
	if stats.DataSource != "" {
		b.WriteString(fmt.Sprintf("- **Data source:** %s\n", stats.DataSource))
	}
	if stats.LinkedScanDir != "" {
		b.WriteString(fmt.Sprintf("- **Linked scan folder:** %s (use summarize_scan_summary on this path for TIFF QC)\n", stats.LinkedScanDir))
	}
	if stats.DilutionFactor > 0 && stats.DilutionFactor != 1 {
		b.WriteString(fmt.Sprintf("- **Dilution factor:** %.1f (multiply unknown concentrations by this factor for final pg/ml)\n", stats.DilutionFactor))
	}
	b.WriteString(fmt.Sprintf("- **Unknown wells within LOQ:** %d\n", stats.UnknownWithinLOQ))
	b.WriteString(fmt.Sprintf("- **Unknown wells outside LOQ:** %d\n", stats.UnknownOutsideLOQ))
	b.WriteString(fmt.Sprintf("- **Standard points within LOQ:** %d\n", stats.StandardsWithinLOQ))
	b.WriteString(fmt.Sprintf("- **Standard points outside LOQ:** %d\n", stats.StandardsOutsideLOQ))
	if len(stats.Analytes) > 0 {
		b.WriteString(fmt.Sprintf("\n### Analytes\n\n%s\n", strings.Join(stats.Analytes, ", ")))
	}
	if stats.ProcessReportExcerpt != "" {
		b.WriteString("\n### Process report (excerpt)\n\n```\n")
		b.WriteString(stats.ProcessReportExcerpt)
		b.WriteString("\n```\n")
	}
	b.WriteString("\nOpen the folder in Neural Junkie file explorer (Life sciences pack) to view the analysis plate heat map and link to scan TIFFs.\n")
	return b.String()
}
