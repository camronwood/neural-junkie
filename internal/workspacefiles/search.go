// Package workspacefiles searches files under a workspace root.
package workspacefiles

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/camronwood/neural-junkie/internal/git"
	"github.com/camronwood/neural-junkie/internal/repo"
)

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true,
	"dist": true, "build": true, ".neural-junkie": true,
	repo.ScenarioBaselineDir: true,
}

const defaultSearchLimit = 50
const walkMaxFiles = 8000

// Search returns relative paths matching query (case-insensitive substring), up to limit.
func Search(ctx context.Context, workspaceRoot, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	query = strings.TrimSpace(strings.ToLower(query))
	root, err := filepath.Abs(filepath.Clean(workspaceRoot))
	if err != nil {
		return nil, err
	}

	var candidates []string
	if git.IsRepo(root) {
		candidates, err = gitLsFiles(ctx, root)
	} else {
		candidates, err = walkFiles(root)
	}
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, p := range candidates {
		if query == "" || strings.Contains(strings.ToLower(p), query) {
			matches = append(matches, p)
		}
	}
	sort.Strings(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func gitLsFiles(ctx context.Context, root string) ([]string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		slash := filepath.ToSlash(line)
		if repo.IsScenarioBaselinePath(slash) {
			continue
		}
		if repo.ShouldIgnoreEntry(filepath.FromSlash(slash), filepath.Base(slash)) {
			continue
		}
		lines = append(lines, slash)
	}
	return lines, nil
}

func walkFiles(root string) ([]string, error) {
	var out []string
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && ignoreDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		count++
		if count > walkMaxFiles {
			return fmt.Errorf("file limit exceeded")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if repo.ShouldIgnoreEntry(rel, d.Name()) {
			return nil
		}
		out = append(out, slash)
		return nil
	})
	if err != nil && !strings.Contains(err.Error(), "file limit") {
		return out, err
	}
	return out, nil
}
