package workspacesymbols

import (
	"context"
	"sort"
	"strings"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

// SearchIndexedViaBackend builds symbol index using workspace backend (remote or local).
func SearchIndexedViaBackend(ctx context.Context, b workspacebackend.Backend, q, kind string, limit int) ([]Symbol, error) {
	if b == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	paths, err := workspacebackend.ListFilesRecursive(ctx, b, ".", 8000)
	if err != nil {
		return nil, err
	}
	var all []Symbol
	for _, rel := range paths {
		if ctx.Err() != nil {
			break
		}
		lang := langForPath(rel)
		if lang == "" {
			continue
		}
		data, err := b.ReadFile(ctx, rel)
		if err != nil {
			continue
		}
		syms, err := scanFileContent(rel, lang, string(data))
		if err != nil {
			continue
		}
		all = append(all, syms...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		return all[i].Line < all[j].Line
	})
	q = strings.ToLower(strings.TrimSpace(q))
	kind = strings.ToLower(strings.TrimSpace(kind))
	var out []Symbol
	for _, s := range all {
		if kind != "" && strings.ToLower(s.Kind) != kind {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(s.Name), q) {
			out = append(out, s)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// scanFileContent parses symbols from in-memory file content.
func scanFileContent(rel, lang, content string) ([]Symbol, error) {
	return scanContent(rel, lang, content)
}
