// Package pathutil provides filesystem path containment checks for workspace APIs.
package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WithinRoot returns an absolute, cleaned path only if candidate resolves inside root.
// It rejects prefix tricks like root "/tmp/ws" vs candidate "/tmp/ws_other/file".
func WithinRoot(root, candidate string) (absClean string, err error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("invalid root path: %w", err)
	}
	candAbs, err := filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("invalid candidate path: %w", err)
	}
	if err := AssertWithinRootAbs(rootAbs, candAbs); err != nil {
		return "", err
	}
	return candAbs, nil
}

// AssertWithinRootAbs checks containment using absolute paths (after filepath.Clean + Abs).
func AssertWithinRootAbs(rootAbs, candAbs string) error {
	if candAbs == rootAbs {
		return nil
	}
	sep := string(filepath.Separator)
	prefix := rootAbs + sep
	if strings.HasPrefix(candAbs, prefix) {
		return nil
	}
	return fmt.Errorf("path outside workspace root")
}
