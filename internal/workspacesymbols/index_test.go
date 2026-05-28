package workspacesymbols

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchIndexedRust(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(path, []byte("pub fn hello_world() {}\npub struct Widget {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	syms, err := SearchIndexed(context.Background(), dir, "hello", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range syms {
		if s.Name == "hello_world" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hello_world in %#v", syms)
	}
}
