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

func TestDetectCommandsSkipsBiologyPanelQCToolInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```bash\nsummarize_panel_qc /path/to/analysis\n```\n"
	suggestions := cd.DetectCommands(content, "BiologyExpert", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no shell suggestion for summarize_panel_qc, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsRun12PlexQCInline(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "Call `run_12plex_qc` on the analysis folder.\n"
	suggestions := cd.DetectCommands(content, "BiologyExpert", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no shell suggestion for run_12plex_qc, got %#v", suggestions)
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

func TestIsSafeShellCommand(t *testing.T) {
	cases := []struct {
		cmd  string
		safe bool
	}{
		{"cat README.md", true},
		{"grep -r schema internal/", true},
		{"git status", true},
		{"go test ./...", true},
		{"npm test", true},
		{"npm run build", true},
		{"ls -la collabs/abc", true},
		{"rm -rf node_modules", false},
		{"npm install", false},
		{"curl -X POST http://example.com", false},
		{"echo hello > out.txt", false},
		{"unknown-binary --help", false},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			got := IsSafeShellCommand(tc.cmd)
			if got != tc.safe {
				t.Fatalf("IsSafeShellCommand(%q) = %v, want %v", tc.cmd, got, tc.safe)
			}
		})
	}
}

func TestDetectCommandsMarksSafeShellMetadata(t *testing.T) {
	cd := NewCommandDetector(nil)
	suggestions := cd.DetectCommands("```bash\ncat README.md\n```", "Agent", "msg-1")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if !suggestions[0].IsSafe {
		t.Fatalf("expected cat to be safe, got %#v", suggestions[0])
	}
}

func TestDetectCommandsSkipsModuleImportPathsInline(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "`github.com/camron/ai-chat-room/internal/confluence/storage.go`"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for module import path, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsGolangciLintOutputInline(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "- `github.com/camron/ai-chat-room/internal/confluence/storage.go:64.16,66.3 1 0` suggests issue"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for linter output line, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsGolangciLintOutputInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```sh\ngithub.com/camron/ai-chat-room/internal/confluence/storage.go:64.16,66.3 1 0\n```"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for linter output in bash block, got %#v", suggestions)
	}
}

func TestDetectCommandsSkipsMultiLineBarePathsInBashBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```bash\ngithub.com/camron/ai-chat-room/internal/confluence/storage.go\ngithub.com/camron/ai-chat-room/internal/confluence/storage.go:64.16,66.3 1 0\n```"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 0 {
		t.Fatalf("expected no suggestions for path-only bash block, got %#v", suggestions)
	}
}

func TestDetectCommandsKeepsGoRunInShBlocks(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "```sh\ngo run main.go\n```"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Command != "go run main.go" {
		t.Fatalf("unexpected command %q", suggestions[0].Command)
	}
}

func TestDetectCommandsGitCommandStillDetected(t *testing.T) {
	cd := NewCommandDetector(nil)
	content := "`git status`"
	suggestions := cd.DetectCommands(content, "Agent", "msg-1")
	if len(suggestions) != 1 {
		t.Fatalf("expected git status suggestion, got %#v", suggestions)
	}
}
