package agent

import (
	"strings"
	"testing"
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
