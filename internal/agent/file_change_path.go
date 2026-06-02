package agent

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var validFileChangeRelPathRE = regexp.MustCompile(`^(?:[a-zA-Z0-9._\-]+/)*[a-zA-Z0-9][a-zA-Z0-9._\-]*\.[a-zA-Z0-9]{1,16}$`)

// isValidFileChangeRelPath rejects label-like garbage (e.g. "File:") from loose [FILE_CHANGE] parsing.
func isValidFileChangeRelPath(path string) bool {
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return false
	}
	if strings.Contains(path, "..") || strings.Contains(path, ":") {
		return false
	}
	base := filepath.Base(path)
	switch base {
	case "Dockerfile", "Makefile", "go.mod", "package.json":
		return true
	}
	if !strings.Contains(base, ".") {
		return false
	}
	stem := strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
	switch stem {
	case "", "file", "path", "operation", "new", "old", "create", "edit":
		return false
	}
	return validFileChangeRelPathRE.MatchString(path)
}

func normalizeFileChangeRelPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	path = strings.TrimSuffix(path, ":")
	return strings.TrimSpace(path)
}

// longestValidPathIn prefers the most specific user-mentioned path (e.g. tailwind.config.js over tailwind.config).
func longestValidPathIn(paths []string) string {
	best := ""
	for _, p := range paths {
		if !isValidFileChangeRelPath(p) {
			continue
		}
		p = normalizeFileChangeRelPath(p)
		if len(p) > len(best) {
			best = p
		}
	}
	return best
}

// preferImplementationTargetPath picks a sensible path when the model emitted a bad one.
func preferImplementationTargetPath(userContent, modelPath string, agentType protocol.AgentType) string {
	if isValidFileChangeRelPath(modelPath) {
		return normalizeFileChangeRelPath(modelPath)
	}
	if p := longestValidPathIn(DetectFilePaths(userContent)); p != "" {
		return p
	}
	for _, p := range implementationSeedCandidates(agentType, userContent) {
		if isValidFileChangeRelPath(p) {
			return p
		}
	}
	return ""
}

// resolveLooseFileChangePath returns the first valid relative path in a loose [FILE_CHANGE] tail.
func resolveLooseFileChangePath(tail string) string {
	if p := normalizeFileChangeRelPath(extractDirectiveField(tail, "path")); isValidFileChangeRelPath(p) {
		return p
	}
	if m := looseFileChangeSameLineRE.FindStringSubmatch(tail); len(m) >= 2 {
		if p := normalizeFileChangeRelPath(m[1]); isValidFileChangeRelPath(p) {
			return p
		}
	}
	for _, p := range DetectFilePaths(tail) {
		if isValidFileChangeRelPath(p) {
			return p
		}
	}
	return ""
}
