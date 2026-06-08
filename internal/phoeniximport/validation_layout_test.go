package phoeniximport

import "testing"

func TestValidationSiblingDir(t *testing.T) {
	tests := []struct {
		summary string
		want    string
	}{
		{"HIF12A-1-260457-1-summary", "HIF12A-1-260457-1-validation"},
		{"phoenix-abc-summary", "phoenix-abc-validation"},
		{"custom-run", "custom-run-validation"},
	}
	for _, tc := range tests {
		if got := ValidationSiblingDir(tc.summary); got != tc.want {
			t.Fatalf("ValidationSiblingDir(%q) = %q, want %q", tc.summary, got, tc.want)
		}
	}
}

func TestDefaultSummaryOutputDir(t *testing.T) {
	raw := []byte(`{"plateBarcode":"HIF12A-1-260457-1","analysisName":"run-1"}`)
	if got := defaultSummaryOutputDir("abc", raw); got != "HIF12A-1-260457-1-summary" {
		t.Fatalf("got %q", got)
	}
	if got := defaultSummaryOutputDir("abc", nil); got != "phoenix-abc-summary" {
		t.Fatalf("fallback got %q", got)
	}
}
