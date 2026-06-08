package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_upsertAndDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ch := Chunk{
		ID:         "msg:test:0",
		SourceType: SourceMessage,
		SourceID:   "test",
		Channel:    "dm-u-a",
		Content:    "JWT refresh rotation agreed",
		ContentHash: ContentHash("JWT refresh rotation agreed"),
		CreatedAt:  time.Now(),
	}
	if err := s.UpsertChunk(ch); err != nil {
		t.Fatal(err)
	}
	has, err := s.HasSource("test")
	if err != nil || !has {
		t.Fatalf("has=%v err=%v", has, err)
	}
	cands, err := s.ListCandidates("dm-u-a", "", 10)
	if err != nil || len(cands) != 1 {
		t.Fatalf("candidates=%d err=%v", len(cands), err)
	}
	if err := s.DeleteByChannel("dm-u-a"); err != nil {
		t.Fatal(err)
	}
	cands, _ = s.ListCandidates("dm-u-a", "", 10)
	if len(cands) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(cands))
	}
}
