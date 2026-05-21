package slack

import (
	"os"
	"path/filepath"
	"testing"
)

// useTempHomeDir redirects ~/.neural-junkie/slack to a temp directory for the test.
func useTempHomeDir(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("HOME", base)
	dir := filepath.Join(base, ".neural-junkie", "slack")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}
