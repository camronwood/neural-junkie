package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectRulesMarkdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Use tabs in Go."), 0o644); err != nil {
		t.Fatal(err)
	}
	out := LoadProjectRulesMarkdown(dir)
	if out == "" || !strings.Contains(out, "Use tabs") {
		t.Fatalf("expected AGENTS.md content, got %q", out)
	}
}
