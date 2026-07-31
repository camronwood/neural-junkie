package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type retrievalCorpusFile struct {
	SchemaVersion int                   `json:"schema_version"`
	Cases         []retrievalCorpusCase `json:"cases"`
}

type retrievalCorpusCase struct {
	Name                  string             `json:"name"`
	Query                 string             `json:"query"`
	Channel               string             `json:"channel"`
	GoalID                string             `json:"goal_id"`
	SourceTypes           []SourceType       `json:"source_types"`
	ExcludeMessageIDs     []string           `json:"exclude_message_ids"`
	SupersededMessageIDs  []string           `json:"superseded_message_ids"`
	Limit                 int                `json:"limit"`
	SeedChunks            []retrievalSeed    `json:"seed_chunks"`
	MustIncludeIDs        []string           `json:"must_include_ids"`
	MustExcludeIDs        []string           `json:"must_exclude_ids"`
}

type retrievalSeed struct {
	ID           string     `json:"id"`
	SourceType   SourceType `json:"source_type"`
	SourceID     string     `json:"source_id"`
	Channel      string     `json:"channel"`
	GoalID       string     `json:"goal_id"`
	IsCorrection bool       `json:"is_correction"`
	RelPath      string     `json:"rel_path"`
	Content      string     `json:"content"`
}

func loadRetrievalCorpus(t *testing.T) retrievalCorpusFile {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "..", "..", "scenarios", "memory", "retrieval-corpus.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus retrievalCorpusFile
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.SchemaVersion != 1 || len(corpus.Cases) == 0 {
		t.Fatalf("invalid retrieval corpus: version=%d n=%d", corpus.SchemaVersion, len(corpus.Cases))
	}
	return corpus
}

// TestRetrievalAgainstCorpus is the CI gate for conversation-memory ranking.
// Deterministic (no Ollama): lexical + recency + collab boosts only.
func TestRetrievalAgainstCorpus(t *testing.T) {
	corpus := loadRetrievalCorpus(t)
	now := time.Now()
	for _, c := range corpus.Cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(filepath.Join(dir, "memory.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			SetStore(s)
			SetEnabledChecker(func() bool { return true })
			SetEmbedClient(nil, "")

			for i, seed := range c.SeedChunks {
				ch := Chunk{
					ID:           seed.ID,
					SourceType:   seed.SourceType,
					SourceID:     seed.SourceID,
					Channel:      seed.Channel,
					GoalID:       seed.GoalID,
					IsCorrection: seed.IsCorrection,
					RelPath:      seed.RelPath,
					Content:      seed.Content,
					ContentHash:  seed.ID,
					CreatedAt:    now.Add(time.Duration(i) * time.Second),
				}
				if ch.SourceType == "" {
					ch.SourceType = SourceMessage
				}
				if err := s.UpsertChunk(ch); err != nil {
					t.Fatal(err)
				}
			}

			limit := c.Limit
			if limit <= 0 {
				limit = DefaultTopK
			}
			results, err := Search(context.Background(), PromptContext{
				Query:                c.Query,
				Channel:              c.Channel,
				GoalID:               c.GoalID,
				SourceTypes:          c.SourceTypes,
				ExcludeMessageIDs:    c.ExcludeMessageIDs,
				SupersededMessageIDs: c.SupersededMessageIDs,
			}, limit)
			if err != nil {
				t.Fatal(err)
			}

			got := map[string]bool{}
			for _, r := range results {
				got[r.Chunk.ID] = true
			}
			for _, id := range c.MustIncludeIDs {
				if !got[id] {
					t.Errorf("missing must_include %q in results=%v", id, resultIDs(results))
				}
			}
			for _, id := range c.MustExcludeIDs {
				if got[id] {
					t.Errorf("forbidden must_exclude %q in results=%v", id, resultIDs(results))
				}
			}
		})
	}
}

func resultIDs(results []SearchResult) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.Chunk.ID)
	}
	return out
}

func TestSearch_multiChunkPerSource(t *testing.T) {
	s := openSearchTestStore(t)
	now := time.Now()
	for _, ch := range []Chunk{
		{ID: "a0", SourceType: SourceCollabArtifact, SourceID: "findings", Channel: "ch", RelPath: "collabs/c/findings.md", Content: "alpha decision JWT auth middleware hub", ContentHash: "a0", CreatedAt: now},
		{ID: "a1", SourceType: SourceCollabArtifact, SourceID: "findings", Channel: "ch", RelPath: "collabs/c/findings.md", Content: "beta follow-up OAuth marketplace only", ContentHash: "a1", CreatedAt: now},
		{ID: "a2", SourceType: SourceCollabArtifact, SourceID: "findings", Channel: "ch", RelPath: "collabs/c/findings.md", Content: "gamma note about token refresh cadence", ContentHash: "a2", CreatedAt: now},
		{ID: "a3", SourceType: SourceCollabArtifact, SourceID: "findings", Channel: "ch", RelPath: "collabs/c/findings.md", Content: "delta leftover chunk should not all fit", ContentHash: "a3", CreatedAt: now},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{
		Query: "JWT OAuth auth middleware token", Channel: "ch",
		SourceTypes: []SourceType{SourceCollabArtifact},
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("want at least 2 chunks from findings, got %+v", results)
	}
	if len(results) > MaxChunksPerSource {
		t.Fatalf("want <= %d chunks per source, got %d: %+v", MaxChunksPerSource, len(results), results)
	}
	for _, r := range results {
		if r.Chunk.SourceID != "findings" {
			t.Fatalf("unexpected source %+v", r)
		}
	}
}

func TestSearch_findingsBoostBeatsChatParaphrase(t *testing.T) {
	s := openSearchTestStore(t)
	now := time.Now()
	for _, ch := range []Chunk{
		{ID: "findings", SourceType: SourceCollabArtifact, SourceID: "findings", Channel: "ch", RelPath: "collabs/c/findings.md", Content: "Decision: pin terraform aws provider in versions.tf", ContentHash: "f", CreatedAt: now},
		{ID: "chat", SourceType: SourceMessage, SourceID: "chat", Channel: "ch", Content: "terraform aws provider pin in versions.tf sounds right", ContentHash: "c", CreatedAt: now},
	} {
		if err := s.UpsertChunk(ch); err != nil {
			t.Fatal(err)
		}
	}
	results, err := Search(context.Background(), PromptContext{
		Query: "terraform aws provider versions.tf decision", Channel: "ch",
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].Chunk.ID != "findings" {
		t.Fatalf("findings should rank first, got %+v", results)
	}
}
