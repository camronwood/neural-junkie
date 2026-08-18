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
		{".venv-icon/lib/python3.14/site-packages/PIL/TiffImagePlugin.py", "TiffImagePlugin.py", true},
		{"lib/python3.12/site-packages/PIL/TiffImagePlugin.py", "TiffImagePlugin.py", true},
		{"cad_venv/lib/python3.11/site-packages/foo.py", "foo.py", true},
		{"internal/repo/analyzer.go", "analyzer.go", false},
		{"reader/libatikcameras.so", "libatikcameras.so", true},
		{"ui/package-lock.json", "package-lock.json", true},
		{"go.sum", "go.sum", true},
		{"Cargo.lock", "Cargo.lock", true},
		{"assets/photo.png", "photo.png", true},
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

func TestIsDependencyPath(t *testing.T) {
	if !IsDependencyPath("/Users/me/.venv-icon/lib/python3.14/site-packages/PIL/TiffImagePlugin.py") {
		t.Fatal("expected absolute site-packages path ignored")
	}
	if IsDependencyPath("internal/agent/plan_mode_prompt.go") {
		t.Fatal("project source must not be treated as dependency")
	}
}
