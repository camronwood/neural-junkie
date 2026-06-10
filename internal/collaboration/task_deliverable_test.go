package collaboration

import (
	"strings"
	"testing"
)

func TestTaskRequiresFileDeliverable(t *testing.T) {
	tasks := []struct {
		title string
		desc  string
		want  bool
	}{
		{"Draft findings", "Write collabs/abc/findings.md with API notes", true},
		{"Findings", "Document findings and next steps in collabs/abc/findings.md", true},
		{"Review", "Summarize risks in chat only", false},
	}
	for _, tc := range tasks {
		got := TaskRequiresFileDeliverable(CollaborationTask{Title: tc.title, Description: tc.desc})
		if got != tc.want {
			t.Fatalf("%q / %q: got %v want %v", tc.title, tc.desc, got, tc.want)
		}
	}
}

func TestTaskDispatchFileDeliverableNote(t *testing.T) {
	note := TaskDispatchFileDeliverableNote(CollaborationTask{Description: "Write collabs/x/out.md"})
	if note == "" || !strings.Contains(note, "FILE_CHANGE") || !strings.Contains(note, "completed") {
		t.Fatalf("unexpected note: %q", note)
	}
	if !strings.Contains(note, "docker-compose") {
		t.Fatalf("expected markdown task note to warn against stack tooling, got: %q", note)
	}
}

func TestSanitizePathToken_rejectsTruncatedPaths(t *testing.T) {
	cases := map[string]string{
		"collabs/b222bffe/index....":           "",
		"collabs/b2.../index.html":             "",
		"collabs/x/<|redacted|>/plan.md":       "",
		"collabs/x/frontend_architecture_plan.md": "collabs/x/frontend_architecture_plan.md",
		"index.html":                           "index.html",
	}
	for in, want := range cases {
		got := sanitizePathToken(in)
		if got != want {
			t.Fatalf("sanitizePathToken(%q) = %q want %q", in, got, want)
		}
	}
}

func TestReferencedDeliverablePaths_ignoresTruncatedTitleTokens(t *testing.T) {
	task := CollaborationTask{
		Title: "Create collabs/b222bffe/index.html, `collabs/b2...`",
	}
	paths := ReferencedDeliverablePaths(task)
	if len(paths) != 1 || paths[0] != "collabs/b222bffe/index.html" {
		t.Fatalf("expected single sanitized path, got %v", paths)
	}
}

func TestTaskLooksLikeMarkdownDeliverable(t *testing.T) {
	if !TaskLooksLikeMarkdownDeliverable(CollaborationTask{Description: "Write collabs/x/out.md"}) {
		t.Fatal("expected markdown deliverable")
	}
	if TaskLooksLikeMarkdownDeliverable(CollaborationTask{Description: "Write collabs/x/schema.json"}) {
		t.Fatal("json deliverable should not count as markdown-only")
	}
}
