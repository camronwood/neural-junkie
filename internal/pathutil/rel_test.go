package pathutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/pathutil"
)

func TestResolveRelWithinRoot_stripsRedundantFixturePrefix(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "react-vite-corrupt-appjs")
	if err := os.MkdirAll(filepath.Join(fixture, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	appJS := filepath.Join(fixture, "src", "App.js")
	if err := os.WriteFile(appJS, []byte("export default function App() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := pathutil.ResolveRelWithinRoot(fixture, "scenarios/fixtures/react-vite-corrupt-appjs/src/App.js")
	if err != nil {
		t.Fatalf("ResolveRelWithinRoot: %v", err)
	}
	want, err := filepath.Abs(appJS)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveRelWithinRoot_directRelative(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := pathutil.ResolveRelWithinRoot(root, "src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(path)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveRelWithinRoot_stripsEmbeddedAbsoluteRoot(t *testing.T) {
	root := t.TempDir()
	makefile := filepath.Join(root, "Makefile")
	if err := os.WriteFile(makefile, []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Model sometimes passes the abs path with root duplicated as a relative string.
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	embedded := strings.TrimPrefix(rootAbs, string(filepath.Separator)) + string(filepath.Separator) + "Makefile"
	got, err := pathutil.ResolveRelWithinRoot(root, embedded)
	if err != nil {
		t.Fatalf("ResolveRelWithinRoot: %v", err)
	}
	want, _ := filepath.Abs(makefile)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	doubled := filepath.Join(rootAbs, embedded)
	got2, err := pathutil.ResolveRelWithinRoot(root, doubled)
	if err != nil {
		t.Fatalf("abs doubled: %v", err)
	}
	if got2 != want {
		t.Fatalf("abs doubled got %q want %q", got2, want)
	}
}
