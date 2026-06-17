package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestParseLooseFileChange_InlinePath(t *testing.T) {
	resp := "[FILE_CHANGE] collabs/abc/findings.md\n\nThe main.go file is minimal.\n\nTASK_STATUS: completed"
	d, ok := parseLooseFileChange(resp)
	if !ok {
		t.Fatal("expected loose parse")
	}
	if d.Path != "collabs/abc/findings.md" {
		t.Fatalf("path = %q", d.Path)
	}
	if !strings.Contains(d.NewContent, "main.go") {
		t.Fatalf("body = %q", d.NewContent)
	}
}

func TestParseLooseFileChange_PathField(t *testing.T) {
	resp := "Done.\n[FILE_CHANGE]\npath: collabs/test/findings.md\n```new\n# Findings\n\n- one\n```\nTASK_STATUS: completed"
	d, ok := parseLooseFileChange(resp)
	if !ok {
		t.Fatal("expected loose parse")
	}
	if d.Path != "collabs/test/findings.md" {
		t.Fatalf("path = %q", d.Path)
	}
	if !strings.Contains(d.NewContent, "Findings") {
		t.Fatalf("body = %q", d.NewContent)
	}
}

func TestParseLooseFileChange_SkipsCanonicalBlock(t *testing.T) {
	resp := "[FILE_CHANGE]\noperation: create\npath: a.md\n```new\nx\n```\n[/FILE_CHANGE]"
	if _, ok := parseLooseFileChange(resp); ok {
		t.Fatal("canonical block should not use loose parser")
	}
}

func TestParseAllLooseFileChanges_CollabExecutionFormat(t *testing.T) {
	resp := `[FILE_CHANGE] operation: create path: collabs/abc/ui_spec.md content: ` + "```markdown\n# UI Spec\n\nColors: blue\n```\n" +
		`[FILE_CHANGE] operation: create path: collabs/abc/index.html content: ` + "```html\n<!DOCTYPE html>\n<title>Home</title>\n```\n" +
		"TASK_STATUS: completed\n"
	got := parseAllLooseFileChanges(resp)
	if len(got) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(got))
	}
	if got[0].Path != "collabs/abc/ui_spec.md" {
		t.Fatalf("first path = %q", got[0].Path)
	}
	if !strings.Contains(got[0].NewContent, "UI Spec") {
		t.Fatalf("first body = %q", got[0].NewContent)
	}
	if got[1].Path != "collabs/abc/index.html" {
		t.Fatalf("second path = %q", got[1].Path)
	}
}

func TestCollabFileChangeParseEnabled_CollabTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "ch", protocol.AgentInfo{Name: "System"}, "task")
	msg.SetCollaborationID("c1")
	msg.SetCollaborationPhase("executing")
	if !collabFileChangeParseEnabled(msg) {
		t.Fatal("expected collab task execution to enable file-change parse")
	}
}
