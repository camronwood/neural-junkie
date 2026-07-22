package agent

import (
	"context"
	"strings"
	"testing"
)

func TestParseRunCommandMCPResult(t *testing.T) {
	code, out, _ := parseRunCommandMCPResult("exit_code=1\nbuild failed\n")
	if code != 1 || out != "build failed" {
		t.Fatalf("got code=%d out=%q", code, out)
	}
	code, out, _ = parseRunCommandMCPResult("exit_code=0\nok")
	if code != 0 || out != "ok" {
		t.Fatalf("got code=%d out=%q", code, out)
	}
}

func TestParseRunCommandMCPResult_errorIsFailure(t *testing.T) {
	code, out, errOut := parseRunCommandMCPResult("ERROR: Error in run_command: command not allowlisted: make start-all")
	if code != 1 {
		t.Fatalf("exit code=%d, want 1", code)
	}
	if out != "" {
		t.Fatalf("stdout=%q, want empty", out)
	}
	if !strings.Contains(errOut, "not allowlisted") {
		t.Fatalf("stderr=%q", errOut)
	}
	code, _, errOut = parseRunCommandMCPResult("command not allowlisted: npm install")
	if code != 1 || !strings.Contains(errOut, "not allowlisted") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestParseRunCommandToolInput(t *testing.T) {
	cmd := parseRunCommandToolInput([]byte(`{"command":"npm run build"}`))
	if cmd != "npm run build" {
		t.Fatalf("got %q", cmd)
	}
}

func TestFormatAgentRunCommandContentIncludesStdout(t *testing.T) {
	got := formatAgentRunCommandContent("FrontendEngineer", "make start-all", 0, "vite ready\n", "")
	if !strings.Contains(got, "stdout:") || !strings.Contains(got, "vite ready") {
		t.Fatalf("expected stdout in content, got:\n%s", got)
	}
	if !strings.Contains(got, "Exit code: 0") {
		t.Fatalf("expected exit code, got:\n%s", got)
	}
	fail := formatAgentRunCommandContent("FrontendEngineer", "make start-all", 1, "", "ERROR: not allowlisted")
	if !strings.Contains(fail, "stderr:") || !strings.Contains(fail, "not allowlisted") {
		t.Fatalf("expected stderr in content, got:\n%s", fail)
	}
}

func TestRunCommandTurnDedupe(t *testing.T) {
	ctx := withRunCommandTurnDedupe(context.Background())
	if cached, ok := lookupOrStoreRunCommandResult(ctx, "git status", "", false); ok || cached != "" {
		t.Fatal("expected miss on first lookup")
	}
	storeRunCommandTurnResult(ctx, "git status", "exit_code=0\nclean")
	cached, ok := lookupOrStoreRunCommandResult(ctx, " git   status ", "", false)
	if !ok || cached != "exit_code=0\nclean" {
		t.Fatalf("expected cache hit, ok=%v cached=%q", ok, cached)
	}
	// Different command should miss.
	if _, ok := lookupOrStoreRunCommandResult(ctx, "git log -1", "", false); ok {
		t.Fatal("expected miss for different command")
	}
}
