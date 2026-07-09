package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTaskContextOpenFiles_readsRepoSources(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"README.md", "core/sample/main.go"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content of "+rel), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := taskContextOpenFiles(root, []string{"README.md", "core/sample/main.go"})
	if len(files) != 2 {
		t.Fatalf("expected 2 open files, got %d", len(files))
	}
}
