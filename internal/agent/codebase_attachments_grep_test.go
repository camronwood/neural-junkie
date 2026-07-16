package agent

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGrepWorkspaceSymbol_minimalRepoWidget(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "scenarios", "fixtures", "minimal-repo")
	got := grepWorkspaceSymbol(root, "ComputeObscureWidget", 4)
	if len(got) == 0 {
		t.Fatal("expected widget.go hit")
	}
	found := false
	for _, r := range got {
		if strings.Contains(r.Content, "ComputeObscureWidget") {
			found = true
			t.Logf("hit %s", r.Path)
		}
	}
	if !found {
		t.Fatalf("missing symbol in %#v", got)
	}
}
