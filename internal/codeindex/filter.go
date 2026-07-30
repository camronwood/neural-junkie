package codeindex

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/repo"
)

// indexableExtensions is the allowlist for semantic/codeindex builds.
// Aligned with LanguageExtensions + backend isSourceFile (.toml).
var indexableExtensions = map[string]struct{}{
	".go": {}, ".py": {}, ".js": {}, ".ts": {}, ".tsx": {}, ".jsx": {},
	".java": {}, ".rs": {}, ".c": {}, ".cpp": {}, ".cs": {}, ".rb": {},
	".php": {}, ".sh": {}, ".bash": {}, ".yml": {}, ".yaml": {},
	".json": {}, ".xml": {}, ".md": {}, ".sql": {}, ".toml": {},
}

const binarySniffBytes = 8 * 1024

// IsIndexableRelPath reports whether a repo-relative path may enter the codeindex.
func IsIndexableRelPath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.HasSuffix(rel, "/") {
		return false
	}
	name := filepath.Base(rel)
	if repo.ShouldIgnoreEntry(filepath.FromSlash(rel), name) {
		return false
	}
	if repo.IsLockfileName(name) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	_, ok := indexableExtensions[ext]
	return ok
}

// IsReadableSourceFile applies path allowlist plus size and binary sniffing.
func IsReadableSourceFile(absPath, rel string) bool {
	if !IsIndexableRelPath(rel) {
		return false
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return false
	}
	if info.Size() > repo.MaxFileSize {
		return false
	}
	f, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, binarySniffBytes)
	n, _ := f.Read(buf)
	return !containsNUL(buf[:n])
}

// LooksLikeBinary reports whether content contains a NUL in the sniff window.
func LooksLikeBinary(data []byte) bool {
	n := len(data)
	if n > binarySniffBytes {
		n = binarySniffBytes
	}
	return containsNUL(data[:n])
}

func containsNUL(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

func filterIndexablePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if IsIndexableRelPath(p) {
			out = append(out, filepath.ToSlash(p))
		}
	}
	return out
}
