package mcp

import "testing"

func TestIsHubMCPToolName_biologyQC(t *testing.T) {
	for _, name := range []string{"summarize_panel_qc", "run_12plex_qc", "SUMMARIZE_PANEL_QC"} {
		if !IsHubMCPToolName(name) {
			t.Fatalf("expected %q to be hub MCP tool", name)
		}
	}
	if IsHubMCPToolName("summarize_panel_qc.sh") {
		t.Fatal("expected false for non-tool suffix")
	}
}
