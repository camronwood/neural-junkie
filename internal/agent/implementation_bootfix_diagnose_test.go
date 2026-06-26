package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/server"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/mcp/workspace"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestPlaybookSignatureFromCommandEvidence_onlyPastedOutput(t *testing.T) {
	pasted := "make: *** No rule to make target 'start-all'.  Stop."
	if got := playbookSignatureFromCommandEvidence(pasted); got != "missing_start_all_target" {
		t.Fatalf("got %q", got)
	}
	vague := "The app is not booting. I think there is an issue with the makefile."
	if got := playbookSignatureFromCommandEvidence(vague); got != "" {
		t.Fatalf("vague message should not match playbook, got %q", got)
	}
}

func TestBootFixDiagnosticCommands_includesMakeStartAll(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "scripts", "start-all.sh"), []byte("#!/bin/sh\n"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\tnpm run build\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"tsc"}}`), 0o644)
	manifest := DetectStackManifest(dir)
	cmds := bootFixDiagnosticCommands(dir, manifest)
	if len(cmds) == 0 || cmds[0] != "make start-all" {
		t.Fatalf("cmds = %v", cmds)
	}
}

func TestRunBootFixDiagnosticBootstrap_provesPlaybookViaMake(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "start-all.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\tnpm run build\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := server.NewMCPServer("bootfix-test", "1.0.0")
	workspace.AttachTools(srv, func() string { return dir })

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	ag.MCPServer = &stubMCP{srv: srv}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"The app is not booting. Can you fix the makefile?")
	msg.Metadata = map[string]interface{}{
		"implementation_session": true,
		"workspace_context": map[string]interface{}{
			"workspace_path": dir,
		},
	}
	state := &ImplementationSessionState{
		BootFixIntent: true,
		StackManifest: DetectStackManifest(dir),
	}
	ctx := withImplementationSessionState(context.Background(), state)
	ctx = attachImplSessionCommandPolicy(ctx, state)

	applied, note := ag.runBootFixDiagnosticBootstrap(ctx, msg, state, dir)
	if !applied {
		t.Fatalf("expected playbook after make start-all diagnostic, note=%q", note)
	}
	if state.PlaybookUsed() != "missing_start_all_target" {
		t.Fatalf("playbook = %q", state.PlaybookUsed())
	}
	if state.ProposedCount < 1 {
		t.Fatal("expected proposal count after playbook")
	}
}
