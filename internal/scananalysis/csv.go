package scananalysis

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const summaryCSVSuffix = "_summary_report.csv"

var summaryCSVNameRe = regexp.MustCompile(`(?i)^(.+)_summary_report\.csv$`)

// IsSummaryCSVPath reports whether path is a Phoenix per-analyte summary CSV.
func IsSummaryCSVPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), summaryCSVSuffix)
}

// AnalyteFromSummaryCSVPath extracts the analyte name from reports/{analyte}_summary_report.csv.
func AnalyteFromSummaryCSVPath(path string) (string, bool) {
	base := path
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		base = path[i+1:]
	}
	m := summaryCSVNameRe.FindStringSubmatch(base)
	if len(m) < 2 || m[1] == "" {
		return "", false
	}
	return m[1], true
}

func parseCSVLine(line string) []string {
	var out []string
	cur := strings.Builder{}
	inQuotes := false
	for _, ch := range line {
		switch {
		case ch == '"':
			inQuotes = !inQuotes
		case ch == ',' && !inQuotes:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(ch)
		}
	}
	out = append(out, strings.TrimSpace(cur.String()))
	return out
}

func parseCSVNum(raw string) *float64 {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" || s == "nan" || s == "null" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseCSVBool(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "true")
}

func wellTypeFromLabel(label string) string {
	l := strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.HasPrefix(l, "stnd"), strings.HasPrefix(l, "std"):
		return "standard"
	case l == "blank":
		return "blank"
	default:
		return "unknown"
	}
}

func normalizeWellLabel(label string) string {
	label = strings.TrimSpace(label)
	re := regexp.MustCompile(`(?i)^(stnd|std|unk)(\d+)$`)
	if m := re.FindStringSubmatch(label); len(m) == 3 {
		prefix := "unk"
		if strings.HasPrefix(strings.ToLower(m[1]), "st") {
			prefix = "stnd"
		}
		n, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s%02d", prefix, n)
	}
	return strings.ToLower(label)
}

type plateGrid map[string]map[int]string

func parsePlateGrid(lines []string, startIdx int) (plateGrid, int) {
	grid := make(plateGrid)
	i := startIdx
	if i >= len(lines) {
		return grid, i
	}
	i++ // section title
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		if strings.HasSuffix(line, "Report") ||
			strings.HasPrefix(line, "Plate Map") ||
			strings.HasPrefix(line, "Limits of") ||
			strings.HasPrefix(line, "Limit of") ||
			strings.HasPrefix(line, "NOTE:") {
			break
		}
		cols := parseCSVLine(line)
		if len(cols) == 0 {
			i++
			continue
		}
		row := strings.ToUpper(cols[0])
		if len(row) != 1 || row < "A" || row > "H" {
			i++
			continue
		}
		byCol := make(map[int]string)
		for c := 1; c < len(cols) && c <= 12; c++ {
			byCol[c] = cols[c]
		}
		grid[row] = byCol
		i++
	}
	return grid, i
}

