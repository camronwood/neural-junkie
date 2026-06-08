package scananalysis

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteValidationReportCSV_matchesPhoenixFixture(t *testing.T) {
	fixture := os.Getenv("PHOENIX_VALIDATION_FIXTURE")
	if fixture == "" {
		fixture = "/Users/camronwood/development/phoenix-customer-cli-downloads"
	}
	resultsPath := filepath.Join(fixture, "results.json")
	expectedPath := filepath.Join(fixture, "validation", "reports", "validation_report.csv")
	if _, err := os.Stat(resultsPath); err != nil {
		t.Skip("fixture results.json not available")
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Skip("fixture validation_report.csv not available")
	}

	doc, err := loadFixtureDocument(t, resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outPath := filepath.Join(dir, "validation_report.csv")
	if err := WriteValidationReportCSV(doc, outPath); err != nil {
		t.Fatal(err)
	}
	gotRows, err := readCSVRows(outPath)
	if err != nil {
		t.Fatal(err)
	}
	wantRows, err := readCSVRows(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	gotMap := csvDataRows(gotRows[1:])
	wantMap := csvDataRows(wantRows[1:])
	if len(gotMap) != len(wantMap) {
		t.Fatalf("row count: got %d want %d", len(gotMap), len(wantMap))
	}
	for k, want := range wantMap {
		got, ok := gotMap[k]
		if !ok {
			t.Fatalf("missing row %q", k)
		}
		for i, w := range want {
			g := got[i]
			if w == "nan" && (g == "nan" || g == "") {
				continue
			}
			if g != w {
				t.Fatalf("row %q col %d: got %q want %q", k, i, g, w)
			}
		}
	}
}

func csvDataRows(rows [][]string) map[string][]string {
	out := make(map[string][]string, len(rows))
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		key := row[0] + "|" + row[4] // well|analyte
		out[key] = row
	}
	return out
}

func TestWriteAllSpotsCSV_rowCountMatchesFixture(t *testing.T) {
	fixture := os.Getenv("PHOENIX_VALIDATION_FIXTURE")
	if fixture == "" {
		fixture = "/Users/camronwood/development/phoenix-customer-cli-downloads"
	}
	resultsPath := filepath.Join(fixture, "results.json")
	expectedPath := filepath.Join(fixture, "validation", "reports", "allspots.csv")
	if _, err := os.Stat(resultsPath); err != nil {
		t.Skip("fixture results.json not available")
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Skip("fixture allspots.csv not available")
	}

	doc, err := loadFixtureDocument(t, resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outPath := filepath.Join(dir, "allspots.csv")
	if err := WriteAllSpotsCSV(doc, outPath); err != nil {
		t.Fatal(err)
	}
	gotRows, err := readCSVRows(outPath)
	if err != nil {
		t.Fatal(err)
	}
	wantRows, err := readCSVRows(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotRows) != len(wantRows) {
		t.Fatalf("row count: got %d want %d", len(gotRows), len(wantRows))
	}
}

func loadFixtureDocument(t *testing.T, resultsPath string) (*Document, error) {
	t.Helper()
	dir := t.TempDir()
	reportsDir := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(resultsPath)
	if err != nil {
		return nil, err
	}
	dest := filepath.Join(reportsDir, ResultsFileName)
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		return nil, err
	}
	idx, err := LoadResults(dir)
	if err != nil {
		return nil, err
	}
	return &idx.Document, nil
}

func readCSVRows(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return csv.NewReader(f).ReadAll()
}
