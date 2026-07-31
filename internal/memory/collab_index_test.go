package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
)

func TestBackfillWorkspaceCollabs_indexesFindings(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEmbedClient(nil, "")
	SetEnabledChecker(func() bool { return true })

	ws := t.TempDir()
	collabID := "c-findings-1"
	collabDir := filepath.Join(ws, "collabs", collabID)
	if err := os.MkdirAll(collabDir, 0o755); err != nil {
		t.Fatal(err)
	}
	findings := filepath.Join(collabDir, "findings.md")
	body := "# Findings\n\nDecision: use JWT auth middleware for hub DMs.\n"
	if err := os.WriteFile(findings, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := BackfillWorkspaceCollabs(ctx, ws); err != nil {
		t.Fatal(err)
	}

	has, err := s.HasSource(findings)
	if err != nil || !has {
		t.Fatalf("expected findings indexed, has=%v err=%v", has, err)
	}
	cands, err := s.ListCandidates("", collabID, 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ch := range cands {
		if ch.RelPath == "collabs/"+collabID+"/findings.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected findings rel_path in candidates, got %+v", cands)
	}
}

func TestBackfillCollabDirs_reviewsAndCollabs(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEmbedClient(nil, "")
	SetEnabledChecker(func() bool { return true })

	assets := t.TempDir()
	reviewFile := filepath.Join(assets, "reviews", "r1", "plan.md")
	if err := os.MkdirAll(filepath.Dir(reviewFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reviewFile, []byte("Plan: ship JWT auth."), 0o644); err != nil {
		t.Fatal(err)
	}
	collabFile := filepath.Join(assets, "collabs", "c1", "findings.md")
	if err := os.MkdirAll(filepath.Dir(collabFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(collabFile, []byte("Findings: JWT for DMs."), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := BackfillCollabDirs(ctx, assets); err != nil {
		t.Fatal(err)
	}
	for _, abs := range []string{reviewFile, collabFile} {
		has, err := s.HasSource(abs)
		if err != nil || !has {
			t.Fatalf("expected %s indexed, has=%v err=%v", abs, has, err)
		}
	}
}

func TestIndexCollabDirMarkdown_indexesFindingsBesidePlan(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "memory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	SetStore(s)
	SetEmbedClient(nil, "")
	SetEnabledChecker(func() bool { return true })

	collabDir := filepath.Join(t.TempDir(), "collabs", "c9")
	if err := os.MkdirAll(collabDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(collabDir, "plan.md")
	findings := filepath.Join(collabDir, "findings.md")
	_ = os.WriteFile(plan, []byte("Plan body"), 0o644)
	_ = os.WriteFile(findings, []byte("Findings body JWT"), 0o644)

	paths := &collaboration.ReviewAssetPaths{
		Directory: collabDir,
		Plan:      plan,
	}
	// Sync path used by IndexReviewAssetPaths (async IndexCollabFile wraps this).
	indexCollabDirMarkdown(paths.Directory, "c9", "ch-c9")
	time.Sleep(20 * time.Millisecond) // IndexCollabFile is async; prefer sync index:
	_ = indexCollabFile(context.Background(), findings, "collabs/c9/findings.md", "c9", "ch-c9")
	_ = indexCollabFile(context.Background(), plan, "collabs/c9/plan.md", "c9", "ch-c9")

	has, err := s.HasSource(findings)
	if err != nil || !has {
		t.Fatalf("expected findings indexed, has=%v err=%v", has, err)
	}
}
