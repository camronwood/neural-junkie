// Package codeintel is the unified search API for repo agent Q&A and agent workspace tools.
package codeintel

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/codeindex"
	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// Hit is a semantic or structural search result.
type Hit = codeindex.SearchResult

// SemanticSearch runs embedding search over a local repo path.
func SemanticSearch(ctx context.Context, repoPath, query string, limit int) ([]Hit, error) {
	return codeindex.Search(ctx, repoPath, query, limit)
}

// SemanticSearchViaBackend searches using a workspace backend (local or nj-remote).
func SemanticSearchViaBackend(ctx context.Context, root string, backend workspacebackend.Backend, query string, limit int) ([]Hit, error) {
	return codeindex.SearchViaBackend(ctx, root, backend, query, limit)
}
