package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/ai"
)

func TestToolActivityDetail(t *testing.T) {
	got := toolActivityDetail(ai.ToolStepEvent{
		Kind:    "start",
		Name:    "read_file",
		Preview: "package.json",
	})
	if got != "read_file — package.json" {
		t.Fatalf("start preview: %q", got)
	}
	got = toolActivityDetail(ai.ToolStepEvent{Kind: "error", Name: "run_command", Preview: "exit 1"})
	if got != "run_command failed — exit 1" {
		t.Fatalf("error: %q", got)
	}
}
