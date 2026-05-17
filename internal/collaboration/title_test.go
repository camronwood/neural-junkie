package collaboration

import "testing"

func TestDeriveCollaborationTitle(t *testing.T) {
	got := DeriveCollaborationTitle("@Cursor @Gemini who should I ask questions about Rust code to?")
	if got == "" || len(got) > 80 {
		t.Fatalf("title: %q", got)
	}
	if got != "who should I ask questions about Rust code to?" {
		t.Fatalf("unexpected title: %q", got)
	}
}
