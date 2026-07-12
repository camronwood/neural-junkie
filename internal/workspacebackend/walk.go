package workspacebackend

import (
	"context"
	"path/filepath"
	"strings"
)

var walkIgnoreDirs = map[string]bool{
	".git": true, "node_modules": true, "target": true,
	"dist": true, "build": true, ".neural-junkie": true,
}

// IsWalkIgnoreDir reports whether a directory name should be skipped during workspace walks.
func IsWalkIgnoreDir(name string) bool {
	return walkIgnoreDirs[name]
}

// ListFilesRecursive returns relative file paths under rel (BFS), up to maxFiles.
func ListFilesRecursive(ctx context.Context, b Backend, rel string, maxFiles int) ([]string, error) {
	if maxFiles <= 0 {
		maxFiles = 8000
	}
	var out []string
	queue := []string{strings.TrimPrefix(rel, "/")}
	if queue[0] == "" {
		queue[0] = "."
	}
	for len(queue) > 0 && len(out) < maxFiles {
		dir := queue[0]
		queue = queue[1:]
		entries, err := b.ReadDir(ctx, dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir {
				if walkIgnoreDirs[e.Name] {
					continue
				}
				sub := e.Path
				if sub == "" {
					sub = filepath.Join(dir, e.Name)
				}
				queue = append(queue, sub)
				continue
			}
			p := e.Path
			if p == "" {
				p = filepath.Join(dir, e.Name)
			}
			p = filepath.ToSlash(strings.TrimPrefix(p, "./"))
			out = append(out, p)
			if len(out) >= maxFiles {
				break
			}
		}
	}
	return out, nil
}
