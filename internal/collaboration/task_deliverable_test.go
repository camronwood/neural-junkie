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
}
