package hub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollaborateNoWorkspaceFlag(t *testing.T) {
	h := NewHub()
	repo := t.TempDir()
	path, warn := h.resolveCollaborateSourceRepoPath(&protocol.Message{
		Metadata: map[string]interface{}{
			"workspace_context": map[string]interface{}{
				"workspace_path": repo,
			},
		},
	}, collaborateFlagParse{NoWorkspace: true})
	if path != "" || warn != "" {
		t.Fatalf("expected empty path for --no-workspace, got path=%q warn=%q", path, warn)
	}
}

func TestCollaborateRepoFlagOverridesMetadata(t *testing.T) {
	h := NewHub()
	repo := t.TempDir()
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "marker.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	path, warn := h.resolveCollaborateSourceRepoPath(&protocol.Message{
		Metadata: map[string]interface{}{
			"collab_source_mode": "active",
			"workspace_context": map[string]interface{}{
				"workspace_path": other,
			},
		},
	}, collaborateFlagParse{RepoPath: repo})
	if warn != "" {
		t.Fatalf("unexpected warn: %s", warn)
	}
	if path != repo {
		t.Fatalf("path = %q want %q", path, repo)
	}
}
