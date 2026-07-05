package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestScopedRepoPathsFromMetadata(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{"workspace_path": "/a"},
			MetadataLinkedWorkspaces: []interface{}{
				map[string]interface{}{"workspace_path": "/b"},
			},
		},
	}
	paths := scopedRepoPathsFromMetadata(msg)
	if len(paths) != 2 || paths[0] != "/a" || paths[1] != "/b" {
		t.Fatalf("paths=%v", paths)
	}
}

func TestMergeCodebaseSearchFallback_multiRepo(t *testing.T) {
	t.Parallel()
	rootA := t.TempDir()
	rootB := t.TempDir()
	writeSymbolFile(t, rootA, "AlphaWidget", 11)
	writeSymbolFile(t, rootB, "AlphaWidget", 22)
	msg := &protocol.Message{Metadata: map[string]interface{}{}}
	ok := mergeCodebaseSearchFallback(msg, []string{rootA, rootB}, "AlphaWidget")
	if !ok {
		t.Fatal("expected fallback hits")
	}
	raw := msg.Metadata[MetadataPromptAttachments]
	arr, ok := raw.([]interface{})
	if !ok || len(arr) < 2 {
		t.Fatalf("attachments=%v", raw)
	}
}

func writeSymbolFile(t *testing.T, root, sym string, val int) {
	t.Helper()
	dir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("package pkg\n\nfunc %s() int { return %d }\n", sym, val)
	if err := os.WriteFile(filepath.Join(dir, "widget.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
