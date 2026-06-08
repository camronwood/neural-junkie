package scananalysis

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ValidationReportFileName = "validation_report.csv"
	AllSpotsFileName         = "allspots.csv"
	blankStandardConcentration = 1e-10
)

// WriteValidationReportCSV writes Phoenix validation_report.csv from a parsed Document.
func WriteValidationReportCSV(doc *Document, path string) error {
	if doc == nil {
		return fmt.Errorf("nil document")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{
		"well",
		"well type",
		"standard concentration",
		"calculated concentration",
		"analyte",
		"replicate mean intensity",
	}); err != nil {
		return err
	}
	for _, row := range doc.Validation {
		if err := w.Write([]string{
			wellIDPadded(row.WellRow, row.WellColumn),
			validationCSVWellType(row),
			formatStandardConcentration(row),
			formatCalculatedConcentration(row.CalculatedConcentration),
			row.Analyte,
			formatFloat(row.Signal),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// WriteAllSpotsCSV writes Phoenix allspots.csv from spot_intensities.
func WriteAllSpotsCSV(doc *Document, path string) error {
	if doc == nil {
		return fmt.Errorf("nil document")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{
		"well_row",
		"well_column",
		"well_replicate_index",
		"well_type",
		"well_label",
		"analyte",
		"row",
		"column",
		"signal",
		"background",
		"signal_intensity_algorithm",
	}); err != nil {
		return err
	}
	for _, spot := range doc.SpotIntensities {
		if err := w.Write([]string{
			spot.WellRow,
			strconv.Itoa(spot.WellColumn),
			strconv.Itoa(spot.WellReplicateIndex),
			spot.WellType,
			spot.WellLabel,
			spot.Analyte,
			strconv.Itoa(spot.Row),
			strconv.Itoa(spot.Column),
			formatFloat(spot.Signal),
			formatFloat(spot.Background),
			spot.SignalIntensityAlgorithm,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func wellIDPadded(row string, col int) string {
	return fmt.Sprintf("%s%02d", row, col)
}

func validationCSVWellType(row ValidationRow) string {
	if strings.EqualFold(row.WellType, "blank") || strings.EqualFold(row.WellLabel, "blank") {
		return "blank"
	}
	return normalizeExportWellLabel(row.WellLabel)
}

func normalizeExportWellLabel(label string) string {
	label = strings.TrimSpace(label)
	lower := strings.ToLower(label)
	switch {
	case strings.HasPrefix(lower, "stnd"), strings.HasPrefix(lower, "std"):
		prefix := "stnd"
		digits := strings.TrimPrefix(strings.TrimPrefix(lower, "stnd"), "std")
		if n, err := strconv.Atoi(digits); err == nil {
			return fmt.Sprintf("%s%02d", prefix, n)
		}
	case strings.HasPrefix(lower, "unk"):
		if n, err := strconv.Atoi(strings.TrimPrefix(lower, "unk")); err == nil {
			return fmt.Sprintf("unk%02d", n)
		}
	}
	return strings.ToLower(label)
}

func formatStandardConcentration(row ValidationRow) string {
	if strings.EqualFold(row.WellType, "blank") || strings.EqualFold(row.WellLabel, "blank") {
		return formatFloat(blankStandardConcentration)
	}
	if row.KnownConcentration != nil {
		return formatFloat(*row.KnownConcentration)
	}
	return ""
}

func formatCalculatedConcentration(v *float64) string {
	if v == nil || math.IsNaN(*v) {
		return "nan"
	}
	return formatFloat(*v)
}

func formatFloat(v float64) string {
	if math.Mod(v, 1) == 0 {
		return strconv.FormatFloat(v, 'f', 1, 64)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
