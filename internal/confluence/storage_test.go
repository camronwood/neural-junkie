package confluence

import (
	"path/filepath"
	"testing"
	"time"
)

func testStorage(t *testing.T) *Storage {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	s, err := NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStorageSaveLoadDeleteIndex(t *testing.T) {
	s := testStorage(t)
	idx := NewConfluenceIndex("My Space", "My Space")
	idx.AddPage(&Page{
		ID: "p1", Title: "Hello", Content: "world", LastUpdated: time.Now(),
	})
	if err := s.SaveIndex(idx); err != nil {
		t.Fatal(err)
	}
	if !s.IndexExists("My Space") {
		t.Fatal("index should exist")
	}
	loaded, err := s.LoadIndex("My Space")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PageCount != 1 || loaded.Pages["p1"].Title != "Hello" {
		t.Fatalf("loaded: %+v", loaded)
	}
	size, err := s.GetIndexSize("My Space")
	if err != nil || size <= 0 {
		t.Fatalf("size: %d %v", size, err)
	}
	keys, err := s.ListIndexes()
	if err != nil || len(keys) != 1 {
		t.Fatalf("list: %v %v", keys, err)
	}
	if err := s.DeleteIndex("My Space"); err != nil {
		t.Fatal(err)
	}
	if s.IndexExists("My Space") {
		t.Fatal("index should be deleted")
	}
}

func TestStorageMetadataRoundTrip(t *testing.T) {
	s := testStorage(t)
	meta := map[string]interface{}{"pages": 42, "name": "docs"}
	if err := s.SaveMetadata("DOC", meta); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadMetadata("DOC")
	if err != nil {
		t.Fatal(err)
	}
	if loaded["pages"] != float64(42) {
		t.Fatalf("metadata: %+v", loaded)
	}
}

func TestStorageGetStorageDirSanitizes(t *testing.T) {
	s := testStorage(t)
	dir := s.GetStorageDir("MY/SPACE Key")
	if filepath.Base(dir) != "my-space-key" {
		t.Fatalf("sanitized dir: %s", dir)
	}
}

func TestStorageLoadMissingIndex(t *testing.T) {
	s := testStorage(t)
	_, err := s.LoadIndex("missing")
	if err == nil {
		t.Fatal("expected error for missing index")
	}
}
