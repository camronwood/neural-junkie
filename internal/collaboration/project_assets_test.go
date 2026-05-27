package collaboration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectCollabDirectory(t *testing.T) {
	repo := t.TempDir()
	dir, err := ProjectCollabDirectory(repo, "abc-123")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(repo, ProjectCollabsDirName, "abc-123")
	if dir != want {
		t.Fatalf("dir=%q want %q", dir, want)
	}
}

func TestCollabAssetPathsUsesProjectDirWhenSourceRepoSet(t *testing.T) {
	repo := t.TempDir()
	c := &Collaboration{
		ID:             "id-1",
		SourceRepoPath: repo,
	}
	paths := CollabAssetPaths(c, t.TempDir())
	wantDir := filepath.Join(repo, ProjectCollabsDirName, "id-1")
	if paths.Directory != wantDir {
		t.Fatalf("directory=%q want %q", paths.Directory, wantDir)
	}
	if filepath.Base(paths.Plan) != ReviewAssetsPlanFileName {
		t.Fatalf("plan path=%q", paths.Plan)
	}
}

func TestEnsureProjectCollabDirCreatesDirectory(t *testing.T) {
	repo := t.TempDir()
	dir, err := EnsureProjectCollabDir(repo, "x")
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatal("expected directory")
	}
}
