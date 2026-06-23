// Package graph provides optional symbol-level navigation (imports, references).
// v1 ships a lightweight stub; tree-sitter / gopls integration is planned for v2.
package graph

import (
	"context"
	"strings"
)

// Reference is a symbol reference site.
type Reference struct {
	Path      string
	Line      int
	Column    int
	Symbol    string
	Context   string
}

// FindReferences returns grep-style references until language servers are wired.
func FindReferences(ctx context.Context, repoPath, symbol string, limit int) ([]Reference, error) {
	_ = ctx
	symbol = strings.TrimSpace(symbol)
	if symbol == "" || limit <= 0 {
		return nil, nil
	}
	// Placeholder: callers should prefer codeintel.SemanticSearch + grep tools until LSP graph ships.
	return nil, nil
}
