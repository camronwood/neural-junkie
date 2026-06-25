package agent

import (
	"os"
	"path/filepath"
	"testing"
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
