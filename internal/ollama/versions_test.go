package ollama

import "testing"

func TestParseOllamaVersion(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"ollama version is 0.17.6", "0.17.6", true},
		{"ollama version is 0.17.6\nWarning: client version is 0.32.0", "0.32.0", true},
		{"v0.30.5", "0.30.5", true},
		{"", "", false},
		{"garbage", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseOllamaVersion(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("ParseOllamaVersion(%q) = %q,%v want %q,%v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	if CompareSemver("0.17.6", "0.30.0") >= 0 {
		t.Fatal("0.17.6 should be < 0.30.0")
	}
	if CompareSemver("0.32.0", "0.30.0") < 0 {
		t.Fatal("0.32.0 should be >= 0.30.0")
	}
	if CompareSemver("0.32.0", RecommendedOllamaVersion) != 0 {
		t.Fatal("recommended pin mismatch")
	}
}

func TestNeedsUpdateAndMeetsMinimum(t *testing.T) {
	if !NeedsUpdate("0.17.6") {
		t.Fatal("expected NeedsUpdate for 0.17.6")
	}
	if NeedsUpdate(RecommendedOllamaVersion) {
		t.Fatal("recommended should not need update")
	}
	if MeetsMinimum("0.17.6") {
		t.Fatal("0.17.6 below min")
	}
	if !MeetsMinimum("0.30.0") {
		t.Fatal("0.30.0 should meet min")
	}
	if !MeetsVersion("0.32.0", "0.30.0") {
		t.Fatal("MeetsVersion failed")
	}
	if MeetsVersion("0.17.6", "0.30.0") {
		t.Fatal("MeetsVersion should fail")
	}
}

func TestFetchOllamaScriptPinSync(t *testing.T) {
	if RecommendedOllamaTag != "v"+RecommendedOllamaVersion {
		t.Fatalf("tag %q", RecommendedOllamaTag)
	}
	if RecommendedOllamaVersion != "0.32.0" {
		t.Fatalf("bump fetch-ollama.sh when changing RecommendedOllamaVersion (got %s)", RecommendedOllamaVersion)
	}
}
