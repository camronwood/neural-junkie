package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
	"github.com/camronwood/neural-junkie/internal/scansummary"
)

func TestRewriteScanSummaryToolInputUsesSharedWorkspacePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, scansummary.MetadataFileName), []byte(`{"metadata":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-biologyexpert", protocol.AgentInfo{Name: "Camron"}, "summarize this scan")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
			"scan_summary": map[string]interface{}{
				"summary_dir": "",
			},
		},
	}

	out := rewriteScanSummaryToolInput(msg, "summarize_scan_summary", json.RawMessage(`{"path":"/path/to/your/open/image"}`))
	var args map[string]string
	if err := json.Unmarshal(out, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != dir {
		t.Fatalf("path = %q, want %q", args["path"], dir)
	}
}

func TestRewriteScanSummaryToolInputKeepsValidExplicitPath(t *testing.T) {
	shared := t.TempDir()
	explicit := t.TempDir()
	for _, dir := range []string{shared, explicit} {
		if err := os.WriteFile(filepath.Join(dir, scansummary.MetadataFileName), []byte(`{"metadata":[]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-camron-biologyexpert", protocol.AgentInfo{Name: "Camron"}, "summarize this scan")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_path": shared,
			"scan_summary": map[string]interface{}{
				"summary_dir": "",
			},
		},
	}

	out := rewriteScanSummaryToolInput(msg, "summarize_scan_summary", json.RawMessage(`{"path":"`+explicit+`"}`))
	var args map[string]string
	if err := json.Unmarshal(out, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != explicit {
		t.Fatalf("path = %q, want explicit %q", args["path"], explicit)
	}
}
