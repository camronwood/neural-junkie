package scananalysis

import (
	"path/filepath"
	"testing"
)

func TestRun12PlexQCOnFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	doc, _, err := LoadAnalysis(fixture)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	report, err := Run12PlexQC(fixture, doc, QCOptions{})
	if err != nil {
		t.Fatalf("Run12PlexQC: %v", err)
	}
	if report.PlateLabel == "" {
		t.Fatal("expected plate label")
	}
	if len(report.Analytes) == 0 {
		t.Fatal("expected analyte rows")
	}
	t.Logf("overall pass=%v analytes=%d", report.OverallPass, len(report.Analytes))
}

func TestLLOQThresholdFor(t *testing.T) {
	v, ok := LLOQThresholdFor("IL-6")
	if !ok || v != 0.05 {
		t.Fatalf("IL-6 threshold: got %v ok=%v", v, ok)
	}
}

func TestLoadAnalytePlateDataFromSummaryCSV(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis", "reports", "IL-6_summary_report.csv")
	data, err := LoadAnalytePlateDataFromSummaryCSV(fixture)
	if err != nil {
		t.Fatalf("LoadAnalytePlateDataFromSummaryCSV: %v", err)
	}
	if data.Analyte != "IL-6" {
		t.Fatalf("analyte=%q", data.Analyte)
	}
}
