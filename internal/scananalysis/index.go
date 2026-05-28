package scananalysis

import (
	"fmt"
	"sort"
	"strconv"
)

// WellIDFromRowCol builds A1-style well id from row letter and column number.
func WellIDFromRowCol(row string, col int) string {
	return fmt.Sprintf("%s%d", row, col)
}

// WellAnalyteKey builds a lookup key for well + analyte.
func WellAnalyteKey(wellID, analyte string) string {
	return wellID + "|" + analyte
}

// BuildIndexes constructs lookup maps from a parsed Document.
func BuildIndexes(doc *Document) *IndexedDocument {
	idx := &IndexedDocument{
		Document:           *doc,
		ByWellAnalyte:      make(map[string]*ValidationRow),
		ByWell:             make(map[string][]ValidationRow),
		SpotsByWellAnalyte: make(map[string][]SpotIntensityRow),
	}

	analyteSet := make(map[string]struct{})
	for k := range doc.Experiment.InitialConcentrations {
		analyteSet[k] = struct{}{}
	}
	for _, row := range doc.Validation {
		analyteSet[row.Analyte] = struct{}{}
		wellID := WellIDFromRowCol(row.WellRow, row.WellColumn)
		key := WellAnalyteKey(wellID, row.Analyte)
		rowCopy := row
		idx.ByWellAnalyte[key] = &rowCopy
		idx.ByWell[wellID] = append(idx.ByWell[wellID], rowCopy)
	}
	for _, spot := range doc.SpotIntensities {
		wellID := WellIDFromRowCol(spot.WellRow, spot.WellColumn)
		key := WellAnalyteKey(wellID, spot.Analyte)
		idx.SpotsByWellAnalyte[key] = append(idx.SpotsByWellAnalyte[key], spot)
	}

	idx.Analytes = make([]string, 0, len(analyteSet))
	for a := range analyteSet {
		idx.Analytes = append(idx.Analytes, a)
	}
	sort.Strings(idx.Analytes)
	return idx
}

// ValidationAt returns the validation row for a well and analyte, if present.
func (idx *IndexedDocument) ValidationAt(wellID, analyte string) *ValidationRow {
	return idx.ByWellAnalyte[WellAnalyteKey(wellID, analyte)]
}

// ConcentrationAt returns calculated concentration for well+analyte, or nil.
func (idx *IndexedDocument) ConcentrationAt(wellID, analyte string) *float64 {
	if row := idx.ValidationAt(wellID, analyte); row != nil {
		return row.CalculatedConcentration
	}
	return nil
}

// ParseWellID splits A1 into row "A" and column 1.
func ParseWellID(wellID string) (row string, col int, ok bool) {
	if len(wellID) < 2 {
		return "", 0, false
	}
	row = wellID[:1]
	c, err := strconv.Atoi(wellID[1:])
	if err != nil || c < 1 || c > 12 {
		return "", 0, false
	}
	return row, c, true
}
