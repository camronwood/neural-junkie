package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLICommandCandidates_dedupes(t *testing.T) {
	cfg := CLIAgentConfig{
		Command:            "copilot",
		AlternateCommands:  []string{"github-copilot-cli", "copilot"},
	}
	got := CLICommandCandidates(cfg)
	if len(got) != 2 || got[0] != "copilot" || got[1] != "github-copilot-cli" {
		t.Fatalf("candidates = %v, want [copilot github-copilot-cli]", got)
	}
}

func TestEffectiveBaseArgs_copilotLegacy(t *testing.T) {
	cfg, _ := GetCLIAgentConfig("copilot")
	args := EffectiveBaseArgs(cfg, "github-copilot-cli")
	if len(args) != 0 {
		t.Fatalf("legacy copilot args = %v, want nil", args)
	}
	args = EffectiveBaseArgs(cfg, "copilot")
	if len(args) != 1 || args[0] != "-p" {
		t.Fatalf("modern copilot args = %v, want [-p]", args)
	}
}

func TestEffectiveBaseArgs_codex(t *testing.T) {
	cfg, _ := GetCLIAgentConfig("codex")
	args := EffectiveBaseArgs(cfg, "codex")
	if len(args) != 1 || args[0] != "exec" {
		t.Fatalf("codex args = %v, want [exec]", args)
	}
}

func TestResolveCLIWithPATH_returnsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-gemini")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := CLIAgentConfig{Command: "fake-gemini", BaseArgs: []string{"-p"}}
	resolved, ok := ResolveCLIWithPATH(cfg, dir)
	if !ok {
		t.Fatal("expected resolve")
	}
	if resolved.Command != bin {
		t.Fatalf("Command=%q, want absolute %q", resolved.Command, bin)
	}
	if len(resolved.BaseArgs) != 1 || resolved.BaseArgs[0] != "-p" {
		t.Fatalf("BaseArgs=%v", resolved.BaseArgs)
	}
}

func TestListCLIAgentTypes_includesCommonAgents(t *testing.T) {
	types := ListCLIAgentTypes()
	want := []string{"aider", "amazonq", "amp", "claude", "codex", "copilot", "crush", "cursor", "droid", "gemini", "kiro", "opencode"}
	if len(types) != len(want) {
		t.Fatalf("len(types) = %d, want %d: %v", len(types), len(want), types)
	}
	set := make(map[string]bool, len(types))
	for _, typ := range types {
		set[typ] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Fatalf("ListCLIAgentTypes missing %q: %v", w, types)
		}
	}
}
