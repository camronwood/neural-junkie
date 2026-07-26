package repo

import "testing"

func TestShouldIgnoreEntry(t *testing.T) {
	tests := []struct {
		relPath string
		name    string
		want    bool
	}{
		{"node_modules/pkg/index.js", "index.js", true},
		{"desktop/src-tauri/binaries/nj-server-aarch64-apple-darwin", "nj-server-aarch64-apple-darwin", true},
		{"server", "server", true},
		{"cmd/server/main.go", "main.go", false},
		{"docs/media/demo.mp4", "demo.mp4", true},
		{"docs/REPO_AGENTS.md", "REPO_AGENTS.md", false},
		{"internal/repo/analyzer.go", "analyzer.go", false},
		{"desktop/src-tauri/icons/icon.icns", "icon.icns", true},
		{".scenario-baseline/Makefile", "Makefile", true},
		{"Makefile", "Makefile", false},
	}
	for _, tt := range tests {
		if got := ShouldIgnoreEntry(tt.relPath, tt.name); got != tt.want {
			t.Errorf("ShouldIgnoreEntry(%q, %q) = %v, want %v", tt.relPath, tt.name, got, tt.want)
		}
	}
}

func TestIsScenarioBaselinePath(t *testing.T) {
	if !IsScenarioBaselinePath(".scenario-baseline/Makefile") {
		t.Fatal("expected baseline path")
	}
	if !IsScenarioBaselinePath("nested/.scenario-baseline/x") {
		t.Fatal("expected nested baseline path")
	}
	if IsScenarioBaselinePath("Makefile") {
		t.Fatal("live Makefile must not be treated as baseline")
	}
}
