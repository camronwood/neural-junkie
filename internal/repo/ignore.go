package repo

import (
	"path/filepath"
	"strings"
)

// rootBuildOutputNames are compiled binaries written to the repo root by `make`
// (see .gitignore). They are not source and should not count toward index size.
var rootBuildOutputNames = map[string]struct{}{
	"server":              {},
	"agent":               {},
	"cli":                 {},
	"gui":                 {},
	"tool-approval-hook":  {},
}

// nonSourceExtensions are skipped during repository walks (marketing media, etc.).
var nonSourceExtensions = map[string]struct{}{
	".mp4":  {},
	".mov":  {},
	".webm": {},
	".mkv":  {},
	".icns": {},
}

// ScenarioBaselineDir is the harness seed tree copied onto fixture roots for reset.
// Agents must not list, read, or edit it — only the live workspace files matter.
const ScenarioBaselineDir = ".scenario-baseline"

// IsScenarioBaselinePath reports whether relPath is under a harness seed directory.
func IsScenarioBaselinePath(relPath string) bool {
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if part == ScenarioBaselineDir {
			return true
		}
	}
	return false
}

// ShouldIgnoreEntry reports whether a path should be excluded from indexing walks.
// relPath uses OS separators and is relative to the repository root.
func ShouldIgnoreEntry(relPath, name string) bool {
	if ShouldIgnore(name) {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if part != "" && ShouldIgnore(part) {
			return true
		}
	}
	if isRootBuildOutput(relPath, name) {
		return true
	}
	slashPath := filepath.ToSlash(relPath)
	if strings.HasPrefix(slashPath, "docs/media") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := nonSourceExtensions[ext]; ok {
		return true
	}
	return false
}

func isRootBuildOutput(relPath, name string) bool {
	if _, ok := rootBuildOutputNames[name]; !ok {
		return false
	}
	// Only ignore when the binary lives at repo root (not e.g. cmd/server/main.go).
	return relPath == name || relPath == "."+string(filepath.Separator)+name
}
