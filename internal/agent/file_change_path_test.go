package agent

import "testing"

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
	got := preferImplementationTargetPath("", user, "File:")
	if got != "tailwind.config.js" {
		t.Fatalf("got %q, want tailwind.config.js", got)
	}
}

func TestPreferImplementationTargetPath_atFileWins(t *testing.T) {
	t.Parallel()
	user := "In @file:src/App.tsx ONLY add a subtitle. Do NOT modify tailwind.config.js or package.json."
	got := preferImplementationTargetPath("", user, "File:")
	if got != "src/App.tsx" {
		t.Fatalf("got %q, want src/App.tsx", got)
	}
}

func TestDetectAtFilePaths(t *testing.T) {
	t.Parallel()
	got := DetectAtFilePaths("Fix @file:core/sample/math.go and @folder:src/ only")
	if len(got) != 1 || got[0] != "core/sample/math.go" {
		t.Fatalf("got %v", got)
	}
}

func TestResolveLooseFileChangePath_PathFieldWins(t *testing.T) {
	t.Parallel()
	tail := "[FILE_CHANGE] File:\npath: tailwind.config.js\n```new\n{}\n```"
	if got := resolveLooseFileChangePath(tail); got != "tailwind.config.js" {
		t.Fatalf("got %q", got)
	}
}
