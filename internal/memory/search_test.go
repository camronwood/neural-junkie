package memory

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSearch_channelScopeAndExclude(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEnabledChecker(func() bool { return true })

	for _, ch := range []Chunk{
		{ID: "msg:old", SourceType: SourceMessage, SourceID: "old", Channel: "ch1", Content: "auth middleware JWT", ContentHash: "h1", CreatedAt: time.Now()},
		{ID: "msg:new", SourceType: SourceMessage, SourceID: "new", Channel: "ch1", Content: "thanks", ContentHash: "h2", CreatedAt: time.Now()},
		{ID: "msg:other", SourceType: SourceMessage, SourceID: "x", Channel: "ch2", Content: "auth middleware JWT", ContentHash: "h3", CreatedAt: time.Now()},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}

	results, err := Search(context.Background(), PromptContext{
		Query:             "auth middleware",
		Channel:           "ch1",
		ExcludeMessageIDs: []string{"new"},
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.SourceID != "old" {
		t.Fatalf("results=%+v", results)
	}
}
