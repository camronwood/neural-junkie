// Package collabworktree creates and removes git worktrees for collaboration execution.
package collabworktree

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitTimeout = 2 * time.Minute

// CreateOptions configures git worktree creation for a collaboration session.
type CreateOptions struct {
	SourceRepoPath string // absolute path to existing git checkout
	AssetsRoot     string // parent dir; worktree at AssetsRoot/worktrees/<CollabID>/
	CollabID       string
	Branch         string // if empty, derived from CollabID
}

// CreateResult holds paths created by git worktree add.
type CreateResult struct {
	WorktreePath string
	Branch       string
}

// DefaultBranchName returns the collaboration branch name for an id.
func DefaultBranchName(collabID string) string {
	short := collabID
	if len(short) > 8 {
		short = short[:8]
	}
	return "nj/collab-" + short
}

// PlannedWorktreePath returns the absolute worktree directory without creating it.
func PlannedWorktreePath(assetsRoot, collabID string) string {
	return filepath.Join(assetsRoot, "worktrees", collabID)
}

// ValidateGitRepo reports whether path is a git repository root.
func ValidateGitRepo(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("repository path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("repository path: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", abs, "rev-parse", "--git-dir")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("not a git repository at %s: %s", abs, msg)
	}
	return nil
}

// Create adds a new worktree and branch from SourceRepoPath.
func Create(opts CreateOptions) (*CreateResult, error) {
	source := strings.TrimSpace(opts.SourceRepoPath)
	if source == "" {
		return nil, fmt.Errorf("source repository path is required")
	}
	if err := ValidateGitRepo(source); err != nil {
		return nil, err
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("source repository path: %w", err)
	}
	collabID := strings.TrimSpace(opts.CollabID)
	if collabID == "" {
		return nil, fmt.Errorf("collaboration id is required")
	}
	assetsRoot := strings.TrimSpace(opts.AssetsRoot)
	if assetsRoot == "" {
		return nil, fmt.Errorf("assets root is required")
	}
	branch := strings.TrimSpace(opts.Branch)
	if branch == "" {
		branch = DefaultBranchName(collabID)
	}
	worktreePath := PlannedWorktreePath(assetsRoot, collabID)
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, fmt.Errorf("worktree parent directory: %w", err)
	}
	if _, err := os.Stat(worktreePath); err == nil {
		return nil, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	branch = uniqueBranchName(sourceAbs, branch)

	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", sourceAbs, "worktree", "add", "-b", branch, worktreePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git worktree add: %s", msg)
	}

	absWork, err := filepath.Abs(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("worktree path: %w", err)
	}
	return &CreateResult{WorktreePath: absWork, Branch: branch}, nil
}

func uniqueBranchName(repoPath, branch string) string {
	if !branchExists(repoPath, branch) {
		return branch
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", branch, i)
		if !branchExists(repoPath, candidate) {
			return candidate
		}
	}
	return branch + "-" + fmt.Sprintf("%d", time.Now().Unix())
}

func branchExists(repoPath, branch string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

// RemoveOptions configures worktree teardown.
type RemoveOptions struct {
	WorktreePath string
	SourceRepoPath string // repo that owns the worktree; required for git worktree remove
	Branch         string
	DeleteBranch   bool
}

// Remove deletes a collaboration worktree. Failures are returned for caller logging.
func Remove(opts RemoveOptions) error {
	worktree := strings.TrimSpace(opts.WorktreePath)
	if worktree == "" {
		return nil
	}
	repo := strings.TrimSpace(opts.SourceRepoPath)
	var errs []string

	if repo != "" {
		if err := ValidateGitRepo(repo); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, "git", "-C", repo, "worktree", "remove", "--force", worktree)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs = append(errs, fmt.Sprintf("worktree remove: %s", strings.TrimSpace(string(out))))
			}
		}
	}

	if _, err := os.Stat(worktree); err == nil {
		if err := os.RemoveAll(worktree); err != nil {
			errs = append(errs, fmt.Sprintf("remove directory: %v", err))
		}
	}

	branch := strings.TrimSpace(opts.Branch)
	if opts.DeleteBranch && branch != "" && repo != "" {
		ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "-C", repo, "branch", "-D", branch)
		if out, err := cmd.CombinedOutput(); err != nil {
			// branch may already be merged/deleted
			if !bytes.Contains(out, []byte("not found")) {
				errs = append(errs, fmt.Sprintf("branch delete: %s", strings.TrimSpace(string(out))))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
