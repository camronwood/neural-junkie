package learning

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddListForget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "learnings.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}

	e, err := store.Add(Entry{
		AgentID:   "agent-a",
		AgentName: "Alpha",
		Content:   "Prefer tabs",
		Category:  CategoryPreference,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.ID == "" {
		t.Fatal("expected id")
	}

	list := store.List("agent-a")
	if len(list) != 1 || list[0].Content != "Prefer tabs" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if store.CountForAgent("agent-a") != 1 {
		t.Fatal("expected count 1")
	}
	if len(store.List("agent-b")) != 0 {
		t.Fatal("expected no learnings for agent-b")
	}

	if err := store.Forget(e.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.List("agent-a")) != 0 {
		t.Fatal("expected forget to soft-delete")
	}
}

func TestStorePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "learnings.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Entry{AgentID: "x", Content: "fact one", Category: CategoryFact}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.List("x")) != 1 {
		t.Fatal("expected persisted entry")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestStoreValidation(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Entry{Content: "no agent"}); err == nil {
		t.Fatal("expected agent_id required")
	}
	if _, err := store.Add(Entry{AgentID: "a", Content: "   "}); err == nil {
		t.Fatal("expected content required")
	}
}

func TestStoreScopeAndUpdate(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := store.Add(Entry{AgentID: "a1", Content: "global note", Category: CategoryPreference, Scope: ScopeGlobal, UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if g.Scope != ScopeGlobal {
		t.Fatalf("expected global scope, got %s", g.Scope)
	}
	global := store.ListFiltered(Filter{UserID: "u1", Scope: ScopeGlobal})
	if len(global) != 1 {
		t.Fatalf("expected 1 global, got %d", len(global))
	}
	updated, err := store.Update(g.ID, UpdatePatch{Content: strPtr("updated global")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "updated global" || updated.UpdatedAt.IsZero() {
		t.Fatalf("unexpected update: %+v", updated)
	}
}

func strPtr(s string) *string { return &s }
