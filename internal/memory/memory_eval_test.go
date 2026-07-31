package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/embed"
)

// TestLiveMemoryRetrievalEvaluation embeds corpus seeds via Ollama and checks
// hit@K / forbidden rates. Opt-in: NJ_RUN_LOCAL_MEMORY_EVAL=1.
func TestLiveMemoryRetrievalEvaluation(t *testing.T) {
	if os.Getenv("NJ_RUN_LOCAL_MEMORY_EVAL") != "1" {
		t.Skip("set NJ_RUN_LOCAL_MEMORY_EVAL=1 to run live memory retrieval eval")
	}
	endpoint := strings.TrimRight(os.Getenv("NJ_OLLAMA_ENDPOINT"), "/")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model := strings.TrimSpace(os.Getenv("NJ_MEMORY_EMBED_MODEL"))
	if model == "" {
		model = embed.DefaultModel
	}

	corpus := loadRetrievalCorpus(t)
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEnabledChecker(func() bool { return true })
	SetEmbedClient(embed.NewClient(endpoint, model), model)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// Probe embed once.
	if _, err := embedClient.Embed(ctx, "memory eval probe", true); err != nil {
		t.Fatalf("ollama embed unavailable (%s / %s): %v", endpoint, model, err)
	}

	minHit := 0.90
	maxForbidden := 0.05
	if v := os.Getenv("NJ_MEMORY_EVAL_MIN_HIT"); v != "" {
		if _, err := fmt.Sscanf(v, "%f", &minHit); err != nil {
			t.Fatalf("bad NJ_MEMORY_EVAL_MIN_HIT=%q", v)
		}
	}
	if v := os.Getenv("NJ_MEMORY_EVAL_MAX_FORBIDDEN"); v != "" {
		if _, err := fmt.Sscanf(v, "%f", &maxForbidden); err != nil {
			t.Fatalf("bad NJ_MEMORY_EVAL_MAX_FORBIDDEN=%q", v)
		}
	}

	type failRow struct {
		Name       string
		Missing    []string
		Forbidden  []string
		GotIDs     []string
		FailKind   string
	}
	var fails []failRow
	hitCases := 0
	forbiddenHits := 0
	now := time.Now()

	for _, c := range corpus.Cases {
		// Fresh store per case so seeds do not leak.
		caseDir := t.TempDir()
		cs, err := Open(filepath.Join(caseDir, "memory.db"))
		if err != nil {
			t.Fatal(err)
		}
		SetStore(cs)
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
			if err := upsertChunkWithEmbed(ctx, ch); err != nil {
				t.Fatalf("%s upsert: %v", c.Name, err)
			}
		}
		limit := c.Limit
		if limit <= 0 {
			limit = DefaultTopK
		}
		results, err := Search(ctx, PromptContext{
			Query:                c.Query,
			Channel:              c.Channel,
			GoalID:               c.GoalID,
			SourceTypes:          c.SourceTypes,
			ExcludeMessageIDs:    c.ExcludeMessageIDs,
			SupersededMessageIDs: c.SupersededMessageIDs,
		}, limit)
		_ = cs.Close()
		if err != nil {
			fails = append(fails, failRow{Name: c.Name, FailKind: "search_error", GotIDs: []string{err.Error()}})
			continue
		}
		got := map[string]bool{}
		ids := resultIDs(results)
		for _, id := range ids {
			got[id] = true
		}
		var missing, forbidden []string
		for _, id := range c.MustIncludeIDs {
			if !got[id] {
				missing = append(missing, id)
			}
		}
		for _, id := range c.MustExcludeIDs {
			if got[id] {
				forbidden = append(forbidden, id)
				forbiddenHits++
			}
		}
		if len(missing) == 0 && len(forbidden) == 0 {
			hitCases++
		} else {
			kind := "miss"
			if len(forbidden) > 0 {
				kind = "forbidden"
			}
			fails = append(fails, failRow{
				Name: c.Name, Missing: missing, Forbidden: forbidden, GotIDs: ids, FailKind: kind,
			})
			t.Logf("case miss name=%s missing=%v forbidden=%v got=%v", c.Name, missing, forbidden, ids)
		}
	}

	n := float64(len(corpus.Cases))
	hitRate := float64(hitCases) / n
	forbiddenRate := float64(forbiddenHits) / n
	t.Logf("memory live eval model=%s hit_rate=%.3f forbidden_hit_rate=%.3f n=%d fails=%d",
		model, hitRate, forbiddenRate, len(corpus.Cases), len(fails))

	if out := os.Getenv("NJ_MEMORY_EVAL_OUT"); out != "" {
		payload := map[string]any{
			"model":               model,
			"n":                   len(corpus.Cases),
			"hit_rate":            hitRate,
			"forbidden_hit_rate":  forbiddenRate,
			"forbidden_hit_count": forbiddenHits,
			"hit_cases":           hitCases,
			"fails":               fails,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		outPath := out
		if !filepath.IsAbs(outPath) {
			_, file, _, _ := runtime.Caller(0)
			outPath = filepath.Join(filepath.Dir(file), "..", "..", outPath)
		}
		_ = os.MkdirAll(filepath.Dir(outPath), 0o755)
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			t.Fatalf("write eval out: %v", err)
		}
		t.Logf("wrote %s", outPath)
	}

	if hitRate < minHit {
		t.Fatalf("memory hit_rate %.3f below %.3f (model=%s)", hitRate, minHit, model)
	}
	if forbiddenRate > maxForbidden {
		t.Fatalf("memory forbidden_hit_rate %.3f above %.3f (model=%s)", forbiddenRate, maxForbidden, model)
	}
}
