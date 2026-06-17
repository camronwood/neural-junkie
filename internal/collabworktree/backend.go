package collabworktree

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

const remoteWorktreesDir = "collabs/worktrees"

// ValidateGitRepoViaBackend reports whether the remote workspace root is a git repo.
func ValidateGitRepoViaBackend(ctx context.Context, b workspacebackend.Backend) error {
	if b == nil {
		return fmt.Errorf("backend required")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    []string{"rev-parse", "--git-dir"},
		RelCwd:  ".",
	})
	if err != nil {
		msg := strings.TrimSpace(res.Stdout + res.Stderr)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("not a git repository: %s", msg)
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stdout + res.Stderr)
		return fmt.Errorf("not a git repository: %s", msg)
	}
	return nil
}

// CreateViaBackend adds a git worktree on a remote workspace via nj-remote exec.
func CreateViaBackend(ctx context.Context, b workspacebackend.Backend, opts CreateOptions) (*CreateResult, error) {
	if b == nil {
		return nil, fmt.Errorf("backend required")
	}
	collabID := strings.TrimSpace(opts.CollabID)
	if collabID == "" {
		return nil, fmt.Errorf("collaboration id is required")
	}
	if err := ValidateGitRepoViaBackend(ctx, b); err != nil {
		return nil, err
	}
	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		branch = DefaultBranchName(collabID)
	}
	worktreeRel := filepath.ToSlash(filepath.Join(remoteWorktreesDir, collabID))
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    []string{"worktree", "add", "-b", branch, worktreeRel},
		RelCwd:  ".",
	})
	if err != nil || res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stdout + res.Stderr)
		if msg == "" && err != nil {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git worktree add: %s", msg)
	}
	logical := filepath.Join(b.Root(), worktreeRel)
	return &CreateResult{WorktreePath: logical, Branch: branch}, nil
}

// RemoveViaBackend tears down a remote collaboration worktree.
func RemoveViaBackend(ctx context.Context, b workspacebackend.Backend, opts RemoveOptions) error {
	if b == nil {
		return nil
	}
	worktree := strings.TrimSpace(opts.WorktreePath)
	if worktree == "" {
		return nil
	}
	root := filepath.Clean(b.Root())
	rel := strings.TrimPrefix(filepath.Clean(worktree), root)
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	if rel == "" || rel == "." {
		rel = filepath.ToSlash(filepath.Join(remoteWorktreesDir, filepath.Base(worktree)))
	}
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	res, err := b.Exec(ctx, workspacebackend.ExecRequest{
		Command: "git",
		Args:    []string{"worktree", "remove", "--force", rel},
		RelCwd:  ".",
	})
	if err != nil || res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stdout + res.Stderr)
		if msg != "" && !strings.Contains(strings.ToLower(msg), "not a working tree") {
			return fmt.Errorf("worktree remove: %s", msg)
		}
	}
	branch := strings.TrimSpace(opts.Branch)
	if opts.DeleteBranch && branch != "" {
		res, err := b.Exec(ctx, workspacebackend.ExecRequest{
			Command: "git",
			Args:    []string{"branch", "-D", branch},
			RelCwd:  ".",
		})
		if err != nil || res.ExitCode != 0 {
			msg := strings.TrimSpace(res.Stdout + res.Stderr)
			if msg != "" && !strings.Contains(strings.ToLower(msg), "not found") {
				return fmt.Errorf("branch delete: %s", msg)
			}
		}
	}
	return nil
}
