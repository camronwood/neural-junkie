package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func execGit(ctx context.Context, b workspacebackend.Backend, args ...string) (stdout, stderr string, err error) {
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    args,
		RelCwd:  ".",
		Timeout: defaultTimeout,
	})
	if err != nil {
		return "", res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("git %s: exit %d: %s", strings.Join(args, " "), res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
}

// StatusViaBackend runs git status using workspace backend exec (remote or local).
func StatusViaBackend(ctx context.Context, b workspacebackend.Backend) (*WorktreeStatus, error) {
	if b == nil {
		return nil, fmt.Errorf("backend required")
	}
	if b.Kind() == workspacebackend.KindLocal {
		return Status(ctx, b.Root())
	}
	branchOut, _, err := execGit(ctx, b, "branch", "--show-current")
	if err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(branchOut)
	porcelain, _, err := execGit(ctx, b, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	st := &WorktreeStatus{Branch: branch}
	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 4 {
			continue
		}
		code := line[:2]
		path := strings.TrimSpace(line[3:])
		switch {
		case code[0] != ' ' && code[0] != '?':
			st.Staged = append(st.Staged, path)
		case code[1] != ' ':
			st.Unstaged = append(st.Unstaged, path)
		case strings.HasPrefix(code, "??"):
			st.Untracked = append(st.Untracked, path)
		}
	}
	st.Clean = len(st.Staged) == 0 && len(st.Unstaged) == 0 && len(st.Untracked) == 0
	return st, nil
}

// DiffViaBackend returns unified diff for a path.
func DiffViaBackend(ctx context.Context, b workspacebackend.Backend, path string, staged bool) (string, error) {
	if b.Kind() == workspacebackend.KindLocal {
		return Diff(ctx, b.Root(), path, staged)
	}
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	out, _, err := execGit(ctx, b, args...)
	return out, err
}

// CommitViaBackend creates a commit, optionally staging paths first.
func CommitViaBackend(ctx context.Context, b workspacebackend.Backend, message string, paths []string) error {
	if b.Kind() == workspacebackend.KindLocal {
		return Commit(ctx, b.Root(), message, paths)
	}
	if len(paths) > 0 {
		if err := AddViaBackend(ctx, b, paths); err != nil {
			return err
		}
	}
	_, _, err := execGit(ctx, b, "commit", "-m", message)
	return err
}

// PushViaBackend pushes to remote.
func PushViaBackend(ctx context.Context, b workspacebackend.Backend) error {
	if b.Kind() == workspacebackend.KindLocal {
		return Push(ctx, b.Root())
	}
	_, _, err := execGit(ctx, b, "push")
	return err
}

// PullViaBackend pulls from remote.
func PullViaBackend(ctx context.Context, b workspacebackend.Backend) error {
	if b.Kind() == workspacebackend.KindLocal {
		return Pull(ctx, b.Root())
	}
	_, _, err := execGit(ctx, b, "pull")
	return err
}

// AddViaBackend stages paths.
func AddViaBackend(ctx context.Context, b workspacebackend.Backend, paths []string) error {
	if b.Kind() == workspacebackend.KindLocal {
		return Add(ctx, b.Root(), paths)
	}
	args := append([]string{"add"}, paths...)
	_, _, err := execGit(ctx, b, args...)
	return err
}

// ResetUnstageViaBackend unstages paths.
func ResetUnstageViaBackend(ctx context.Context, b workspacebackend.Backend, paths []string) error {
	if b.Kind() == workspacebackend.KindLocal {
		return ResetUnstage(ctx, b.Root(), paths)
	}
	args := append([]string{"reset", "HEAD"}, paths...)
	_, _, err := execGit(ctx, b, args...)
	return err
}
