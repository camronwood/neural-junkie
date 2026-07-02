package collaboration

import "testing"

func TestStripNestedCollabPrefixes(t *testing.T) {
	in := "collabs/collabs/abc-123/report.md"
	got := stripNestedCollabPrefixes(in)
	want := "collabs/abc-123/report.md"
	if got != want {
		t.Fatalf("stripNestedCollabPrefixes(%q) = %q, want %q", in, got, want)
	}
}

func TestNormalizeDeliverableRelPath_stripsNestedCollabsInProject(t *testing.T) {
	c := &Collaboration{
		ID:             "abc-12345-6789-90ab-cdef00000000",
		SourceRepoPath: "/tmp/repo",
	}
	got := normalizeDeliverableRelPath(c, "collabs/collabs/abc-1234/report.md")
	want := "report.md"
	if got != want {
		t.Fatalf("normalizeDeliverableRelPath = %q, want %q", got, want)
	}
}
