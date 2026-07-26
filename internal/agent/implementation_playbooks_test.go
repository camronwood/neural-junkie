package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestSynthesizeMakefileStartAllPlaybookBody(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "start-all.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := ".PHONY: help\nhelp:\n\t@echo ok\n"
	body := synthesizeMakefileWithStartAll(existing)
	if body == "" || !containsAllParts(body, "start-all:", "scripts/start-all.sh") {
		t.Fatalf("body missing start-all target: %q", body)
	}
	sig := commandOutputMatchesPlaybook("make: *** No rule to make target 'start-all'.  Stop.")
	if sig != "missing_start_all_target" {
		t.Fatalf("sig = %q", sig)
	}
}

func TestTryEarlyCommandEvidencePlaybook_requiresPastedOutput(t *testing.T) {
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

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{BootFixIntent: true}
	meta := map[string]interface{}{
		"implementation_session": true,
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}

	vague := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"The app is not booting. Can you fix the makefile?")
	vague.Metadata = meta
	if ag.tryEarlyCommandEvidencePlaybook(context.Background(), vague, dir, state) {
		t.Fatal("vague message should not trigger early playbook")
	}

	pasted := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"make start-all fails:\n```\n$ make start-all\nmake: *** No rule to make target 'start-all'.  Stop.\n```")
	pasted.Metadata = meta
	if !ag.tryEarlyCommandEvidencePlaybook(context.Background(), pasted, dir, state) {
		t.Fatal("pasted command output should trigger early playbook")
	}
	if !makefileHasStartAllTarget(dir) {
		t.Fatal("Makefile on disk should contain start-all after playbook")
	}
}

func TestTryEarlyCommandEvidencePlaybook_survivesRollback(t *testing.T) {
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

	ag := NewAgent(protocol.AgentTypeFrontend, "FrontendEngineer", nil, ai.NewMockProvider(), shouldRespondTestHub{})
	state := &ImplementationSessionState{
		BootFixIntent: true,
		TrustMode:     editorTrustAutoApply,
		VerifyFailed:  true, // would normally trigger rollback
	}
	meta := map[string]interface{}{
		"implementation_session": true,
		"editor_agent_trust":     "auto_apply_edits",
		"workspace_context":      map[string]interface{}{"workspace_path": dir},
	}
	pasted := protocol.NewMessage(protocol.MessageTypeQuestion, "dm-test",
		protocol.AgentInfo{ID: "human", Name: "camron", Type: "human"},
		"make start-all fails:\n```\n$ make start-all\nmake: *** No rule to make target 'start-all'.  Stop.\n```")
	pasted.Metadata = meta
	if !ag.tryEarlyCommandEvidencePlaybook(context.Background(), pasted, dir, state) {
		t.Fatal("pasted command output should trigger early playbook")
	}
	state.rollbackFailedAutoApplySession(dir)
	if !makefileHasStartAllTarget(dir) {
		t.Fatal("Makefile start-all must survive session rollback after playbook")
	}
}
