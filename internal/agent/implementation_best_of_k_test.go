package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotRestoreWorkspaceFiles(t *testing.T) {
	dir := t.TempDir()
	rel := "Makefile"
	path := filepath.Join(dir, rel)
	if err := os.WriteFile(path, []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := snapshotWorkspaceFiles(dir, []string{rel})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreWorkspaceFiles(dir, snap); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "all:\n" {
		t.Fatalf("got %q", data)
	}
}

func TestOutcomeScoreBestOfK(t *testing.T) {
	if outcomeScore(map[string]interface{}{"outcome": "applied_and_verified"}) <= outcomeScore(map[string]interface{}{"outcome": "applied_verify_failed"}) {
		t.Fatal("verified should outrank verify failed")
	}
}
