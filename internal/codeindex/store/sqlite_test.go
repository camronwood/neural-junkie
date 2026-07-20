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
