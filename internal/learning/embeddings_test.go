package learning

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/embed"
)

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	if embed.CosineSimilarity(a, b) < 0.99 {
		t.Fatal("identical vectors should score ~1")
	}
	c := []float64{0, 1, 0}
	if embed.CosineSimilarity(a, c) > 0.01 {
		t.Fatal("orthogonal vectors should score ~0")
	}
}

func TestKeywordScoreFallback(t *testing.T) {
	score := embed.KeywordScore("please use tabs in go", "Always use tabs for indentation in Go files")
	if score <= 0 {
		t.Fatal("expected positive keyword overlap")
	}
}

func TestSelectForPrompt_keywordFallback(t *testing.T) {
	unlock := LockTestGlobals()
	dir := testDataDir(t)
	t.Cleanup(func() {
		unlock()
		_ = os.RemoveAll(dir)
	})
	store, err := NewStore(filepath.Join(dir, "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	emb, err := NewEmbedStore(filepath.Join(dir, "emb.json"))
	if err != nil {
		t.Fatal(err)
	}
	SetGlobalStore(store)
	SetEmbedStore(emb)
	SetEnabledChecker(func() bool { return true })

	_, _ = store.Add(Entry{AgentID: "a1", Content: "Prefer PostgreSQL over MySQL", Category: CategoryPreference, Scope: ScopeAgent})
	_, _ = store.Add(Entry{AgentID: "a1", Content: "Use dark mode UI", Category: CategoryPreference, Scope: ScopeAgent})

	_, agent, _, ids := SelectForPrompt(context.Background(), PromptContext{
		Query:  "database postgres",
		UserID: "",
	}, "a1")
	if len(agent) == 0 && len(ids) == 0 {
		t.Fatal("expected at least one match via keyword fallback")
	}
	foundPG := false
	for _, e := range agent {
		if e.Content == "Prefer PostgreSQL over MySQL" {
			foundPG = true
		}
	}
	if !foundPG {
		t.Fatalf("expected postgres-related learning, got %+v", agent)
	}
}

func TestQueryPreview_scopes(t *testing.T) {
	unlock := LockTestGlobals()
	dir := testDataDir(t)
	t.Cleanup(func() {
		unlock()
		_ = os.RemoveAll(dir)
	})
	store, err := NewStore(filepath.Join(dir, "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	SetGlobalStore(store)
	SetEnabledChecker(func() bool { return true })

	_, _ = store.Add(Entry{AgentID: "a1", Content: "Global pref", Category: CategoryPreference, Scope: ScopeGlobal})
	_, _ = store.Add(Entry{AgentID: "a1", Content: "Agent pref", Category: CategoryPreference, Scope: ScopeAgent})

	g := QueryPreview(context.Background(), PromptContext{Query: "pref"}, "a1", ScopeGlobal)
	if len(g) != 1 || g[0].Content != "Global pref" {
		t.Fatalf("global preview: %+v", g)
	}
	a := QueryPreview(context.Background(), PromptContext{Query: "pref"}, "a1", ScopeAgent)
	if len(a) != 1 || a[0].Content != "Agent pref" {
		t.Fatalf("agent preview: %+v", a)
	}
}
