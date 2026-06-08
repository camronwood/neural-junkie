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
