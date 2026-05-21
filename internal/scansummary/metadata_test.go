package scansummary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSummaryStats(t *testing.T) {
	doc := &Document{
		Metadata: []WellMeta{
			{
				ImageName: "A1",
				Spots: []Spot{
					{Analyte: "BLANK", Row: "1", Column: "1"},
					{Analyte: "IL-6", Row: "2", Column: "2"},
				},
			},
			{ImageName: "A2", Spots: nil},
		},
	}
	stats := BuildSummaryStats("/data/HIF12A-1-254292-1 - summary", doc)
	if stats.WellCount != 2 {
		t.Fatalf("well count: %d", stats.WellCount)
	}
	if stats.AnalyteCounts["BLANK"] != 1 {
		t.Fatalf("BLANK count: %d", stats.AnalyteCounts["BLANK"])
	}
	if len(stats.WellsMissing) != 1 || stats.WellsMissing[0] != "A2" {
		t.Fatalf("wells missing: %v", stats.WellsMissing)
	}
}

func TestValidateWellID(t *testing.T) {
	if !ValidateWellID("A1") || !ValidateWellID("H12") {
		t.Fatal("valid wells rejected")
	}
	if ValidateWellID("I1") || ValidateWellID("A13") {
		t.Fatal("invalid wells accepted")
	}
}

func TestResolveSummaryDir(t *testing.T) {
	dir := t.TempDir()
	meta := filepath.Join(dir, MetadataFileName)
	if err := os.WriteFile(meta, []byte(`{"metadata":[{"imageName":"A1","spots":[]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveSummaryDir(dir)
	if err != nil || got != dir {
		t.Fatalf("dir resolve: %v %q", err, got)
	}
	got2, err := ResolveSummaryDir(meta)
	if err != nil || got2 != dir {
		t.Fatalf("file resolve: %v %q", err, got2)
	}
}
