package agent

import (
	"context"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp_export"
	"github.com/camronwood/neural-junkie/internal/repo"
	"github.com/camronwood/neural-junkie/internal/testutil"
)

func sampleShareExport() *mcp_export.AgentExport {
	return &mcp_export.AgentExport{
		Version: "1.0",
		Agent: mcp_export.AgentMetadata{
			Name:       "Widget Expert",
			Type:       "repo",
			Expertise:  []string{"Go", "Widgets"},
			Repository: "/some/path/that/does/not/exist/on/this/machine",
		},
		Resources: []mcp_export.MCPResource{
			mcp_export.CreateRepoResource("repo://architecture", "Architecture Overview", "text/markdown", "# Widget Service\n\nA widget service."),
			mcp_export.CreateRepoResource("repo://patterns", "Code Patterns", "text/plain", "REST API\nEvent-driven"),
			mcp_export.CreateRepoResource("repo://dependencies", "Dependencies", "text/plain", "=== GO Dependencies ===\n- github.com/gorilla/mux\n"),
			mcp_export.CreateRepoResource("repo://files/README.md", "Key File: README.md", "text/markdown", "# Widget\nA widget service."),
			mcp_export.CreateRepoResource("repo://files/main.go", "Source File: main.go", "text/x-go", "package main\n\nfunc main() {}\n"),
		},
		SystemPrompt: "You are Widget Expert, a repository expert agent with deep knowledge of the widget-service codebase.",
		CustomRulesMarkdown: "Always use small PRs.",
		Learnings: []mcp_export.LearningEntry{
			{Content: "Prefers Rust for new services.", Category: "preference"},
		},
	}
}

func newHydratableRepoAgent(t *testing.T) *RepoAgent {
	t.Helper()
	testutil.IsolateNeuralJunkieHome(t)
	hub := &captureHub{}
	ra, err := NewRepoAgentWithOptions("Widget Expert", "(hydrated)", ai.NewMockProvider(), hub, RepoAgentOptions{SkipPathCheck: true})
	if err != nil {
		t.Fatalf("NewRepoAgentWithOptions: %v", err)
	}
	return ra
}

func TestHydrateFromExport_PopulatesIndexFromResources(t *testing.T) {
	ra := newHydratableRepoAgent(t)
	export := sampleShareExport()

	if err := ra.HydrateFromExport(export, ""); err != nil {
		t.Fatalf("HydrateFromExport: %v", err)
	}

	ra.mu.RLock()
	idx := ra.index
	ra.mu.RUnlock()
	if idx == nil {
		t.Fatal("expected index to be populated")
	}
	if idx.ArchitectureDoc == "" {
		t.Fatal("expected architecture doc to be hydrated")
	}
	if got, ok := idx.KeyFiles["README.md"]; !ok || got == "" {
		t.Fatalf("expected README.md to be hydrated as a key file, got %q (ok=%v)", got, ok)
	}
	src, ok := idx.SourceFiles["main.go"]
	if !ok {
		t.Fatal("expected main.go to be hydrated as a source file")
	}
	content, err := repo.DecompressContent(src.Content)
	if err != nil {
		t.Fatalf("failed to decompress hydrated source file: %v", err)
	}
	if content != "package main\n\nfunc main() {}\n" {
		t.Fatalf("unexpected hydrated source content: %q", content)
	}
	if src.Language != "Go" {
		t.Fatalf("expected language 'Go', got %q", src.Language)
	}
	if len(idx.CodePatterns) != 2 {
		t.Fatalf("expected 2 code patterns, got %d", len(idx.CodePatterns))
	}
	if len(idx.Dependencies["exported"]) != 1 {
		t.Fatalf("expected exported dependency bucket, got %v", idx.Dependencies)
	}
	if idx.FileCount != 2 {
		t.Fatalf("expected FileCount 2 (1 key file + 1 source file), got %d", idx.FileCount)
	}
	if !export.HydratedFromResources {
		t.Fatal("expected export.HydratedFromResources to be set true after hydration")
	}
	if len(ra.Info.Expertise) != 2 {
		t.Fatalf("expected expertise to be copied from export, got %v", ra.Info.Expertise)
	}
}

func TestHydrateFromExport_RepoPathOverride(t *testing.T) {
	ra := newHydratableRepoAgent(t)
	export := sampleShareExport()

	if err := ra.HydrateFromExport(export, "/tmp/remapped-widget-repo"); err != nil {
		t.Fatalf("HydrateFromExport: %v", err)
	}
	if ra.repoPath != "/tmp/remapped-widget-repo" {
		t.Fatalf("expected repoPath override to take effect, got %q", ra.repoPath)
	}
}

func TestHydrateFromExport_NilExportErrors(t *testing.T) {
	ra := newHydratableRepoAgent(t)
	if err := ra.HydrateFromExport(nil, ""); err == nil {
		t.Fatal("expected error hydrating from nil export")
	}
}

func TestStartHydrated_JoinsChannelWithoutDiskIndexing(t *testing.T) {
	ra := newHydratableRepoAgent(t)
	export := sampleShareExport()
	if err := ra.HydrateFromExport(export, ""); err != nil {
		t.Fatalf("HydrateFromExport: %v", err)
	}

	if err := ra.StartHydrated(context.Background(), "general"); err != nil {
		t.Fatalf("StartHydrated: %v", err)
	}

	ra.mu.RLock()
	idx := ra.index
	ra.mu.RUnlock()
	if idx == nil {
		t.Fatal("index should remain set after StartHydrated (no disk re-index)")
	}
	if ra.Context.CurrentChannel != "general" {
		t.Fatalf("expected current channel to be set, got %q", ra.Context.CurrentChannel)
	}
}
