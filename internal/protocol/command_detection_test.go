package protocol

import "testing"

func TestDetectCommandsSkipsBareFilePathsInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "Write output to:\n\n```bash\nfindings.md\n```\n"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for bare file path, got %#v", suggestions)
	}
}

func TestDetectCommandsKeepsRealShellInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```bash\ncat collabs/abc/findings.md\n```"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Command != "cat collabs/abc/findings.md" {
		t.Fatalf("unexpected command %q", suggestions[0].Command)
	}
}

func TestDetectCommandsSkipsMCPToolNamesInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "Run QC:\n\n```bash\nsummarize_scan_summary\n```\n"
	suggestions := cd.DetectCommands(content, "BiologyExpert", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no shell suggestion for MCP tool name, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsMCPToolInInlineBackticks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "Run `summarize_scan_analysis` on the open file.\n"
	suggestions := cd.DetectCommands(content, "BiologyExpert", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no shell suggestion for inline MCP tool, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsMCPToolMultilineBash(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```bash\nsummarize_scan_analysis\n/path/to/export\n```\n"
	suggestions := cd.DetectCommands(content, "BiologyExpert", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no shell suggestion for MCP tool bash block, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsCollabDeliverablePathOnly(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```bash\ncollabs/902f2cf4-0626-4726-835a-4f1b715c23f6/schema-standardization.md\n```"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %#v", suggestions)
	}
}
