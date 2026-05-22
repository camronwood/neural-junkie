package lsp

import "testing"

func TestParseGoplsCheckOutput(t *testing.T) {
	root := "/tmp/ws"
	text := "pkg/foo.go:10:2: undefined: Bar\n"
	got := parseGoplsCheckOutput(root, text)
	if len(got) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(got))
	}
	if got[0].Line != 10 || got[0].Message == "" {
		t.Fatalf("unexpected: %+v", got[0])
	}
}
