package ai

import (
	"encoding/json"
	"testing"
)

func TestParseToolCallFromTextBareJSON(t *testing.T) {
	name, input, ok := ParseToolCallFromText(`{"name":"summarize_scan_analysis","arguments":{"path":""}}`)
	if !ok || name != "summarize_scan_analysis" {
		t.Fatalf("parse failed: ok=%v name=%q", ok, name)
	}
	var args map[string]string
	if err := json.Unmarshal(input, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "" {
		t.Fatalf("path = %q", args["path"])
	}
}

func TestParseToolCallFromTextSkipsNonToolJSON(t *testing.T) {
	if _, _, ok := ParseToolCallFromText(`{"foo":"bar"}`); ok {
		t.Fatal("expected non-tool JSON to be ignored")
	}
}

func TestParseToolCallFromTextEmbeddedInProse(t *testing.T) {
	text := `I'll create that file for you.

json { "name": "write_openscad", "arguments": { "path": "ball.scad", "content": "sphere(10);" } }`
	name, input, ok := ParseToolCallFromText(text)
	if !ok || name != "write_openscad" {
		t.Fatalf("parse failed: ok=%v name=%q", ok, name)
	}
	var args map[string]string
	if err := json.Unmarshal(input, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "ball.scad" {
		t.Fatalf("path = %q", args["path"])
	}
}

func TestParseToolCallFromTextInlineCodeFence(t *testing.T) {
	text := "Here is the tool call:\n```json\n{\"name\":\"write_openscad\",\"arguments\":{\"path\":\"model.scad\"}}\n```\n"
	name, _, ok := ParseToolCallFromText(text)
	if !ok || name != "write_openscad" {
		t.Fatalf("parse failed: ok=%v name=%q", ok, name)
	}
}

func TestParseToolCallFromTextTagged(t *testing.T) {
	text := `I will read the file.
<tool_call>{"name":"read_file","arguments":{"path":"src/App.tsx"}}</tool_call>`
	name, input, ok := ParseToolCallFromText(text)
	if !ok || name != "read_file" {
		t.Fatalf("parse failed: ok=%v name=%q", ok, name)
	}
	var args map[string]string
	if err := json.Unmarshal(input, &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "src/App.tsx" {
		t.Fatalf("path = %q", args["path"])
	}
}

func TestStripToolCallFromText(t *testing.T) {
	text := `Done.
<tool_call>{"name":"read_file","arguments":{"path":"x"}}</tool_call>`
	got := StripToolCallFromText(text)
	if got != "Done." {
		t.Fatalf("got %q", got)
	}
}
