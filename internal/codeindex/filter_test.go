package codeindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsIndexableRelPath(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		{"internal/repo/analyzer.go", true},
		{"desktop/src/api/chatAPI.ts", true},
		{"docs/REPO_AGENTS.md", true},
		{"config.toml", true},
		{"docs/media/demo.mp4", false},
		{"reader/libatikcameras.so", false},
		{"ui/package-lock.json", false},
		{"Cargo.lock", false},
		{"go.sum", false},
		{"yarn.lock", false},
		{"bin/tool", false},
		{"node_modules/pkg/index.js", false},
		{"Lfa-Reader-Gui.arm64", false},
	}
	for _, tt := range tests {
		if got := IsIndexableRelPath(tt.rel); got != tt.want {
			t.Errorf("IsIndexableRelPath(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}

func TestLooksLikeBinary(t *testing.T) {
	if LooksLikeBinary([]byte("package main\n")) {
		t.Fatal("text should not look binary")
	}
	if !LooksLikeBinary([]byte{0x7f, 'E', 'L', 'F', 0, 1, 2}) {
		t.Fatal("NUL-containing bytes should look binary")
	}
}

func TestIsReadableSourceFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "ok.go")
	if err := os.WriteFile(src, []byte("package ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsReadableSourceFile(src, "ok.go") {
		t.Fatal("expected readable source")
	}
	bin := filepath.Join(dir, "blob.go")
	if err := os.WriteFile(bin, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}
	if IsReadableSourceFile(bin, "blob.go") {
		t.Fatal("NUL sniff should reject")
	}
}
