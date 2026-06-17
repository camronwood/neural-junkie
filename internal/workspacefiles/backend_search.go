package workspacefiles

import (
	"context"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// SearchBackend finds files via workspace backend (local or remote).
func SearchBackend(ctx context.Context, b workspacebackend.Backend, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	query = strings.TrimSpace(strings.ToLower(query))
	var candidates []string
	var err error
	if b.Kind() == workspacebackend.KindLocal {
		return Search(ctx, b.Root(), query, limit)
	}
	candidates, err = workspacebackend.ListFilesRecursive(ctx, b, ".", walkMaxFiles)
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
