package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendAndPath(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	if err := Append("collab-test", Event{Type: "phase_transition", FromPhase: "planning", ToPhase: "reviewing"}); err != nil {
		t.Fatal(err)
	}
	p, err := EventLogPath("collab-test")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected event line")
	}
	want := filepath.Join(dir, ".neural-junkie", "workflow-events", "collab-test.jsonl")
	if p != want {
		t.Fatalf("path=%q want %q", p, want)
	}
}
