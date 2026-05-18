package collabworktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestValidateGitRepo(t *testing.T) {
	tmp := t.TempDir()
	if err := ValidateGitRepo(tmp); err == nil {
		t.Fatal("expected error for non-git dir")
	}
	initGitRepo(t, tmp)
	if err := ValidateGitRepo(tmp); err != nil {
		t.Fatalf("ValidateGitRepo: %v", err)
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)

	assets := filepath.Join(t.TempDir(), "collab-assets")
	collabID := "test-collab-uuid-1234"

	res, err := Create(CreateOptions{
		SourceRepoPath: repo,
		AssetsRoot:     assets,
		CollabID:       collabID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.WorktreePath == "" || res.Branch == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if _, err := os.Stat(res.WorktreePath); err != nil {
		t.Fatalf("worktree missing: %v", err)
	}
	if res.Branch != DefaultBranchName(collabID) {
		t.Fatalf("branch = %q want %q", res.Branch, DefaultBranchName(collabID))
	}

	if err := Remove(RemoveOptions{
		WorktreePath:   res.WorktreePath,
		SourceRepoPath: repo,
		Branch:         res.Branch,
		DeleteBranch:   false,
	}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(res.WorktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree dir should be gone: %v", err)
	}
}

func TestDefaultBranchName(t *testing.T) {
	if got := DefaultBranchName("abcdef12-0000"); got != "nj/collab-abcdef12" {
		t.Fatalf("got %q", got)
	}
}
