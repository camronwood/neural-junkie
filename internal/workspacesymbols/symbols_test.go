package workspacesymbols

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchGoFunction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\n\nfunc Hello() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	syms, err := Search(context.Background(), dir, "hello", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Name != "Hello" {
		t.Fatalf("got %+v", syms)
	}
}
