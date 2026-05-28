package biology

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeScanAnalysisPath(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "testdata", "scan-analysis")
	if _, err := os.Stat(fixture); err != nil {
		t.Skip("testdata not present")
	}
	out, err := summarizeScanAnalysisPath(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"Scan analysis", "IL-6", "Product"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
	results := filepath.Join(fixture, "reports", "results.json")
	out2, err := summarizeScanAnalysisPath(results)
	if err != nil {
		t.Fatal(err)
	}
	if out2 == "" {
		t.Fatal("expected summary from results.json path")
	}

	csv := filepath.Join(fixture, "reports", "IL-6_summary_report.csv")
	if _, err := os.Stat(csv); err == nil {
		out3, err := summarizeScanAnalysisPath(csv)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out3, "IL-6") {
			t.Fatalf("expected IL-6 in csv summary: %s", out3)
		}
	}
}
