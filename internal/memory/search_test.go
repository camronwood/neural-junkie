package memory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/embed"
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

func TestSearch_sourceTypeFilter(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEnabledChecker(func() bool { return true })

	for _, ch := range []Chunk{
		{ID: "msg:1", SourceType: SourceMessage, SourceID: "1", Channel: "ch1", Content: "auth decision JWT", ContentHash: "h1", CreatedAt: time.Now()},
		{ID: "art:1", SourceType: SourceCollabArtifact, SourceID: "plan", Channel: "ch1", RelPath: "collabs/plan.md", Content: "auth decision OAuth", ContentHash: "h2", CreatedAt: time.Now()},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}

	collabOnly, err := Search(context.Background(), PromptContext{
		Query: "auth decision", Channel: "ch1", SourceTypes: []SourceType{SourceCollabArtifact},
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(collabOnly) != 1 || collabOnly[0].Chunk.SourceType != SourceCollabArtifact {
		t.Fatalf("collabOnly=%+v", collabOnly)
	}

	msgOnly, err := Search(context.Background(), PromptContext{
		Query: "auth decision", Channel: "ch1", SourceTypes: []SourceType{SourceMessage},
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgOnly) != 1 || msgOnly[0].Chunk.SourceType != SourceMessage {
		t.Fatalf("msgOnly=%+v", msgOnly)
	}
}

func TestSearch_threadFirstRecencyAndThreshold(t *testing.T) {
	s := openSearchTestStore(t)
	now := time.Now()
	for _, ch := range []Chunk{
		{ID: "thread", SourceType: SourceMessage, SourceID: "thread", Channel: "ch", ThreadID: "t1", Content: "deploy lambda canary", ContentHash: "1", CreatedAt: now.Add(-60 * 24 * time.Hour)},
		{ID: "channel", SourceType: SourceMessage, SourceID: "channel", Channel: "ch", Content: "deploy lambda canary rollout recent", ContentHash: "2", CreatedAt: now},
		{ID: "noise", SourceType: SourceMessage, SourceID: "noise", Channel: "ch", Content: "unrelated lunch notes", ContentHash: "3", CreatedAt: now},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{Query: "deploy lambda", Channel: "ch", ThreadID: "t1"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Chunk.ID != "thread" {
		t.Fatalf("thread-first results=%+v", results)
	}
	irrelevant, err := Search(context.Background(), PromptContext{Query: "postgres migration", Channel: "ch"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(irrelevant) != 0 {
		t.Fatalf("relevance floor admitted noise: %+v", irrelevant)
	}
}

func TestSearch_filtersSupersededAndForeignCorrections(t *testing.T) {
	s := openSearchTestStore(t)
	now := time.Now()
	for _, ch := range []Chunk{
		{ID: "old", SourceType: SourceMessage, SourceID: "old", Channel: "ch", GoalID: "goal-1", Content: "use terraform provider aws", ContentHash: "1", CreatedAt: now},
		{ID: "current-correction", SourceType: SourceMessage, SourceID: "current-correction", Channel: "ch", GoalID: "goal-1", IsCorrection: true, Content: "use configured aws provider", ContentHash: "2", CreatedAt: now},
		{ID: "foreign-correction", SourceType: SourceMessage, SourceID: "foreign-correction", Channel: "ch", GoalID: "goal-2", IsCorrection: true, Content: "use legacy aws provider", ContentHash: "3", CreatedAt: now},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{
		Query: "aws provider", Channel: "ch", GoalID: "goal-1", SupersededMessageIDs: []string{"old"},
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.ID != "current-correction" {
		t.Fatalf("filtered results=%+v", results)
	}
}

func TestSearch_FTSFindsRelevantChunkOutsideRecencyPrefilter(t *testing.T) {
	s := openSearchTestStore(t)
	now := time.Now()
	if err := s.UpsertChunk(Chunk{
		ID: "old-relevant", SourceType: SourceMessage, SourceID: "old-relevant", Channel: "ch",
		Content: "quasarneedle deployment decision", ContentHash: "target", CreatedAt: now.Add(-365 * 24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < DefaultSearchPrefilter+5; i++ {
		ch := Chunk{
			ID: fmt.Sprintf("recent-%03d", i), SourceType: SourceMessage, SourceID: fmt.Sprintf("recent-%03d", i),
			Channel: "ch", Content: "routine status update", ContentHash: fmt.Sprintf("h-%03d", i),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{Query: "quasarneedle", Channel: "ch"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.ID != "old-relevant" {
		t.Fatalf("FTS retrieval results=%+v", results)
	}
}

func TestSearch_hybridKeepsLexicalAndEmbeddingMatches(t *testing.T) {
	s := openSearchTestStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embedding":[1,0]}`))
	}))
	defer server.Close()
	SetEmbedClient(embed.NewClient(server.URL, "test"), "test")
	t.Cleanup(func() { SetEmbedClient(nil, "") })
	now := time.Now()
	for _, ch := range []Chunk{
		{ID: "lexical", SourceType: SourceMessage, SourceID: "lexical", Channel: "ch", Content: "quasarneedle exact term", ContentHash: "1", CreatedAt: now},
		{ID: "semantic", SourceType: SourceMessage, SourceID: "semantic", Channel: "ch", Content: "deployment architecture decision", ContentHash: "2", Vector: []float64{1, 0}, CreatedAt: now},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{Query: "quasarneedle", Channel: "ch"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("hybrid results=%+v", results)
	}
	got := map[string]bool{results[0].Chunk.ID: true, results[1].Chunk.ID: true}
	if !got["lexical"] || !got["semantic"] {
		t.Fatalf("hybrid signals not retained: %+v", results)
	}
}

func openSearchTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	SetStore(s)
	SetEmbedClient(nil, "")
	SetEnabledChecker(func() bool { return true })
	return s
}
