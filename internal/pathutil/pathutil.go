// Package pathutil provides filesystem path containment checks for workspace APIs.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WithinRoot returns an absolute, cleaned path only if candidate resolves inside root.
// It rejects prefix tricks like root "/tmp/ws" vs candidate "/tmp/ws_other/file".
// Symlink targets are resolved (EvalSymlinks) so a link inside root that points
// outside cannot escape on read/write. The returned path is expressed under the
// caller's rootAbs form (macOS /var vs /private/var) so workspace prefix checks stay stable.
func WithinRoot(root, candidate string) (absClean string, err error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("invalid root path: %w", err)
	}
	candAbs, err := filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("invalid candidate path: %w", err)
	}
	rootResolved, err := resolvePath(rootAbs)
	if err != nil {
		return "", fmt.Errorf("invalid root path: %w", err)
	}
	candResolved, err := resolvePath(candAbs)
	if err != nil {
		return "", fmt.Errorf("invalid candidate path: %w", err)
	}
	if err := AssertWithinRootAbs(rootResolved, candResolved); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootResolved, candResolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Contained but Rel failed; fall back to resolved path.
		return candResolved, nil
	}
	if rel == "." {
		return rootAbs, nil
	}
	return filepath.Join(rootAbs, rel), nil
}

// resolvePath returns a symlink-resolved absolute path. When the final component
// does not exist yet (common for create/write), existing ancestors are resolved
// and the missing suffix is re-joined so symlink-dir escapes are still caught.
func resolvePath(abs string) (string, error) {
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		// EvalSymlinks can return other errors for broken links; still refuse.
		if _, statErr := os.Lstat(abs); statErr == nil {
			return "", fmt.Errorf("symlink escape or unresolvable path: %w", err)
		}
	}
	cur := abs
	var suffix []string
	for {
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("cannot resolve path %q: %w", abs, err)
		}
		suffix = append([]string{filepath.Base(cur)}, suffix...)
		parentResolved, perr := filepath.EvalSymlinks(parent)
		if perr == nil {
			return filepath.Join(append([]string{parentResolved}, suffix...)...), nil
		}
		if _, statErr := os.Lstat(parent); statErr == nil && !os.IsNotExist(perr) {
			return "", fmt.Errorf("symlink escape or unresolvable path: %w", perr)
		}
		cur = parent
	}
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
