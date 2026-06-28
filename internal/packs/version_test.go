package packs

import "testing"

func TestUpdateAvailable(t *testing.T) {
	tests := []struct {
		installed, catalog string
		want               bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"", "1.0.0", true},
		{"1.0.0", "", false},
		{"v1.0.0", "1.0.1", true},
	}
	for _, tc := range tests {
		if got := UpdateAvailable(tc.installed, tc.catalog); got != tc.want {
			t.Errorf("UpdateAvailable(%q, %q) = %v, want %v", tc.installed, tc.catalog, got, tc.want)
		}
	}
}
