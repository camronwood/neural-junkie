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
	note := TaskDispatchFileDeliverableNote(CollaborationTask{Description: "Write collabs/x/out.md"}, "")
	if note == "" || !strings.Contains(note, "FILE_CHANGE") || !strings.Contains(note, "completed") {
		t.Fatalf("unexpected note: %q", note)
	}
	if !strings.Contains(note, "docker-compose") {
		t.Fatalf("expected markdown task note to warn against stack tooling, got: %q", note)
	}
	if strings.Contains(note, "Research deliverable") {
		t.Fatalf("generic markdown task should not get research note: %q", note)
	}
}

func TestTaskDispatchFileDeliverableNote_researchFindings(t *testing.T) {
	cases := []struct {
		name string
		task CollaborationTask
		goal string
	}{
		{
			name: "summarize wording",
			task: CollaborationTask{
				Description: "Summarize README.md and core/sample/main.go in collabs/x/findings.md",
			},
		},
		{
			name: "citing wording",
			task: CollaborationTask{
				Description: "Write collabs/x/findings.md citing README.md and core/sample/main.go",
			},
		},
		{
			name: "three bullets wording",
			task: CollaborationTask{
				Description: "Write collabs/x/findings.md with three bullets from README.md and core/sample/main.go",
			},
		},
		{
			name: "collab goal cites sources when task omits wording",
			task: CollaborationTask{
				Description: "Write collabs/x/findings.md",
			},
			goal: "@SoftwareArchitect @BackendEngineer Plan one task: Write findings.md with three bullets citing README.md and core/sample/main.go only.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			note := TaskDispatchFileDeliverableNote(tc.task, tc.goal)
			if !strings.Contains(note, "Research deliverable") {
				t.Fatalf("expected research note, got: %q", note)
			}
			if !strings.Contains(note, "at least three substantive Markdown bullet") {
				t.Fatalf("expected bullet requirement, got: %q", note)
			}
			if !strings.Contains(note, "Cite or reference every source path") {
				t.Fatalf("expected citation requirement, got: %q", note)
			}
			if !strings.Contains(note, "README.md") || !strings.Contains(note, "core/sample/main.go") {
				t.Fatalf("expected named source paths, got: %q", note)
			}
			if !strings.Contains(note, "task list") || !strings.Contains(note, "guessed stack") {
				t.Fatalf("expected anti-pattern guidance, got: %q", note)
			}
		})
	}
}

func TestTaskDispatchFileDeliverableNote_researchScopeLimit(t *testing.T) {
	task := CollaborationTask{
		Description: "Write collabs/x/findings.md summarizing README.md and core/sample/main.go only",
	}
	goal := "Plan one task: findings.md from README.md and core/sample/main.go only."
	note := TaskDispatchFileDeliverableNote(task, goal)
	if !strings.Contains(note, "Scope limit") {
		t.Fatalf("expected scope limit for 'only' research task, got: %q", note)
	}
	if !strings.Contains(note, "App.tsx") {
		t.Fatalf("expected frontend exclusion in scope limit, got: %q", note)
	}
}

func TestTaskDispatchFileDeliverableNote_nonResearchFindings(t *testing.T) {
	cases := []CollaborationTask{
		{Description: "Write collabs/x/findings.md with implementation notes"},
		{Description: "Draft collabs/x/plan.md summarizing architecture"},
	}
	for i, task := range cases {
		note := TaskDispatchFileDeliverableNote(task, "")
		if strings.Contains(note, "Research deliverable") {
			t.Fatalf("case %d: unexpected research note: %q", i, note)
		}
	}
}

func TestSanitizePathToken_rejectsTruncatedPaths(t *testing.T) {
	cases := map[string]string{
		"collabs/b222bffe/index....":              "",
		"collabs/b2.../index.html":                "",
		"collabs/x/<|redacted|>/plan.md":          "",
		"collabs/x/frontend_architecture_plan.md": "collabs/x/frontend_architecture_plan.md",
		"index.html": "index.html",
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
