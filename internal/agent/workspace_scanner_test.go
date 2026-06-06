package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/repo"
)

func TestWorkspaceScanPathPriority_prefersComponentsOverDocs(t *testing.T) {
	t.Parallel()
	query := "settings modal dark light theme font size"
	component := workspaceScanPathPriority("src/components/SettingsModal.tsx", query, protocol.AgentTypeFrontend)
	docs := workspaceScanPathPriority("docs/release-notes.html", query, protocol.AgentTypeFrontend)
	if component >= docs {
		t.Fatalf("expected component priority %d < docs %d", component, docs)
	}
}

func TestBuildWorkspaceScanQuery_includesHistory(t *testing.T) {
	t.Parallel()
	history := []*protocol.Message{
		{Content: "add settings modal with themes dark/light"},
	}
	q := BuildWorkspaceScanQuery("ok goahead", history)
	if q == "" || !strings.Contains(q, "ok goahead") || !strings.Contains(q, "settings modal") {
		t.Fatalf("expected merged query, got %q", q)
	}
}

func TestSortFilesForWorkspaceScan(t *testing.T) {
	t.Parallel()
	files := []*repo.SourceFile{
		{Path: "docs/release-notes.html"},
		{Path: "src/components/SettingsModal.tsx"},
		{Path: "README.md"},
	}
	sorted := sortFilesForWorkspaceScan(files, "settings modal theme", protocol.AgentTypeFrontend)
	if sorted[0].Path != "src/components/SettingsModal.tsx" {
		t.Fatalf("expected SettingsModal first, got %v", sorted[0].Path)
	}
}
