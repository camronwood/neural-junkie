package codeindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkFile(t *testing.T) {
	content := strings.Repeat("line\n", 250)
	chunks := chunkFile("pkg/foo.go", content)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if chunks[0].Path != "pkg/foo.go" || chunks[0].Start != 1 {
		t.Fatalf("unexpected first chunk: %+v", chunks[0])
	}
}

func TestKeywordSearchFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(path, []byte("func ComputeObscureWidget() int { return 42 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err := Search(t.Context(), dir, "helper", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected keyword fallback hit")
	}
	found := false
	for _, r := range results {
		if strings.Contains(r.Content, "ObscureWidget") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ObscureWidget in results: %+v", results)
	}
}

func TestRepoHashStable(t *testing.T) {
	dir := t.TempDir()
	h1 := RepoHash(dir)
	h2 := RepoHash(dir)
	if h1 != h2 || h1 == "" {
		t.Fatalf("hash unstable: %q %q", h1, h2)
	}
}
