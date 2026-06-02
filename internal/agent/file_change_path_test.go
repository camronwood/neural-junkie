package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestIsValidFileChangeRelPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		ok   bool
	}{
		{"tailwind.config.js", true},
		{"src/App.tsx", true},
		{"File:", false},
		{"File", false},
		{"path", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isValidFileChangeRelPath(tc.path); got != tc.ok {
			t.Errorf("isValidFileChangeRelPath(%q) = %v, want %v", tc.path, got, tc.ok)
		}
	}
}

func TestParseLooseFileChange_RejectsFileLabel(t *testing.T) {
	t.Parallel()
	resp := "[FILE_CHANGE] File:\n```tsx\n<FileExplorerPanel />\n```"
	if _, ok := parseLooseFileChange(resp); ok {
		t.Fatal("should not accept File: as path")
	}
}

func TestPreferImplementationTargetPath(t *testing.T) {
	t.Parallel()
	user := "Please implement themes. Emit [FILE_CHANGE] for tailwind.config.js"
	got := preferImplementationTargetPath(user, "File:", protocol.AgentTypeFrontend)
	if got != "tailwind.config.js" {
		t.Fatalf("got %q, want tailwind.config.js", got)
	}
}

func TestResolveLooseFileChangePath_PathFieldWins(t *testing.T) {
	t.Parallel()
	tail := "[FILE_CHANGE] File:\npath: tailwind.config.js\n```new\n{}\n```"
	if got := resolveLooseFileChangePath(tail); got != "tailwind.config.js" {
		t.Fatalf("got %q", got)
	}
}
