package scananalysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSummaryCSVFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis", "reports", "IL-6_summary_report.csv")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Skip("csv fixture not present")
	}
	doc, err := ParseSummaryCSV(string(raw), "IL-6")
	if err != nil {
		t.Fatal(err)
	}
	idx := BuildIndexes(doc)
	if idx.ValidationAt("A1", "IL-6") == nil {
		t.Fatal("expected validation at A1")
	}
	if idx.ConcentrationAt("A1", "IL-6") == nil {
		t.Fatal("expected concentration at A1")
	}
	if len(doc.StandardReport["IL-6"]) == 0 {
		t.Fatal("expected standard rows")
	}
	if doc.LimitsOfQuant["IL-6"].LLOQ == "" {
		t.Fatal("expected LLOQ")
	}
}

func TestAnalyteFromSummaryCSVPath(t *testing.T) {
	a, ok := AnalyteFromSummaryCSVPath("reports/IL-6_summary_report.csv")
	if !ok || a != "IL-6" {
		t.Fatalf("got %q %v", a, ok)
	}
}

func TestLoadAnalysisCSVOnly(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "scan-analysis", "reports", "IL-6_summary_report.csv")
	if _, err := os.Stat(src); err != nil {
		t.Skip("csv fixture not present")
	}
	dir := t.TempDir()
	reports := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(reports, "IL-6_summary_report.csv")
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	doc, source, err := LoadAnalysis(dir)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceCSV {
		t.Fatalf("expected csv source, got %s", source)
	}
	if doc.ValidationAt("A1", "IL-6") == nil {
		t.Fatal("expected validation from csv-only export")
	}

	out, _, err := LoadAnalysis(dst)
	if err != nil {
		t.Fatal(err)
	}
	if out.ValidationAt("B1", "IL-6") == nil {
		t.Fatal("expected validation when loading csv path directly")
	}
}

func TestProcessReportExcerpt(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	excerpt := ProcessReportExcerpt(fixture, 5)
	if excerpt == "" {
		t.Fatal("expected process report excerpt")
	}
	if !strings.Contains(excerpt, "Starting analysis") {
		t.Fatalf("unexpected excerpt: %s", excerpt)
	}
}

func TestResolveLinkedScanExportSubdir(t *testing.T) {
	dir := t.TempDir()
	scanExport := filepath.Join(dir, "scan-export")
	if err := os.MkdirAll(scanExport, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scanExport, "imageMetadata.json"), []byte(`{"metadata":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reports := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reports, "IL-6_summary_report.csv"), []byte("Standard Report\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linked := ResolveLinkedScanDir(dir)
	if linked != "scan-export" {
		t.Fatalf("expected scan-export, got %q", linked)
	}
}
