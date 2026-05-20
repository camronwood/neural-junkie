package meetnotes

import "testing"

func TestExtractDocID(t *testing.T) {
	body := `<a href="https://docs.google.com/document/d/1abcDEF-xyz_123/edit?usp=sharing">Open meeting notes</a>`
	got := ExtractDocID(body)
	if got != "1abcDEF-xyz_123" {
		t.Fatalf("ExtractDocID() = %q, want 1abcDEF-xyz_123", got)
	}
}

func TestExtractDocID_missing(t *testing.T) {
	if got := ExtractDocID("no link here"); got != "" {
		t.Fatalf("ExtractDocID() = %q, want empty", got)
	}
}

func TestDocLink(t *testing.T) {
	want := "https://docs.google.com/document/d/abc123/edit"
	if got := DocLink("abc123"); got != want {
		t.Fatalf("DocLink() = %q, want %q", got, want)
	}
}
