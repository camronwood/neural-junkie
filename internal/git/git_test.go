package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStatusCommitInTempRepo(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	st, err := Status(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Clean {
		t.Fatal("expected dirty tree")
	}
	if len(st.Untracked) == 0 && len(st.Unstaged) == 0 {
		t.Fatalf("expected untracked or unstaged a.txt, got %+v", st)
	}

	run("add", "a.txt")
	st2, err := Status(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st2.Staged) == 0 {
		t.Fatalf("expected staged file, got %+v", st2)
	}
	if err := Commit(ctx, dir, "init", nil); err != nil {
		t.Fatal(err)
	}
	st3, err := Status(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st3.Clean {
		t.Fatalf("expected clean after commit, got %+v", st3)
	}
}

func TestDiffWithinRoot(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "t@e.com")
	run("config", "user.name", "T")
	_ = os.WriteFile(filepath.Join(dir, "f.go"), []byte("package main\n"), 0o644)
	run("add", "f.go")
	run("commit", "-m", "add")
	_ = os.WriteFile(filepath.Join(dir, "f.go"), []byte("package main\n\n"), 0o644)

	diff, err := Diff(context.Background(), dir, "f.go", false)
	if err != nil {
		t.Fatal(err)
	}
	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
}

func TestAddAndReset(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "t@e.com")
	run("config", "user.name", "T")
	_ = os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644)
	ctx := context.Background()
	if err := Add(ctx, dir, []string{"x.txt"}); err != nil {
		t.Fatal(err)
	}
	st, _ := Status(ctx, dir)
	if len(st.Staged) != 1 {
		t.Fatalf("expected staged, got %+v", st)
	}
	if err := ResetUnstage(ctx, dir, []string{"x.txt"}); err != nil {
		t.Fatal(err)
	}
	st2, _ := Status(ctx, dir)
	if len(st2.Staged) != 0 {
		t.Fatalf("expected unstaged, got %+v", st2)
	}
}
