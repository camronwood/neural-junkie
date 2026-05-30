package learning

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendForAgent_scopedAndGated(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/learnings.json")
	if err != nil {
		t.Fatal(err)
	}
	SetGlobalStore(store)
	SetEnabledChecker(func() bool { return true })
	t.Cleanup(func() {
		SetGlobalStore(nil)
		SetEnabledChecker(nil)
	})

	if _, err := store.Add(Entry{AgentID: "a1", Content: "Alpha note", Category: CategoryPreference}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(Entry{AgentID: "a2", Content: "Beta note", Category: CategoryFact}); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	n := AppendForAgent(&sb, &protocol.AgentInfo{ID: "a1", Name: "Alpha", Type: protocol.AgentTypeBackend})
	if n != 1 {
		t.Fatalf("expected 1 injected, got %d", n)
	}
	out := sb.String()
	if !strings.Contains(out, "Alpha note") || strings.Contains(out, "Beta note") {
		t.Fatalf("unexpected prompt: %s", out)
	}

	sb.Reset()
	if AppendForAgent(&sb, &protocol.AgentInfo{ID: "mod", Type: protocol.AgentTypeModerator}) != 0 {
		t.Fatal("moderator should not inject")
	}
	if sb.Len() != 0 {
		t.Fatal("moderator prompt should stay empty")
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
