package scananalysis

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// QCCheckResult is one pass/fail check for an analyte.
type QCCheckResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Value  string `json:"value,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// AnalyteQCRow aggregates QC results for one analyte.
type AnalyteQCRow struct {
	Analyte string          `json:"analyte"`
	Pass    bool            `json:"pass"`
	Checks  []QCCheckResult `json:"checks"`
}

// PanelQCReport is the full 12-Plex QC result for one plate.
type PanelQCReport struct {
	PlateLabel  string         `json:"plate_label"`
	ProductName string         `json:"product_name,omitempty"`
	OverallPass bool           `json:"overall_pass"`
	Messages    []string       `json:"messages,omitempty"`
	Analytes    []AnalyteQCRow `json:"analytes"`
}

// QCOptions configures 12-Plex QC execution.
type QCOptions struct {
	SpikePercent   float64
	SpikeDilution  float64
	WriteReport    bool
	AnalysisDir    string
}

// Run12PlexQC evaluates Human Inflammatory 12-Plex SOP rules on an analysis export.
func Run12PlexQC(analysisDir string, doc *IndexedDocument, opts QCOptions) (*PanelQCReport, error) {
	if opts.SpikePercent <= 0 {
		opts.SpikePercent = DefaultSpikePct
	}
	if opts.SpikeDilution <= 0 {
		opts.SpikeDilution = DefaultSpikeDilution
	}

	plateData, err := LoadAllAnalytePlateData(analysisDir)
	if err != nil {
		plateData = nil
	}
	byAnalyte := map[string]*AnalytePlateData{}
	for i := range plateData {
		byAnalyte[plateData[i].Analyte] = &plateData[i]
	}

	report := &PanelQCReport{
		PlateLabel:  strings.TrimSuffix(filepath.Base(analysisDir), "-summary"),
		ProductName: doc.Experiment.ProductName,
		OverallPass: true,
	}

	analytes := doc.Analytes
	if len(analytes) == 0 {
		for a := range LLOQThresholdsPGML {
			analytes = append(analytes, a)
		}
		sort.Strings(analytes)
	}

	for _, analyte := range analytes {
		row := evaluateAnalyteQC(analyte, doc, byAnalyte[analyte], opts)
		if !row.Pass {
			report.OverallPass = false
		}
		report.Analytes = append(report.Analytes, row)
	}

	if opts.WriteReport && opts.AnalysisDir != "" {
		outPath := filepath.Join(opts.AnalysisDir, ReportsDirName, "qc_12plex_report.json")
		b, _ := json.MarshalIndent(report, "", "  ")
		_ = os.WriteFile(outPath, b, 0o644)
	}
	return report, nil
}

func evaluateAnalyteQC(analyte string, doc *IndexedDocument, pd *AnalytePlateData, opts QCOptions) AnalyteQCRow {
	row := AnalyteQCRow{Analyte: analyte, Pass: true}
	var checks []QCCheckResult
	var messages []string

	// LLOQ check
	var lloq, uloq *float64
	if pd != nil {
		lloq, uloq = pd.LLOQ, pd.ULOQ
	}
	if lloq == nil {
		if lim, ok := doc.LimitsOfQuant[analyte]; ok {
			lloq = parseLimitFloat(lim.LLOQ)
			uloq = parseLimitFloat(lim.ULOQ)
		}
	}
	if thresh, ok := LLOQThresholdFor(analyte); ok {
		pass := lloq != nil && !math.IsNaN(*lloq) && *lloq <= thresh
		val := "N/A"
		if lloq != nil {
			val = fmt.Sprintf("%.4g", *lloq)
		}
		checks = append(checks, QCCheckResult{
			Name: "LLOQ", Pass: pass, Value: val,
			Detail: fmt.Sprintf("threshold %.4g pg/mL", thresh),
		})
		if !pass {
			row.Pass = false
			msg := fmt.Sprintf("%s LLOQ of %s pg/mL exceeds threshold %.4g pg/mL", analyte, val, thresh)
			messages = append(messages, msg)
		}
	}

	// ULOQ check
	if uloq != nil {
		pass := !math.IsNaN(*uloq) && *uloq >= ULOQMinimumPGML
		checks = append(checks, QCCheckResult{
			Name: "ULOQ", Pass: pass, Value: fmt.Sprintf("%.4g", *uloq),
			Detail: fmt.Sprintf("minimum %.0f pg/mL", ULOQMinimumPGML),
		})
		if !pass {
			row.Pass = false
			messages = append(messages, fmt.Sprintf("%s ULOQ below %.0f pg/mL", analyte, ULOQMinimumPGML))
		}
	}

	// Concentration grid for CV / column / row / spike
	var grid PlateGrid
	if pd != nil && hasGridData(pd.Concentrations) {
		grid = pd.Concentrations
	} else {
		grid = BuildConcentrationGridFromValidation(&doc.Document, analyte)
	}

	if hasGridData(grid) {
		cv := percentCV(gridValuesMasked(grid))
		cvPass := !math.IsNaN(cv) && cv <= IntraplateCVMaxPct
		checks = append(checks, QCCheckResult{
			Name: "IntraplateCV", Pass: cvPass,
			Value: fmt.Sprintf("%.2f%%", cv),
			Detail: fmt.Sprintf("max %.0f%%", IntraplateCVMaxPct),
		})
		if !cvPass {
			row.Pass = false
			messages = append(messages, fmt.Sprintf("%%CV for %s is %.2f%%", analyte, cv))
		}

		colFails := checkColumnDeviation(grid)
		colPass := len(colFails) == 0
		checks = append(checks, QCCheckResult{
			Name: "ColumnDeviation", Pass: colPass,
			Detail: strings.Join(colFails, "; "),
		})
		if !colPass {
			row.Pass = false
			for _, f := range colFails {
				messages = append(messages, f)
			}
		}

		rowFails := checkRowDeviation(grid)
		rowPass := len(rowFails) == 0
		checks = append(checks, QCCheckResult{
			Name: "RowDeviation", Pass: rowPass,
			Detail: strings.Join(rowFails, "; "),
		})
		if !rowPass {
			row.Pass = false
		}

		std1 := (*float64)(nil)
		if pd != nil {
			std1 = pd.Std1Conc
		}
		if std1 == nil {
			std1 = std1FromDocument(doc, analyte)
		}
		srPass, srDetail := checkSpikeRecovery(grid, std1, opts.SpikePercent, opts.SpikeDilution)
		checks = append(checks, QCCheckResult{
			Name: "SpikeRecovery", Pass: srPass, Detail: srDetail,
		})
		if !srPass {
			row.Pass = false
			messages = append(messages, fmt.Sprintf("Spike recoveries for %s out of acceptable range", analyte))
		}
	}

	row.Checks = checks
	return row
}

func hasGridData(g PlateGrid) bool {
	for r := 0; r < 8; r++ {
		for c := 0; c < 12; c++ {
			if g[r][c] != nil {
				return true
			}
		}
	}
	return false
}

func std1FromDocument(doc *IndexedDocument, analyte string) *float64 {
	rows := doc.StandardReport[analyte]
	if len(rows) == 0 {
		return nil
	}
	c := rows[0].Concentration
	return &c
}

func maskedGridForStats(grid PlateGrid) PlateGrid {
	var out PlateGrid
	for r := 0; r < 8; r++ {
		for c := 0; c < 12; c++ {
			if r < 2 || (r < 4 && c < 4) {
				continue
			}
			out[r][c] = grid[r][c]
		}
	}
	return out
}

func checkColumnDeviation(grid PlateGrid) []string {
	masked := maskedGridForStats(grid)
	vals := gridValuesMasked(masked)
	overall := nanMean(vals)
	if math.IsNaN(overall) || overall == 0 {
		return nil
	}
	var fails []string
	for c := 0; c < 12; c++ {
		var colVals []float64
		for r := 0; r < 8; r++ {
			if masked[r][c] != nil && !math.IsNaN(*masked[r][c]) {
				colVals = append(colVals, *masked[r][c])
			}
		}
		colMean := nanMean(colVals)
		if math.IsNaN(colMean) {
			continue
		}
		pct := 100 * ((colMean / overall) - 1)
		if math.Abs(pct) > ColumnRowDevMaxPct {
			fails = append(fails, fmt.Sprintf("Column %d deviates from plate mean by %.2f%%", c+1, pct))
		}
	}
	return fails
}

func checkRowDeviation(grid PlateGrid) []string {
	masked := maskedGridForStats(grid)
	vals := gridValuesMasked(masked)
	overall := nanMean(vals)
	if math.IsNaN(overall) || overall == 0 {
		return nil
	}
	var fails []string
	for r := 0; r < 8; r++ {
		var rowVals []float64
		for c := 0; c < 12; c++ {
			if masked[r][c] != nil && !math.IsNaN(*masked[r][c]) {
				rowVals = append(rowVals, *masked[r][c])
			}
		}
		rowMean := nanMean(rowVals)
		if math.IsNaN(rowMean) {
			continue
		}
		pct := 100 * ((rowMean / overall) - 1)
		if math.Abs(pct) > ColumnRowDevMaxPct {
			fails = append(fails, fmt.Sprintf("Row %c deviates from plate mean by %.2f%%", 'A'+rune(r), pct))
		}
	}
	return fails
}

func checkSpikeRecovery(grid PlateGrid, std1 *float64, spikePct, dilution float64) (bool, string) {
	if std1 == nil || *std1 == 0 {
		return true, "skipped (no std1)"
	}
	// Wells C-D rows (index 2-3), cols 1-4 (index 0-3): serum unspiked, serum spiked, plasma unspiked, plasma spiked
	cell := func(r, c int) float64 {
		if grid[r][c] == nil || math.IsNaN(*grid[r][c]) {
			return 0
		}
		return *grid[r][c]
	}
	sUS := (cell(2, 0) + cell(3, 0)) / 2
	sS := (cell(2, 1) + cell(3, 1)) / 2
	pUS := (cell(2, 2) + cell(3, 2)) / 2
	pS := (cell(2, 3) + cell(3, 3)) / 2

	denom := *std1 * (spikePct / 100) * (1 / dilution)
	if denom == 0 {
		return true, "skipped"
	}
	srS := 100 * (sS - sUS) / denom
	srP := 100 * (pS - pUS) / denom
	pass := math.Abs(100-srS) < 30 && math.Abs(100-srP) < 30
	detail := fmt.Sprintf("serum SR=%.1f%%, plasma SR=%.1f%%", srS, srP)
	return pass, detail
}

// FormatPanelQCMarkdown renders a PanelQCReport for MCP/chat.
func FormatPanelQCMarkdown(report *PanelQCReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 12-Plex QC: %s\n\n", report.PlateLabel))
	if report.ProductName != "" {
		b.WriteString(fmt.Sprintf("- **Product:** %s\n", report.ProductName))
	}
	status := "PASS"
	if !report.OverallPass {
		status = "FAIL"
	}
	b.WriteString(fmt.Sprintf("- **Overall:** %s\n\n", status))
	for _, a := range report.Analytes {
		tag := "Pass"
		if !a.Pass {
			tag = "Fail"
		}
		b.WriteString(fmt.Sprintf("### %s — %s\n", a.Analyte, tag))
		for _, c := range a.Checks {
			ch := "pass"
			if !c.Pass {
				ch = "fail"
			}
			b.WriteString(fmt.Sprintf("- %s (%s): %s %s\n", c.Name, ch, c.Value, c.Detail))
		}
		b.WriteString("\n")
	}
	return b.String()
}
