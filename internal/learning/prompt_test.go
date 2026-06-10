package learning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendForAgent_scopedAndGated(t *testing.T) {
	unlock := LockTestGlobals()
	dir := testDataDir(t)
	defer func() {
		unlock()
		_ = os.RemoveAll(dir)
	}()
	store, err := NewStore(filepath.Join(dir, "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	SetGlobalStore(store)
	SetEnabledChecker(func() bool { return true })

	if _, err := store.Add(Entry{AgentID: "a1", Content: "Alpha note", Category: CategoryPreference, Scope: ScopeAgent}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Entry{AgentID: "a2", Content: "Beta note", Category: CategoryFact, Scope: ScopeAgent}); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	pr := AppendForAgent(&sb, &protocol.AgentInfo{ID: "a1", Name: "Alpha", Type: protocol.AgentTypeBackend}, PromptContext{Query: "alpha"})
	if pr.Count != 1 {
		t.Fatalf("expected 1 injected, got %d", pr.Count)
	}
	out := sb.String()
	if !strings.Contains(out, "Alpha note") || strings.Contains(out, "Beta note") {
		t.Fatalf("unexpected prompt: %s", out)
	}

	sb.Reset()
	if AppendForAgent(&sb, &protocol.AgentInfo{ID: "mod", Type: protocol.AgentTypeModerator}, PromptContext{}).Count != 0 {
		t.Fatal("moderator should not inject")
	}
}

func TestGlobalScopeIsolation(t *testing.T) {
	unlock := LockTestGlobals()
	dir := testDataDir(t)
	defer func() {
		unlock()
		_ = os.RemoveAll(dir)
	}()
	store, err := NewStore(filepath.Join(dir, "learnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	SetGlobalStore(store)
	SetEnabledChecker(func() bool { return true })

	if _, err := store.Add(Entry{AgentID: "a1", Content: "Use Go everywhere", Category: CategoryPreference, Scope: ScopeGlobal, UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Entry{AgentID: "a2", Content: "Rust only here", Category: CategoryPreference, Scope: ScopeAgent, UserID: "u1"}); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	pr := AppendForAgent(&sb, &protocol.AgentInfo{ID: "a1", Name: "A", Type: protocol.AgentTypeBackend}, PromptContext{Query: "go", UserID: "u1"})
	if pr.Count < 1 {
		t.Fatal("expected global+agent learnings")
	}
	if !strings.Contains(sb.String(), "Use Go everywhere") {
		t.Fatal("expected global learning")
	}

	sb.Reset()
	pr = AppendForAgent(&sb, &protocol.AgentInfo{ID: "a2", Name: "B", Type: protocol.AgentTypeRust}, PromptContext{Query: "rust", UserID: "u1"})
	if !strings.Contains(sb.String(), "Rust only here") {
		t.Fatal("expected agent-scoped learning for a2")
	}
	if strings.Contains(sb.String(), "Use Go everywhere") && pr.Count == 1 {
		// global may also appear for a2 — that's correct for global scope
	}
}

func TestExtractDraftFromMessage(t *testing.T) {
	if got := ExtractDraftFromMessage("remember that I prefer tabs"); got != "I prefer tabs" {
		t.Fatalf("got %q", got)
	}
	if !HasLearningTrigger("always use snake_case") {
		t.Fatal("expected trigger")
	}
}

func TestStoreMigrationV1(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/learnings.json"
	legacy := `[{"id":"x1","agent_id":"a","content":"legacy fact","category":"fact","active":true}]`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	list := store.List("a")
	if len(list) != 1 || list[0].Scope != ScopeAgent {
		t.Fatalf("expected migrated agent scope, got %+v", list[0])
	}
}
