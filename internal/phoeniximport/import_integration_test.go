package phoeniximport

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestImportAnalysisIntegration(t *testing.T) {
	settings := phoenixIntegrationSettings(t)
	requirePhoenixAuthenticated(t, settings)
	items, err := ListAnalyses(context.Background(), settings, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one COMPLETE analysis")
	}
	root := t.TempDir()
	res, err := ImportAnalysis(context.Background(), ImportRequest{
		WorkspaceRoot: root,
		AnalysisID:    items[0].ID,
		Settings:      settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(root, res.AnalysisDir, "reports", "results.json")
	if st, err := os.Stat(resultsPath); err != nil {
		t.Fatalf("results.json missing: %v", err)
	} else if st.Size() == 0 {
		t.Fatal("results.json empty")
	}
}
