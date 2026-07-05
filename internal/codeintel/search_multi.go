package codeintel

import (
	"context"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// RepoSearchHit is a search result tagged with its repository root.
type RepoSearchHit struct {
	Hit
	RepoPath string
	RepoName string
}

// SemanticSearchMulti searches multiple repo paths and merges ranked results.
func SemanticSearchMulti(ctx context.Context, paths []string, query string, limitPerRepo, totalLimit int) ([]RepoSearchHit, error) {
	if limitPerRepo <= 0 {
		limitPerRepo = 4
	}
	if totalLimit <= 0 {
		totalLimit = 12
	}
	var merged []RepoSearchHit
	for _, root := range paths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		part, err := SemanticSearch(ctx, root, query, limitPerRepo)
		if err != nil {
			continue
		}
		name := repoDisplayName(root)
		for _, h := range part {
			merged = append(merged, RepoSearchHit{Hit: h, RepoPath: root, RepoName: name})
		}
	}
	if len(merged) <= totalLimit {
		return merged, nil
	}
	return merged[:totalLimit], nil
}

// SemanticSearchMultiViaBackend searches multiple repos using optional backends keyed by path.
func SemanticSearchMultiViaBackend(
	ctx context.Context,
	paths []string,
	backendFor func(path string) workspacebackend.Backend,
	query string,
	limitPerRepo, totalLimit int,
) ([]RepoSearchHit, error) {
	if limitPerRepo <= 0 {
		limitPerRepo = 4
	}
	if totalLimit <= 0 {
		totalLimit = 12
	}
	type scored struct {
		hit   RepoSearchHit
		score float64
	}
	var merged []scored
	for i, root := range paths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		var part []Hit
		var err error
		if backendFor != nil {
			part, err = SemanticSearchViaBackend(ctx, root, backendFor(root), query, limitPerRepo)
		} else {
			part, err = SemanticSearch(ctx, root, query, limitPerRepo)
		}
		if err != nil {
			continue
		}
		name := repoDisplayName(root)
		for j, h := range part {
			merged = append(merged, scored{
				hit:   RepoSearchHit{Hit: h, RepoPath: root, RepoName: name},
				score: float64(len(paths)-i)*100 - float64(j),
			})
		}
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].score > merged[j].score })
	out := make([]RepoSearchHit, 0, totalLimit)
	for _, s := range merged {
		out = append(out, s.hit)
		if len(out) >= totalLimit {
			break
		}
	}
	return out, nil
}

func repoDisplayName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "repo"
	}
	path = strings.TrimRight(path, "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
