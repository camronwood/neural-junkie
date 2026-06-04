package scananalysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseResultsStringSignal(t *testing.T) {
	raw := []byte(`{
		"header_data": {},
		"experiment_data": {"product_name": "test"},
		"unknown_report_data": {
			"IL-6": [{
				"analyte": "IL-6",
				"well_label": "unk1",
				"replicates": [
					{"replicate_index": 0, "signal": "12345.67", "concentration": "1.5"}
				]
			}]
		}
	}`)
	doc, err := ParseResults(raw)
	if err != nil {
		t.Fatal(err)
	}
	rows := doc.UnknownReport["IL-6"]
	if len(rows) != 1 || len(rows[0].Replicates) != 1 {
		t.Fatal("expected one unknown row with one replicate")
	}
	if got := rows[0].Replicates[0].Signal; got != 12345.67 {
		t.Fatalf("signal: got %v want 12345.67", got)
	}
	if rows[0].Replicates[0].Concentration == nil || *rows[0].Replicates[0].Concentration != 1.5 {
		t.Fatalf("concentration: got %v want 1.5", rows[0].Replicates[0].Concentration)
	}
}

func TestNormalizeJSONNaN(t *testing.T) {
	raw := []byte(`{"header_data":{},"experiment_data":{"product_name":"test"},"validation_data":[{"analyte":"IL-6","signal":1,"well_row":"A","well_column":1,"well_type":"unknown","well_label":"unk1","calculated_concentration": NaN}]}`)
	out := NormalizeJSONNaN(raw)
	if strings.Contains(string(out), "NaN") {
		t.Fatalf("expected NaN replaced: %s", out)
	}
	doc, err := ParseResults(out)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ValidationAt("A1", "IL-6") == nil {
		t.Fatal("expected validation row")
	}
	if doc.ValidationAt("A1", "IL-6").CalculatedConcentration != nil {
		t.Fatal("expected null concentration")
	}
}

func TestLoadResultsFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	doc, err := LoadResults(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Experiment.ProductName == "" {
		t.Fatal("expected product name")
	}
	if len(doc.Analytes) == 0 {
		t.Fatal("expected analytes")
	}
	row := doc.ValidationAt("A1", "IL-6")
	if row == nil {
		t.Fatal("expected validation at A1 IL-6")
	}
	if row.WellType != "unknown" {
		t.Fatalf("expected unknown well type, got %q", row.WellType)
	}
}

func TestResolveAnalysisDir(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	results := filepath.Join(fixture, "reports", "results.json")

	dir, err := ResolveAnalysisDir(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if dir != fixture {
		t.Fatalf("expected %s, got %s", fixture, dir)
	}

	dir2, err := ResolveAnalysisDir(results)
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != fixture {
		t.Fatalf("expected %s from results path, got %s", fixture, dir2)
	}
}

func TestBuildAnalysisStats(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	doc, err := LoadResults(fixture)
	if err != nil {
		t.Fatal(err)
	}
	stats := BuildAnalysisStats(fixture, doc)
	out := FormatAnalysisMarkdown(stats)
	for _, part := range []string{"Scan analysis", "IL-6", "Dilution factor"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
}

func TestWellIDFromRowCol(t *testing.T) {
	if got := WellIDFromRowCol("A", 1); got != "A1" {
		t.Fatalf("got %q", got)
	}
	row, col, ok := ParseWellID("H12")
	if !ok || row != "H" || col != 12 {
		t.Fatalf("parse H12 failed: %q %d %v", row, col, ok)
	}
}

func TestResolveLinkedScanDirCombined(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "imageMetadata.json"), []byte(`{"metadata":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reports, "results.json"), []byte(`{"header_data":{},"experiment_data":{"product_name":"test"},"validation_data":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	linked := ResolveLinkedScanDir(dir)
	if linked != dir {
		t.Fatalf("expected linked scan dir %s, got %s", dir, linked)
	}
}
