package agent

import (
	"strings"
	"testing"
)

func TestBuildDialogueSummary(t *testing.T) {
	state := &ImplementationSessionState{
		LastReadPaths:     []string{"Makefile", "package.json"},
		FilesChanged:      []string{"Makefile"},
		LastFailedCommand: "make start-all",
		LastCommandOutputText: "No rule to make target 'start-all'",
		CommandHistory:    []CommandRunRecord{{Cmd: "make start-all"}},
	}
	summary := state.buildDialogueSummary()
	if !strings.Contains(summary, "Makefile") || !strings.Contains(summary, "make start-all") {
		t.Fatalf("summary=%q", summary)
	}
}

func TestAppendDialogueSummaryPrompt(t *testing.T) {
	state := &ImplementationSessionState{DialogueSummary: "Compressed session context"}
	out := appendDialogueSummaryPrompt("base", state)
	if !strings.Contains(out, "SESSION SUMMARY") {
		t.Fatalf("out=%q", out)
	}
}
