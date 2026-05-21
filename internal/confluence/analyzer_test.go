package confluence

import "testing"

func TestAnalyzerExtractTextContent(t *testing.T) {
	a := NewAnalyzer(nil, nil)
	html := `<p>Hello <strong>world</strong> &amp; team</p><script>alert(1)</script>`
	got := a.extractTextContent(html)
	if got != "Hello world & team" {
		t.Fatalf("got %q", got)
	}
}

func TestAnalyzerNormalizeWhitespace(t *testing.T) {
	a := NewAnalyzer(nil, nil)
	got := a.normalizeWhitespace("  foo   bar\n\n\n\nbaz  ")
	if got != " foo bar baz " {
		t.Fatalf("got %q", got)
	}
}
