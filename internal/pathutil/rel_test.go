package pathutil_test

import (
	"os"
	"path/filepath"
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
