package codeindex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/codeindex/store"
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

func TestBuildIndexSkipsJunkAndUsesSQLite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repoDir := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repoDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("pkg/widget.go", "package pkg\n\nfunc ComputeWidget() int { return 1 }\n")
	mustWrite("docs/media/demo.mp4", "not-really-video-but-extension-matters")
	mustWrite("ui/package-lock.json", `{"lockfileVersion": 2}`)
	mustWrite("bin/libfoo.so", string([]byte{0x7f, 'E', 'L', 'F', 0, 1}))
	mustWrite("assets/readme.pdf", "%PDF-1.4 binary-ish")

	if err := BuildIndex(t.Context(), repoDir); err != nil {
		t.Fatal(err)
	}

	meta, err := Status(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.Ready || meta.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.ChunkCount < 1 {
		t.Fatal("expected at least one chunk")
	}

	dir, err := indexDir(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "chunks.json")); !os.IsNotExist(err) {
		t.Fatalf("chunks.json should be absent, err=%v", err)
	}
	if !store.Exists(dir) {
		t.Fatal("expected index.db")
	}

	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hits := s.LexicalCandidates("ComputeWidget", 10)
	if len(hits) == 0 {
		t.Fatal("expected indexed widget chunk")
	}
	for _, h := range hits {
		if strings.Contains(h.Path, "package-lock") || strings.HasSuffix(h.Path, ".mp4") ||
			strings.HasSuffix(h.Path, ".so") || strings.HasSuffix(h.Path, ".pdf") {
			t.Fatalf("junk path indexed: %s", h.Path)
		}
	}

	results, err := Search(t.Context(), repoDir, "ComputeWidget", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("Search should hit SQLite index without chunks.json")
	}
}

func TestStatusLegacySchemaNotReady(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := t.TempDir()

	dir, err := indexDir(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := IndexMeta{
		RepoPath:      repoDir,
		RepoHash:      RepoHash(repoDir),
		ChunkCount:    99,
		SchemaVersion: 1,
		Ready:         true,
	}
	raw, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	// Legacy chunks.json alone must not count as ready.
	if err := os.WriteFile(filepath.Join(dir, "chunks.json"), []byte(`[]`), 0o644); err != nil {
		t.Fatal(err)
	}

	meta, err := Status(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Ready {
		t.Fatalf("legacy schema should not be ready: %+v", meta)
	}
}
