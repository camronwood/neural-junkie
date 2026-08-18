package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/codeintel"
	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/routing"
)

func TestMergeCodebaseAttachments_noMention(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{Content: "hello"}
	MergeCodebaseAttachments(msg)
	if msg.Metadata != nil {
		t.Fatalf("expected no metadata")
	}
}

func TestWorkspacePathFromMetadata(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/repo",
			},
		},
	}
	if got := workspacePathFromMetadata(msg); got != "/tmp/repo" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeCodebaseAttachments_keywordFallback(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	widgetDir := filepath.Join(root, "core", "obscure", "internal")
	if err := os.MkdirAll(widgetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package internal\n\nfunc ComputeObscureWidget() int { return 42 }\n"
	if err := os.WriteFile(filepath.Join(widgetDir, "widget.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := &protocol.Message{
		Content: "@codebase What does ComputeObscureWidget return?",
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": root,
			},
		},
	}
	MergeCodebaseAttachments(msg)
	raw, ok := msg.Metadata[MetadataPromptAttachments]
	if !ok {
		t.Fatal("expected prompt_attachments")
	}
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatalf("expected chunks, got %v", raw)
	}
}

func TestSemanticHitsContainAnySymbol(t *testing.T) {
	t.Parallel()
	hits := []codeintel.RepoSearchHit{
		{Hit: codeintel.Hit{Path: "README.md", Content: "overview only"}},
	}
	if semanticHitsContainAnySymbol(hits, []string{"ComputeObscureWidget"}) {
		t.Fatal("expected false when symbol missing from semantic hits")
	}
	hits[0].Content = "func ComputeObscureWidget() int { return 42 }"
	if !semanticHitsContainAnySymbol(hits, []string{"ComputeObscureWidget"}) {
		t.Fatal("expected true when symbol present in hit content")
	}
}

func TestMentionedSourcePaths(t *testing.T) {
	t.Parallel()
	got := mentionedSourcePaths("Plan how to add HelloWorld to core/sample/main.go please")
	if len(got) != 1 || got[0] != "core/sample/main.go" {
		t.Fatalf("got %v", got)
	}
	if mentionedSourcePaths("plan the architecture") != nil {
		t.Fatal("expected no path tokens")
	}
}

func TestFilterCodebaseHitsDropsSitePackagesAndUnmentioned(t *testing.T) {
	t.Parallel()
	hits := []codeintel.RepoSearchHit{
		{Hit: codeintel.Hit{Path: "lib/python3.12/site-packages/PIL/TiffImagePlugin.py", Content: "IFD"}},
		{Hit: codeintel.Hit{Path: "core/sample/main.go", Content: "package main"}},
		{Hit: codeintel.Hit{Path: "docs/IDE_V3.md", Content: "Plan mode"}},
	}
	got := filterCodebaseHits(hits, "add HelloWorld to core/sample/main.go", nil)
	if len(got) != 1 || got[0].Path != "core/sample/main.go" {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeCodebaseForRoute_planSkipsUnscopedDump(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := &protocol.Message{
		Content: "Plan how to add a helper",
		Metadata: map[string]interface{}{
			"composer_mode": "plan",
			"editor_mode":   "plan",
			"workspace_context": map[string]interface{}{
				"workspace_path": root,
			},
		},
	}
	if MergeCodebaseForRoute(msg, routing.KnowledgePlan{Targets: []routing.RouteTarget{routing.RouteCodebase}}) {
		t.Fatal("plan without mentioned paths must not dump unscoped codebase chunks")
	}
}

func TestMergeCodebaseForRoute_constrainedAgentSkipsUnscopedDump(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Info: protocol.AgentInfo{AIProvider: "ollama", AIModel: "qwen3.5:9b"}}
	msg := &protocol.Message{
		Content: "add a helper function",
		Metadata: map[string]interface{}{
			"composer_mode": "agent",
			"editor_mode":   "agent",
			"workspace_context": map[string]interface{}{
				"workspace_path": root,
			},
		},
	}
	_ = a.turnContextProfile(msg)
	if MergeCodebaseForRoute(msg, routing.KnowledgePlan{Targets: []routing.RouteTarget{routing.RouteCodebase}}) {
		t.Fatal("constrained agent without mentioned paths must not dump unscoped codebase chunks")
	}
}
