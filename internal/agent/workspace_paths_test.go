package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestDetectBareFilenames(t *testing.T) {
	t.Parallel()
	paths := DetectFilePaths("Emit [FILE_CHANGE] for tailwind.config.js and src/App.tsx")
	want := map[string]bool{"tailwind.config.js": true, "src/App.tsx": true}
	for p := range want {
		found := false
		for _, got := range paths {
			if got == p {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("DetectFilePaths missing %q in %v", p, paths)
		}
	}
}

func TestDetectFilePaths_atFilePrefersFullPath(t *testing.T) {
	t.Parallel()
	paths := DetectFilePaths("What does the main function in @file:core/sample/main.go do?")
	foundFull := false
	for _, p := range paths {
		if p == "core/sample/main.go" {
			foundFull = true
		}
		if p == "main.go" {
			t.Fatalf("basename-only main.go should not win over @file:core/sample/main.go; got %v", paths)
		}
	}
	if !foundFull {
		t.Fatalf("expected core/sample/main.go in %v", paths)
	}
}

func TestExpandPathRanges_numberedDirsWithBasename(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		dir := filepath.Join(root, "noise", fmt.Sprintf("pkg%04d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "util.go"), []byte(fmt.Sprintf("package pkg%04d\n", i)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	content := "in noise/pkg0000 through noise/pkg0009 rename each UtilN in util.go (10 files)"
	paths := DetectFilePathsInWorkspace(content, root)
	want := map[string]bool{}
	for i := 0; i < 10; i++ {
		want[fmt.Sprintf("noise/pkg%04d/util.go", i)] = true
	}
	for p := range want {
		found := false
		for _, got := range paths {
			if got == p {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %v", p, paths)
		}
	}
	n := AppendReferencedFiles(&strings.Builder{}, content, root)
	if n < 8 {
		t.Fatalf("expected expanded referenced seeds, got %d", n)
	}
}

func TestAppendImplementationSeedFiles_loadsTailwind(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := "module.exports = { darkMode: 'class' }\n"
	if err := os.WriteFile(dir+"/tailwind.config.js", []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	msg := &protocol.Message{Content: "please implement light/dark themes"}
	n := AppendImplementationSeedFiles(&b, nil, msg, dir, protocol.AgentTypeFrontend, nil)
	if n < 1 {
		t.Fatalf("expected at least tailwind loaded, got %d", n)
	}
	out := b.String()
	if !strings.Contains(out, "darkMode") {
		t.Fatalf("expected tailwind content in prompt, got %q", out)
	}
}
