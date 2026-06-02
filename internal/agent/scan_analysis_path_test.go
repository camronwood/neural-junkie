package agent

import (
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSharedScanAnalysisPathFromOpenFiles(t *testing.T) {
	fixture := filepath.Join("..", "..", "testdata", "scan-analysis")
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	resultsPath := filepath.Join(abs, "reports", "results.json")
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "general", protocol.AgentInfo{Name: "u"}, "summarize")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": abs,
			"open_files": []interface{}{
				map[string]interface{}{
					"path":              resultsPath,
					"is_active":         true,
					"scan_analysis_dir": abs,
					"view_mode":         "scan-analysis",
				},
			},
		},
	}
	path, ok := sharedScanAnalysisPath(msg)
	if !ok {
		t.Fatal("expected path")
	}
	if !scanAnalysisPathExists(path) {
		t.Fatalf("got %q", path)
	}
}
