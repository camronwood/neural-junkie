package fileedit

import (
	"testing"
)

func TestSearchReplace_unique(t *testing.T) {
	t.Parallel()
	out, err := SearchReplace("hello world", "world", "there", false)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello there" {
		t.Fatalf("got %q", out)
	}
}

func TestSearchReplace_notUnique(t *testing.T) {
	t.Parallel()
	_, err := SearchReplace("aaa", "a", "b", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if pe, ok := err.(*PatchError); !ok || pe.Code != ErrNotUnique {
		t.Fatalf("got %v", err)
	}
}

func TestSearchReplaceWithFallback_trimmedTrailingSpaces(t *testing.T) {
	t.Parallel()
	content := "abc  \ndef"
	old := "abc\ndef"
	new := "ABC\ndef"
	out, strategy, err := SearchReplaceWithFallback(content, old, new, false)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != "trimmed_trailing_spaces" {
		t.Fatalf("strategy %q", strategy)
	}
	if out != "ABC\ndef" {
		t.Fatalf("got %q", out)
	}
}

func TestRequireOldStringInSelection_lineRangeFallback(t *testing.T) {
	t.Parallel()
	scope := &SelectionScope{Path: "a.go", StartLine: 2, EndLine: 4, Text: "stale selection text"}
	if err := RequireOldStringInSelection(scope, "actual old"); err != nil {
		t.Fatalf("expected line-range fallback to allow, got %v", err)
	}
}

func TestContentFingerprintAndHint(t *testing.T) {
	t.Parallel()
	fp := ContentFingerprint("hello\nworld\n")
	if len(fp) < 8 {
		t.Fatalf("fingerprint too short: %q", fp)
	}
	hint := ClosestLineHint("alpha\nhello world\nbeta", "hello")
	if hint == "" {
		t.Fatal("expected hint")
	}
}

func TestApplyUnifiedPatch_singleHunk(t *testing.T) {
	t.Parallel()
	old := "alpha\nbeta\ngamma\n"
	patch := `--- a/f
+++ b/f
@@ -1,3 +1,3 @@
 alpha
-beta
+BRAVO
 gamma
`
	out, err := ApplyUnifiedPatch(old, patch)
	if err != nil {
		t.Fatal(err)
	}
	want := "alpha\nBRAVO\ngamma"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestUnifiedDiff_roundTrip(t *testing.T) {
	t.Parallel()
	old := "package main\n\nfunc main() {\n\tprintln(1)\n}\n"
	newC := "package main\n\nfunc main() {\n\tprintln(2)\n}\n"
	diff := UnifiedDiff("main.go", old, newC)
	if diff == "" {
		t.Fatal("expected diff")
	}
	out, err := ApplyUnifiedPatch(old, diff)
	if err != nil {
		t.Fatal(err)
	}
	if out != trimTrailingNewline(newC) {
		t.Fatalf("round trip mismatch:\n%q\nvs\n%q", out, newC)
	}
}

func TestValidateSelectionScope_rejectsOutside(t *testing.T) {
	t.Parallel()
	old := "line1\nline2\nline3\n"
	newC := "CHANGED\nline2\nline3\n"
	scope := &SelectionScope{Path: "f.txt", StartLine: 2, EndLine: 3}
	if err := ValidateSelectionScope(scope, old, newC); err == nil {
		t.Fatal("expected out of scope")
	}
}

func trimTrailingNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