// ParseSummaryCSV parses one Phoenix *_summary_report.csv into a Document (single analyte).
func ParseSummaryCSV(raw string, analyte string) (*Document, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	doc := &Document{
		StandardReport: make(map[string][]StandardRow),
		UnknownReport:  make(map[string][]UnknownRow),
		LimitsOfQuant:  make(map[string]LimitsOfQuantification),
		FitParameters:  make(map[string]FitParameters),
	}
	doc.StandardReport[analyte] = nil
	doc.UnknownReport[analyte] = nil

	var plateConcentrations, plateLabels, plateIntensities plateGrid

	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		switch line {
		case "Standard Report":
			i += 2
			for i < len(lines) {
				row := strings.TrimSpace(lines[i])
				if row == "" || strings.HasPrefix(row, "NOTE:") || row == "Unknown Report" {
					break
				}
				cols := parseCSVLine(row)
				if len(cols) < 2 {
					break
				}
				doc.StandardReport[analyte] = append(doc.StandardReport[analyte], StandardRow{
					Analyte:                              analyte,
					WellLabel:                            cols[0],
					Concentration:                        derefFloat(parseCSVNum(cols[1])),
					Replicates:                           map[string]float64{},
					MeanReplicateIntensity:               parseCSVNum(colAt(cols, 4)),
					MeanReplicateCalculatedConcentration: parseCSVNum(colAt(cols, 5)),
					PercentBias:                          parseCSVNum(colAt(cols, 6)),
					WithinLimitsOfQuantificationV2:       parseCSVBool(colAt(cols, 7)),
					UpperPercentDifferenceV2:             parseCSVNum(colAt(cols, 8)),
					LowerPercentDifferenceV2:             parseCSVNum(colAt(cols, 9)),
				})
				i++
			}
		case "Unknown Report":
			i += 2
			for i < len(lines) {
				row := strings.TrimSpace(lines[i])
				if row == "" || strings.HasPrefix(row, "Plate Map") {
					break
				}
				cols := parseCSVLine(row)
				if len(cols) < 2 {
					break
				}
				doc.UnknownReport[analyte] = append(doc.UnknownReport[analyte], UnknownRow{
					Analyte:   analyte,
					WellLabel: cols[0],
					Replicates: []UnknownReplicate{{
						ReplicateIndex: 0,
						Signal:         derefFloat(parseCSVNum(cols[1])),
						Concentration:  parseCSVNum(colAt(cols, 2)),
					}},
					MeanReplicateConcentration:    parseCSVNum(colAt(cols, 3)),
					StdevOfReplicateConcentration: parseCSVNum(colAt(cols, 4)),
					WithinLimitsOfQuantification:  parseCSVBool(colAt(cols, 5)),
					ConcentrationUnit:             "pg/ml",
				})
				i++
			}
		case "Plate Map Concentrations":
			plateConcentrations, i = parsePlateGrid(lines, i)
		case "Plate Map Labels":
			plateLabels, i = parsePlateGrid(lines, i)
		case "Plate Map Intensities":
			plateIntensities, i = parsePlateGrid(lines, i)
		default:
			if strings.HasPrefix(line, "Limits of Quantification") {
				i += 2
				if i < len(lines) {
					cols := parseCSVLine(lines[i])
					doc.LimitsOfQuant[analyte] = LimitsOfQuantification{
						LLOQ:               colAt(cols, 0),
						ULOQ:               colAt(cols, 1),
						ConcentrationUnits: "pg/ml",
					}
				}
				i++
			} else if strings.HasPrefix(line, "Limit of Detection") {
				i += 2
				if i < len(lines) && doc.LimitsOfQuant[analyte].LLOQ != "" {
					loq := doc.LimitsOfQuant[analyte]
					loq.LOD = parseCSVLine(lines[i])[0]
					loq.LODLabel = "Calculated LOD"
					doc.LimitsOfQuant[analyte] = loq
				}
				i++
			} else {
				i++
			}
		}
	}

	for row, cols := range plateLabels {
		for col, label := range cols {
			if label == "" {
				continue
			}
			if _, _, ok := ParseWellID(WellIDFromRowCol(row, col)); !ok {
				continue
			}
			conc := parseCSVNum(plateCell(plateConcentrations, row, col))
			intensity := parseCSVNum(plateCell(plateIntensities, row, col))
			var unkConc *float64
			for _, u := range doc.UnknownReport[analyte] {
				if normalizeWellLabel(u.WellLabel) == normalizeWellLabel(label) {
					unkConc = u.MeanReplicateConcentration
					break
				}
			}
			calc := conc
			if calc == nil {
				calc = unkConc
			}
			sig := 0.0
			if intensity != nil {
				sig = *intensity
			}
			doc.Validation = append(doc.Validation, ValidationRow{
				Analyte:                 analyte,
				Signal:                  sig,
				WellRow:                 row,
				WellColumn:              col,
				WellType:                wellTypeFromLabel(label),
				WellLabel:               label,
				CalculatedConcentration: calc,
			})
		}
	}

	if len(doc.Validation) == 0 && len(doc.StandardReport[analyte]) == 0 && len(doc.UnknownReport[analyte]) == 0 {
		return nil, fmt.Errorf("empty or invalid summary CSV for analyte %s", analyte)
	}
	doc.Experiment.InitialConcentrations = map[string]float64{analyte: 0}
	return doc, nil
}

func colAt(cols []string, idx int) string {
	if idx < 0 || idx >= len(cols) {
		return ""
	}
	return cols[idx]
}

func plateCell(grid plateGrid, row string, col int) string {
	if grid == nil {
		return ""
	}
	if byCol, ok := grid[row]; ok {
		return byCol[col]
	}
	return ""
}

func derefFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// MergeDocuments combines multiple analyte documents (typically from CSV files).
func MergeDocuments(docs []*Document) *Document {
	if len(docs) == 0 {
		return &Document{
			StandardReport: make(map[string][]StandardRow),
			UnknownReport:  make(map[string][]UnknownRow),
			LimitsOfQuant:  make(map[string]LimitsOfQuantification),
			FitParameters:  make(map[string]FitParameters),
		}
	}
	if len(docs) == 1 {
		return docs[0]
	}
	merged := &Document{
		Header:          docs[0].Header,
		Experiment:      docs[0].Experiment,
		StandardReport:  make(map[string][]StandardRow),
		UnknownReport:   make(map[string][]UnknownRow),
		LimitsOfQuant:   make(map[string]LimitsOfQuantification),
		FitParameters:   make(map[string]FitParameters),
		SpotIntensities: nil,
	}
	merged.Experiment.InitialConcentrations = make(map[string]float64)
	for _, doc := range docs {
		for k, v := range doc.Experiment.InitialConcentrations {
			merged.Experiment.InitialConcentrations[k] = v
		}
		for k, v := range doc.StandardReport {
			merged.StandardReport[k] = v
		}
		for k, v := range doc.UnknownReport {
			merged.UnknownReport[k] = v
		}
		for k, v := range doc.LimitsOfQuant {
			merged.LimitsOfQuant[k] = v
		}
		for k, v := range doc.FitParameters {
			merged.FitParameters[k] = v
		}
		merged.Validation = append(merged.Validation, doc.Validation...)
		merged.SpotIntensities = append(merged.SpotIntensities, doc.SpotIntensities...)
		if merged.Experiment.ProductName == "" && doc.Experiment.ProductName != "" {
			merged.Experiment.ProductName = doc.Experiment.ProductName
		}
		if merged.Experiment.DilutionFactor == 0 && doc.Experiment.DilutionFactor != 0 {
			merged.Experiment.DilutionFactor = doc.Experiment.DilutionFactor
		}
		if merged.Experiment.PlateBarcode == "" && doc.Experiment.PlateBarcode != "" {
			merged.Experiment.PlateBarcode = doc.Experiment.PlateBarcode
		}
	}
	return merged
}
