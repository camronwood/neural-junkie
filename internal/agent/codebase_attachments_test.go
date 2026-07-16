package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
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
