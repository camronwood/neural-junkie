package store

import (
	"path/filepath"
	"testing"
)

func TestSQLitePutGetDeleteMissing(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "idx"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	vec := []float64{0.1, 0.2, 0.3}
	if err := s.Put("chunk-a", vec); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get("chunk-a")
	if !ok {
		t.Fatal("expected chunk-a")
	}
	if len(got.Vector) != 3 || got.Vector[1] != 0.2 {
		t.Fatalf("vector = %#v", got.Vector)
	}
	if err := s.Put("chunk-b", []float64{1}); err != nil {
		t.Fatal(err)
	}
	n, err := s.Count()
	if err != nil || n != 2 {
		t.Fatalf("count = %d err=%v", n, err)
	}
	if err := s.DeleteMissing(map[string]struct{}{"chunk-a": {}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("chunk-b"); ok {
		t.Fatal("chunk-b should be deleted")
	}
	if _, ok := s.Get("chunk-a"); !ok {
		t.Fatal("chunk-a should remain")
	}
}

func TestReplaceAllChunksAndLexicalCandidates(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "idx")
	s, err := Open(idx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.ReplaceAllChunks([]ChunkRecord{
		{ID: "a:1-2", Path: "pkg/a.go", Start: 1, End: 2, Content: "func ComputeWidget() {}"},
		{ID: "b:1-2", Path: "pkg/b.go", Start: 1, End: 2, Content: "func OtherThing() {}"},
	}); err != nil {
		t.Fatal(err)
	}
	n, err := s.ChunkCount()
	if err != nil || n != 2 {
		t.Fatalf("chunk count = %d err=%v", n, err)
	}
	hits := s.LexicalCandidates("ComputeWidget", 10)
	if len(hits) == 0 {
		t.Fatal("expected lexical hit")
	}
	found := false
	for _, h := range hits {
		if h.ID == "a:1-2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a:1-2 in %#v", hits)
	}
	got := s.GetChunks([]string{"a:1-2"})
	if len(got) != 1 || got[0].Path != "pkg/a.go" {
		t.Fatalf("GetChunks = %#v", got)
	}
	if !Exists(idx) {
		t.Fatal("Exists should be true")
	}
}
