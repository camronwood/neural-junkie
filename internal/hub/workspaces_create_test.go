package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddWorkspaceCreate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	wm, err := NewWorkspaceManager()
	if err != nil {
		t.Fatal(err)
	}

	ws, err := wm.AddWorkspace("Phoenix Run", "", AddWorkspaceOptions{Create: true})
	if err != nil {
		t.Fatal(err)
	}
	if ws.Path == "" {
		t.Fatal("empty path")
	}
	info, err := os.Stat(ws.Path)
	if err != nil || !info.IsDir() {
		t.Fatalf("workspace dir missing: %v", err)
	}
	want := filepath.Join(home, ".neural-junkie", "workspaces", "Phoenix-Run")
	if ws.Path != want {
		t.Fatalf("path: got %q want %q", ws.Path, want)
	}
}

func TestAddWorkspaceCreateUnderParent(t *testing.T) {
	parent := t.TempDir()
	wm, err := NewWorkspaceManager()
	if err != nil {
		t.Fatal(err)
	}
	ws, err := wm.AddWorkspace("Lab Data", "", AddWorkspaceOptions{Create: true, ParentPath: parent})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(parent, "Lab-Data")
	if ws.Path != want {
		t.Fatalf("got %q want %q", ws.Path, want)
	}
}

func TestAddWorkspaceLinkRequiresPath(t *testing.T) {
	wm, err := NewWorkspaceManager()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wm.AddWorkspace("x", "", AddWorkspaceOptions{}); err == nil {
		t.Fatal("expected error without path")
	}
}

func TestSanitizeWorkspaceDirName(t *testing.T) {
	if got := sanitizeWorkspaceDirName("My Run 2026"); got != "My-Run-2026" {
		t.Fatalf("got %q", got)
	}
}
