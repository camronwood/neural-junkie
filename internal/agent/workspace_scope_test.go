package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestLinkedWorkspacesFromMetadata_dedupesPrimary(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/primary",
				"workspace_name": "primary",
			},
			MetadataLinkedWorkspaces: []interface{}{
				map[string]interface{}{
					"workspace_path": "/tmp/primary",
					"workspace_name": "primary",
				},
				map[string]interface{}{
					"workspace_path": "/tmp/linked",
					"workspace_name": "linked",
				},
			},
		},
	}
	linked := linkedWorkspacesFromMetadata(msg)
	if len(linked) != 1 || linked[0].Path != "/tmp/linked" {
		t.Fatalf("linked=%+v", linked)
	}
	scoped := scopedWorkspacesFromMetadata(msg)
	if len(scoped) != 2 {
		t.Fatalf("scoped=%+v", scoped)
	}
}

func TestSelectReposForConsult_includesLinked(t *testing.T) {
	t.Parallel()
	msg := &protocol.Message{
		Content: "How do cross-repo-primary and cross-repo-linked connect?",
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": "/tmp/primary",
				"workspace_name": "primary",
			},
			MetadataLinkedWorkspaces: []interface{}{
				map[string]interface{}{
					"workspace_path": "/tmp/linked",
					"workspace_name": "linked",
					"open_files":     []interface{}{map[string]interface{}{"path": "foo.go"}},
				},
			},
		},
	}
	refs := selectReposForConsult(msg, IntentSubstantive)
	if len(refs) < 2 {
		t.Fatalf("refs=%+v", refs)
	}
}

func TestResolveWorkspaceForRelativePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	other := t.TempDir()
	file := filepath.Join(root, "pkg", "foo.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := &protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": other,
				"workspace_name": "other",
			},
			MetadataLinkedWorkspaces: []interface{}{
				map[string]interface{}{
					"workspace_path": root,
					"workspace_name": "root",
					"source":         "open_tab",
				},
			},
		},
	}
	ref, ok := resolveWorkspaceForRelativePath(msg, "pkg/foo.go")
	if !ok || ref.Path != root {
		t.Fatalf("ref=%+v ok=%v", ref, ok)
	}
}
