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

// nonSourceExtensions are skipped during repository walks (media, binaries, archives).
var nonSourceExtensions = map[string]struct{}{
	".mp4": {}, ".mov": {}, ".webm": {}, ".mkv": {}, ".avi": {}, ".icns": {},
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".ico": {},
	".svg": {}, ".bmp": {}, ".tiff": {},
	".zip": {}, ".tar": {}, ".gz": {}, ".bz2": {}, ".xz": {}, ".rar": {},
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".bin": {},
	".pdf": {}, ".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {},
	".mp3": {}, ".wav": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".eot": {},
	".o": {}, ".a": {}, ".pyc": {}, ".class": {},
	".gguf": {}, ".safetensors": {}, ".pb": {}, ".onnx": {},
}

// lockfileNames are dependency lock manifests excluded from source indexes.
var lockfileNames = map[string]struct{}{
	"package-lock.json": {},
	"yarn.lock":         {},
	"pnpm-lock.yaml":    {},
	"cargo.lock":        {},
	"go.sum":            {},
	"composer.lock":     {},
	"gemfile.lock":      {},
	"poetry.lock":       {},
}

// IsLockfileName reports whether basename is a dependency lockfile.
func IsLockfileName(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	if _, ok := lockfileNames[base]; ok {
		return true
	}
	return strings.HasSuffix(base, ".lock")
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
	if IsLockfileName(name) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := nonSourceExtensions[ext]; ok {
		return true
	}
	return false
}

// IsDependencyPath reports whether path (relative or absolute) is third-party
// or generated (venv, site-packages, node_modules, …).
func IsDependencyPath(path string) bool {
	slash := filepath.ToSlash(strings.TrimSpace(path))
	if slash == "" {
		return false
	}
	return ShouldIgnoreEntry(filepath.FromSlash(slash), filepath.Base(slash))
}

func isRootBuildOutput(relPath, name string) bool {
	if _, ok := rootBuildOutputNames[name]; !ok {
		return false
	}
	// Only ignore when the binary lives at repo root (not e.g. cmd/server/main.go).
	return relPath == name || relPath == "."+string(filepath.Separator)+name
}
