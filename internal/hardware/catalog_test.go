package hardware

import "testing"

func TestParseSizeHintGB(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"~9 GB", 9, true},
		{"~4.5 GB (Q4)", 4.5, true},
		{"~40 GB", 40, true},
		{"", 0, false},
		{"small", 0, false},
	}
	for _, tc := range tests {
		got, ok := ParseSizeHintGB(tc.in)
		if ok != tc.ok || (tc.ok && got != tc.want) {
			t.Errorf("ParseSizeHintGB(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEstimatedRAMGBFromModelSize(t *testing.T) {
	if got := EstimatedRAMGBFromModelSize(9); got != 16 {
		t.Fatalf("9 GB model: got %d want 16", got)
	}
	if got := EstimatedRAMGBFromModelSize(4.5); got != 10 {
		t.Fatalf("4.5 GB model: got %d want 10", got)
	}
}

func TestTierForMemoryGB(t *testing.T) {
	cases := []struct {
		gb   int
		want Tier
	}{
		{4, TierMinimal},
		{7, TierMinimal},
		{8, TierLight},
		{15, TierLight},
		{16, TierRecommended},
		{31, TierRecommended},
		{32, TierHeavy},
		{64, TierHeavy},
	}
	for _, tc := range cases {
		if got := TierForMemoryGB(tc.gb); got != tc.want {
			t.Errorf("TierForMemoryGB(%d) = %q, want %q", tc.gb, got, tc.want)
		}
	}
}

func TestLookupModel(t *testing.T) {
	row, err := LookupModel("qwen3.5:27b")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("expected catalog row")
	}
	if row.SizeHint == "" {
		t.Fatal("expected size_hint")
	}
	if row.EstimatedDiskGB != 17 {
		t.Fatalf("disk gb %v", row.EstimatedDiskGB)
	}
	if row.EstimatedRAMGB != 26 {
		t.Fatalf("ram gb %v", row.EstimatedRAMGB)
	}

	missing, err := LookupModel("not-a-real-model-tag")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatal("expected nil for unknown model")
	}
}

func TestRecommendationsForTier(t *testing.T) {
	light := RecommendationsForTier(TierLight, 12)
	if light["developer"].PrimaryModel != "qwen3.5:9b" {
		t.Fatalf("developer light: %s", light["developer"].PrimaryModel)
	}
	rec := RecommendationsForTier(TierRecommended, 16)
	if rec["developer"].PrimaryModel != "qwen3.5:9b" {
		t.Fatalf("developer recommended: %s", rec["developer"].PrimaryModel)
	}
}

func TestTotalMemoryBytes(t *testing.T) {
	n, err := TotalMemoryBytes()
	if err != nil {
		t.Skipf("memory probe unavailable: %v", err)
	}
	if n == 0 {
		t.Fatal("expected non-zero memory on this host")
	}
	gb := MemoryGB(n)
	if gb < 1 {
		t.Fatalf("MemoryGB = %d", gb)
	}
}
