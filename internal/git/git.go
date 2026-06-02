// Package git runs git CLI operations inside a repository root with path validation.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/pathutil"
)

const defaultTimeout = 60 * time.Second

// WorktreeStatus summarizes a working tree.
type WorktreeStatus struct {
	Branch    string   `json:"branch"`
	Clean     bool     `json:"clean"`
	Staged    []string `json:"staged"`
	Unstaged  []string `json:"unstaged"`
	Untracked []string `json:"untracked"`
}

// RevParseHEAD returns the current git HEAD commit hash, or empty if unavailable.
func RevParseHEAD(repoRoot string) (string, error) {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return "", err
	}
	if !IsRepo(repoRoot) {
		return "", fmt.Errorf("not a git repository")
	}
	out, stderr, err := runGit(context.Background(), repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %s: %w", strings.TrimSpace(stderr), err)
	}
	return strings.TrimSpace(out), nil
}

// IsRepo reports whether dir contains a .git directory or file.
func IsRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func runGit(ctx context.Context, dir string, args ...string) (string, string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Status returns branch and porcelain file lists relative to repo root.
func Status(ctx context.Context, repoRoot string) (*WorktreeStatus, error) {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return nil, err
	}
	if !IsRepo(repoRoot) {
		return nil, fmt.Errorf("not a git repository")
	}
	branchOut, _, err := runGit(ctx, repoRoot, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("git branch: %w", err)
	}
	branch := strings.TrimSpace(branchOut)

	porcelain, _, err := runGit(ctx, repoRoot, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	st := &WorktreeStatus{
		Branch:    branch,
		Clean:     true,
		Staged:    []string{},
		Unstaged:  []string{},
		Untracked: []string{},
	}
	for _, line := range strings.Split(porcelain, "\n") {
		if line == "" {
			continue
		}
		st.Clean = false
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])
		if x == '?' && y == '?' {
			st.Untracked = append(st.Untracked, path)
			continue
		}
		if x != ' ' && x != '?' {
			st.Staged = append(st.Staged, path)
		}
		if y != ' ' && y != '?' {
			st.Unstaged = append(st.Unstaged, path)
		}
	}
	return st, nil
}

// Add stages paths (git add --).
func Add(ctx context.Context, repoRoot string, paths []string) error {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		_, stderr, err := runGit(ctx, repoRoot, "add", "-A")
		if err != nil {
			return fmt.Errorf("git add: %s: %w", strings.TrimSpace(stderr), err)
		}
		return nil
	}
	for _, p := range paths {
		if _, err := validateRelPath(repoRoot, p); err != nil {
			return err
		}
		if _, _, err := runGit(ctx, repoRoot, "add", "--", p); err != nil {
			return fmt.Errorf("git add %s: %w", p, err)
		}
	}
	return nil
}

// ResetUnstage unstages paths (git reset HEAD --).
func ResetUnstage(ctx context.Context, repoRoot string, paths []string) error {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return err
	}
	args := []string{"reset", "HEAD"}
	if len(paths) > 0 {
		args = append(args, "--")
		for _, p := range paths {
			if _, err := validateRelPath(repoRoot, p); err != nil {
				return err
			}
			args = append(args, p)
		}
	}
	_, stderr, err := runGit(ctx, repoRoot, args...)
	if err != nil {
		return fmt.Errorf("git reset: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}

func validateRelPath(repoRoot, relPath string) (string, error) {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return "", err
	}
	full := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
	if _, err := pathutil.WithinRoot(repoRoot, full); err != nil {
		return "", err
	}
	return relPath, nil
}

// Diff returns unified diff for a path under repoRoot. staged uses --cached.
func Diff(ctx context.Context, repoRoot, relPath string, staged bool) (string, error) {
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return "", err
	}
	if _, err := validateRelPath(repoRoot, relPath); err != nil {
		return "", err
	}
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", relPath)
	out, stderr, err := runGit(ctx, repoRoot, args...)
	if err != nil {
		if strings.TrimSpace(out) != "" {
			return out, nil
		}
		return "", fmt.Errorf("git diff: %s: %w", strings.TrimSpace(stderr), err)
	}
	return out, nil
}

// Commit creates a commit with the given message; if paths empty, commits all staged changes.
func Commit(ctx context.Context, repoRoot, message string, paths []string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("commit message required")
	}
	repoRoot, err := filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return err
	}
	if len(paths) > 0 {
		for _, p := range paths {
			full := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimPrefix(p, "/")))
			if _, err := pathutil.WithinRoot(repoRoot, full); err != nil {
				return err
			}
			if _, _, err := runGit(ctx, repoRoot, "add", "--", p); err != nil {
				return fmt.Errorf("git add %s: %w", p, err)
			}
		}
	}
	_, stderr, err := runGit(ctx, repoRoot, "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}

// FileSides returns before/after content for diff UI. staged compares HEAD vs index.
func FileSides(ctx context.Context, repoRoot, relPath string, staged bool) (original, modified string, err error) {
	if _, err := validateRelPath(repoRoot, relPath); err != nil {
		return "", "", err
	}
	repoRoot, err = filepath.Abs(filepath.Clean(repoRoot))
	if err != nil {
		return "", "", err
	}
	full := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
	modBytes, readErr := os.ReadFile(full)
	if readErr != nil && !os.IsNotExist(readErr) {
		return "", "", readErr
	}
	working := string(modBytes)

	headOut, _, _ := runGit(ctx, repoRoot, "show", "HEAD:"+relPath)
	head := headOut

	if staged {
		idxOut, _, idxErr := runGit(ctx, repoRoot, "show", ":"+relPath)
		if idxErr != nil {
			return head, working, nil
		}
		return head, idxOut, nil
	}
	return head, working, nil
}

// Push runs git push in the repo.
func Push(ctx context.Context, repoRoot string) error {
	_, stderr, err := runGit(ctx, repoRoot, "push")
	if err != nil {
		return fmt.Errorf("git push: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}

// Pull runs git pull in the repo.
func Pull(ctx context.Context, repoRoot string) error {
	_, stderr, err := runGit(ctx, repoRoot, "pull")
	if err != nil {
		return fmt.Errorf("git pull: %s: %w", strings.TrimSpace(stderr), err)
	}
	return nil
}
