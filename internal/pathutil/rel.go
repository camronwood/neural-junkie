package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRelWithinRoot resolves rel against root. When rel embeds redundant
// fixture/repo prefixes (e.g. scenarios/fixtures/foo/src/App.js inside foo/),
// the shortest suffix that exists under root wins.
func ResolveRelWithinRoot(root, rel string) (absClean string, err error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("invalid root path: %w", err)
	}
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	rel = filepath.FromSlash(rel)
	if rel == "" || rel == "." {
		return WithinRoot(rootAbs, rootAbs)
	}
	rel = stripEmbeddedRootPrefix(rootAbs, rel)
	if filepath.IsAbs(rel) {
		return WithinRoot(rootAbs, rel)
	}

	tryJoin := func(suffix string) (string, bool) {
		suffix = strings.TrimPrefix(strings.TrimSpace(suffix), "/")
		if suffix == "" || suffix == "." {
			return "", false
		}
		full := filepath.Join(rootAbs, suffix)
		if _, statErr := os.Stat(full); statErr != nil {
			return "", false
		}
		abs, err := WithinRoot(rootAbs, full)
		if err != nil {
			return "", false
		}
		return abs, true
	}

	if abs, ok := tryJoin(rel); ok {
		return abs, nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		if abs, ok := tryJoin(filepath.Join(parts[i:]...)); ok {
			return abs, nil
		}
	}

	return WithinRoot(rootAbs, filepath.Join(rootAbs, rel))
}

// stripEmbeddedRootPrefix removes a workspace root that was incorrectly baked
// into a relative path (e.g. Users/.../dickory-docs/Makefile when root is that repo).
func stripEmbeddedRootPrefix(rootAbs, rel string) string {
	rootAbs = filepath.Clean(rootAbs)
	rel = filepath.Clean(rel)
	sep := string(filepath.Separator)
	if filepath.IsAbs(rel) {
		if rel == rootAbs {
			return "."
		}
		prefix := rootAbs + sep
		if strings.HasPrefix(rel, prefix) {
			return strings.TrimPrefix(rel, prefix)
		}
		return rel
	}
	rootNoLead := strings.TrimPrefix(rootAbs, sep)
	if rel == rootNoLead {
		return "."
	}
	if strings.HasPrefix(rel, rootNoLead+sep) {
		return strings.TrimPrefix(rel, rootNoLead+sep)
	}
	return rel
}
