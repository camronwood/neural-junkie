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

func TestBootFixDiagnosticCommands_prefersBuildNotStartAll(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "scripts", "start-all.sh"), []byte("#!/bin/sh\n"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "Makefile"), []byte("start-all:\n\tbash scripts/start-all.sh\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"tsc"}}`), 0o644)
	manifest := DetectStackManifest(dir)
	cmds := bootFixDiagnosticCommands(dir, manifest)
	if len(cmds) == 0 || cmds[0] != "npm run build" {
		t.Fatalf("cmds = %v, want npm run build first", cmds)
	}
	for _, cmd := range cmds {
		if cmd == "make start-all" {
			t.Fatalf("bootstrap must not run make start-all: %v", cmds)
		}
	}
}

func TestBootFixDiagnosticCommands_usesTscWhenNoBuildScript(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"devDependencies":{"typescript":"^5.0.0"}}`), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "node_modules", ".bin"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "node_modules", ".bin", "tsc"), []byte("#!/bin/sh\n"), 0o755)
	manifest := DetectStackManifest(dir)
	cmds := bootFixDiagnosticCommands(dir, manifest)
	if len(cmds) != 1 || cmds[0] != "./node_modules/.bin/tsc --noEmit" {
		t.Fatalf("cmds = %v", cmds)
	}
}

func TestRunBootFixDiagnosticBootstrap_runsBuildDiagnostic(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "start-all.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("start-all:\n\tbash scripts/start-all.sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"node -e \"process.exit(1)\""}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := server.NewMCPServer("bootfix-test", "1.0.0")
	workspace.AttachTools(srv, func() string { return dir })

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	ag.MCPServer = &stubMCP{srv: srv}
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"The app is not booting.")
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
	state.RecordReadPath("Makefile")
	ctx := withImplementationSessionState(context.Background(), state)
	ctx = attachImplSessionCommandPolicy(ctx, state)

	applied, note := ag.runBootFixDiagnosticBootstrap(ctx, msg, state, dir)
	if applied {
		t.Fatalf("build failure should not auto-apply playbook, note=%q", note)
	}
	if len(state.CommandHistory) == 0 {
		t.Fatal("expected diagnostic command history")
	}
	if state.CommandHistory[0].Cmd != "npm run build" {
		t.Fatalf("first cmd = %q", state.CommandHistory[0].Cmd)
	}
}

func TestCommandOutputMatchesPlaybook_expandedSignatures(t *testing.T) {
	cases := map[string]string{
		"error TS2307: Cannot find module 'react-bootstrap'": "missing_npm_module",
		"react-bootstrap (imported by /src/Header.tsx)\nAre they installed?": "missing_npm_module",
		"The following dependencies are imported but could not be resolved:\n\n  react-bootstrap (imported by": "missing_npm_module",
		"bootfix_hint=dev_server_timeout\nDev server command timed out": "dev_server_timeout",
		"Error: listen EADDRINUSE: address already in use :::5177":   "vite_strict_port_conflict",
		"strictPort is true and port 5177 is in use, trying 5179":    "vite_strict_port_conflict",
	}
	for output, want := range cases {
		if got := commandOutputMatchesPlaybook(output); got != want {
			t.Fatalf("commandOutputMatchesPlaybook(%q) = %q, want %q", output, got, want)
		}
	}
}
