package agent

import "testing"

func TestLegacyFileChangeParseEnabled(t *testing.T) {
	t.Setenv("NEURAL_JUNKIE_LEGACY_FILE_CHANGE_PARSE", "")
	if legacyFileChangeParseEnabled() {
		t.Fatal("expected default off")
	}
	t.Setenv("NEURAL_JUNKIE_LEGACY_FILE_CHANGE_PARSE", "1")
	if !legacyFileChangeParseEnabled() {
		t.Fatal("expected on when env=1")
	}
}

func TestSanitizeAbsolutePathFileChangeFromResponse(t *testing.T) {
	in := "Done.\n[FILE_CHANGE path=\"/Users/test/proj/foo.md\"]\nMore"
	out := sanitizeAbsolutePathFileChangeFromResponse(in)
	if out == in {
		t.Fatal("expected absolute path block stripped")
	}
}
