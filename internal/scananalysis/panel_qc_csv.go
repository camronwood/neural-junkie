package scananalysis

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

const plateRows = "ABCDEFGH"

// PlateGrid is an 8×12 numeric plate map (rows A–H, cols 1–12).
type PlateGrid [8][12]*float64

// AnalytePlateData holds parsed per-analyte plate sections from summary CSV.
type AnalytePlateData struct {
	Analyte        string
	Concentrations PlateGrid
	Intensities    PlateGrid
	LLOQ           *float64
	ULOQ           *float64
	Std1Conc       *float64
}

func findSectionLine(lines []string, sectionName string) int {
	target := strings.ToLower(strings.TrimSpace(sectionName))
	for idx, line := range lines {
		parts := parseCSVLine(line)
		if len(parts) == 0 {
			continue
		}
		if strings.ToLower(strings.TrimSpace(parts[0])) == target {
			return idx + 2
		}
	}
	return -1
}

func parsePlateMapGrid(lines []string, startIdx int) (PlateGrid, int) {
	var grid PlateGrid
	i := startIdx
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		parts := parseCSVLine(line)
		if len(parts) == 0 {
			i++
			continue
		}
		row := strings.ToUpper(strings.TrimSpace(parts[0]))
		if len(row) != 1 || row < "A" || row > "H" {
			break
		}
		rowIdx := int(row[0] - 'A')
		for col := 1; col <= 12 && col < len(parts); col++ {
			grid[rowIdx][col-1] = parseCSVNum(parts[col])
		}
		i++
	}
	return grid, i
}

// LoadAnalytePlateDataFromSummaryCSV parses plate maps and LOQ from reports/{analyte}_summary_report.csv.
func LoadAnalytePlateDataFromSummaryCSV(path string) (*AnalytePlateData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	analyte, ok := AnalyteFromSummaryCSVPath(path)
	if !ok {
		return nil, fmt.Errorf("not a summary report CSV: %s", path)
	}
	lines := strings.Split(string(raw), "\n")
	data := &AnalytePlateData{Analyte: analyte}

	if idx := findSectionLine(lines, "Plate Map Intensities"); idx >= 0 {
		data.Intensities, _ = parsePlateMapGrid(lines, idx)
	}
	if idx := findSectionLine(lines, "Plate Map Concentrations"); idx >= 0 {
		data.Concentrations, _ = parsePlateMapGrid(lines, idx)
	}

	if idx := findSectionLine(lines, "Limits of Quantification"); idx >= 0 && idx < len(lines) {
		parts := parseCSVLine(lines[idx])
		if len(parts) >= 2 {
			data.LLOQ = parseCSVNum(parts[0])
			data.ULOQ = parseCSVNum(parts[1])
		}
	}

	// Std1 concentration from first standard row (line index 2 in file = stnd01 row after header).
	if len(lines) > 2 {
		parts := parseCSVLine(lines[2])
		if len(parts) > 1 {
			data.Std1Conc = parseCSVNum(parts[1])
		}
	}
	return data, nil
}

// LoadAllAnalytePlateData loads plate data for every summary CSV in analysisDir/reports/.
func LoadAllAnalytePlateData(analysisDir string) ([]AnalytePlateData, error) {
	paths, err := FindSummaryCSVFiles(analysisDir)
	if err != nil {
		return nil, err
	}
	out := make([]AnalytePlateData, 0, len(paths))
	for _, p := range paths {
		d, err := LoadAnalytePlateDataFromSummaryCSV(p)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, nil
}

// BuildConcentrationGridFromValidation builds an 8×12 grid from validation_data rows.
func BuildConcentrationGridFromValidation(doc *Document, analyte string) PlateGrid {
	var grid PlateGrid
	for _, row := range doc.Validation {
		if row.Analyte != analyte || row.CalculatedConcentration == nil {
			continue
		}
		r := strings.ToUpper(row.WellRow)
		if len(r) != 1 || row.WellColumn < 1 || row.WellColumn > 12 {
			continue
		}
		idx := int(r[0] - 'A')
		v := *row.CalculatedConcentration
		grid[idx][row.WellColumn-1] = &v
	}
	return grid
}

func parseLimitFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &n
}

func gridValuesMasked(grid PlateGrid) []float64 {
	var vals []float64
	for r := 0; r < 8; r++ {
		for c := 0; c < 12; c++ {
			if r < 2 {
				continue
			}
			if r < 4 && c < 4 {
				continue
			}
			if grid[r][c] != nil && !math.IsNaN(*grid[r][c]) {
				vals = append(vals, *grid[r][c])
			}
		}
	}
	return vals
}

func nanMean(vals []float64) float64 {
	if len(vals) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func nanStd(vals []float64) float64 {
	if len(vals) < 2 {
		return math.NaN()
	}
	mean := nanMean(vals)
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func percentCV(vals []float64) float64 {
	mean := nanMean(vals)
	if math.IsNaN(mean) || mean == 0 {
		return math.NaN()
	}
	return 100 * nanStd(vals) / mean
}
