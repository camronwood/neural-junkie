package memory

import (
	"database/sql"
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
		ID:          "msg:test:0",
		SourceType:  SourceMessage,
		SourceID:    "test",
		Channel:     "dm-u-a",
		Content:     "JWT refresh rotation agreed",
		ContentHash: ContentHash("JWT refresh rotation agreed"),
		CreatedAt:   time.Now(),
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

func TestOpen_migratesLegacySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE memory_chunks (
 id TEXT PRIMARY KEY, source_type TEXT NOT NULL, source_id TEXT NOT NULL,
 channel TEXT NOT NULL DEFAULT '', thread_id TEXT NOT NULL DEFAULT '',
 collaboration_id TEXT NOT NULL DEFAULT '', rel_path TEXT NOT NULL DEFAULT '',
 sender_name TEXT NOT NULL DEFAULT '', content TEXT NOT NULL, content_hash TEXT NOT NULL,
 embedding_model TEXT NOT NULL DEFAULT '', vector_json TEXT NOT NULL DEFAULT '',
 created_at INTEGER NOT NULL
)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ch := Chunk{
		ID: "corrected", SourceType: SourceMessage, SourceID: "corrected", Channel: "ch",
		GoalID: "goal-1", IsCorrection: true, Content: "use rust", ContentHash: "h",
	}
	if err := s.UpsertChunk(ch); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListCandidates("ch", "", 1)
	if err != nil || len(got) != 1 || got[0].GoalID != "goal-1" || !got[0].IsCorrection {
		t.Fatalf("legacy migration result=%+v err=%v", got, err)
	}
}
