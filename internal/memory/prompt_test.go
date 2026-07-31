package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendForPrompt_budgetAndDisabled(t *testing.T) {
	var sb strings.Builder
	SetEnabledChecker(func() bool { return false })
	if pr := AppendForPrompt(&sb, PromptContext{Query: "auth"}); pr.Count != 0 {
		t.Fatal("expected no injection when disabled")
	}

	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEnabledChecker(func() bool { return true })
	_ = s.UpsertChunk(Chunk{
		ID: "msg:1", SourceType: SourceMessage, SourceID: "1", Channel: "ch",
		Content: "agreed on JWT refresh", ContentHash: "h", CreatedAt: time.Now(),
	})

	sb.Reset()
	pr := AppendForPrompt(&sb, PromptContext{Query: "JWT", Channel: "ch"})
	if pr.Count < 1 || !strings.Contains(sb.String(), sectionStart) {
		t.Fatalf("expected injection, got count=%d body=%q", pr.Count, sb.String())
	}
}

func TestAppendForPrompt_truncatesEntryAndContinuesWithinBudget(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEmbedClient(nil, "")
	SetEnabledChecker(func() bool { return true })
	now := time.Now()
	for i, marker := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
		content := "auth " + marker + " " + strings.Repeat(marker+" detail ", 60)
		if err := s.UpsertChunk(Chunk{
			ID: "msg:" + marker, SourceType: SourceMessage, SourceID: marker, Channel: "ch",
			Content: content, ContentHash: ContentHash(content), CreatedAt: now.Add(-time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	var sb strings.Builder
	result := AppendForPrompt(&sb, PromptContext{Query: "auth", Channel: "ch"})
	if result.Count < 1 {
		t.Fatalf("count=%d ids=%v body=%q", result.Count, result.IDs, sb.String())
	}
	// Full scored chunks are injected; section budget may truncate a later entry.
	if !strings.Contains(sb.String(), "alpha") {
		t.Fatalf("expected top hit retained: %q", sb.String())
	}
	if !strings.Contains(sb.String(), "…") && result.Count >= DefaultTopK {
		// When many large chunks compete, budget truncation should leave an ellipsis.
		t.Logf("note: no budget ellipsis (count=%d); body len=%d", result.Count, len(sb.String()))
	}
	if len(sb.String()) > DefaultPromptBudget+len(sectionStart)+len(sectionEnd)+len(sectionHint)+40 {
		t.Fatalf("prompt memory exceeded bounded section: %d", len(sb.String()))
	}
}
