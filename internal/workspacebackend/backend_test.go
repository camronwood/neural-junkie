package workspacebackend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBackendReadWrite(t *testing.T) {
	dir := t.TempDir()
	b := NewLocal(dir)
	ctx := context.Background()
	if err := b.WriteFile(ctx, "hello.txt", []byte("world")); err != nil {
		t.Fatal(err)
	}
	data, err := b.ReadFile(ctx, "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world" {
		t.Fatalf("got %q", data)
	}
	entries, err := b.ReadDir(ctx, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "hello.txt" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestLocalBackendPathContainment(t *testing.T) {
	dir := t.TempDir()
	b := NewLocal(dir)
	ctx := context.Background()
	_, err := b.ReadFile(ctx, "../outside.txt")
	if err == nil {
		t.Fatal("expected path escape to fail")
	}
	abs := filepath.Join(dir, "nested", "a.go")
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Stat(ctx, "nested/a.go"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalBackendStripsEmbeddedRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := NewLocal(dir)
	rootAbs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	embedded := strings.TrimPrefix(rootAbs, string(filepath.Separator)) + "/Makefile"
	data, err := b.ReadFile(context.Background(), embedded)
	if err != nil {
		t.Fatalf("ReadFile embedded root: %v", err)
	}
	if string(data) != "all:\n" {
		t.Fatalf("got %q", data)
	}
}
