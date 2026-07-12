package agent

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

var validFileChangeRelPathRE = regexp.MustCompile(`^(?:[a-zA-Z0-9._\-]+/)*[a-zA-Z0-9][a-zA-Z0-9._\-]*\.[a-zA-Z0-9]{1,16}$`)

var atFilePathRE = regexp.MustCompile(`(?i)@file:([^\s]+)`)

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
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return ""
	}
	return path
}

// RelativizeFileChangePath maps absolute paths under workspaceRoot to repo-relative form.
func RelativizeFileChangePath(path, workspaceRoot string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	if path == "" {
		return ""
	}
	ws := strings.TrimSpace(workspaceRoot)
	if ws != "" && (filepath.IsAbs(path) || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\")) {
		wsClean := filepath.Clean(ws)
		abs := filepath.Clean(path)
		if rel, err := filepath.Rel(wsClean, abs); err == nil {
			rel = filepath.ToSlash(rel)
			if rel != ".." && !strings.HasPrefix(rel, "../") {
				return normalizeFileChangeRelPath(rel)
			}
		}
		return ""
	}
	return normalizeFileChangeRelPath(path)
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

// preferImplementationTargetPathForMessage resolves the edit target, including continuation turns.
func preferImplementationTargetPathForMessage(a *Agent, msg *protocol.Message) string {
	if a == nil || msg == nil {
		return ""
	}
	content := msg.Content
	if userAffirmsPendingImplementation(content) {
		for i := len(a.channelHistory(msg.Channel)) - 1; i >= 0; i-- {
			m := a.channelHistory(msg.Channel)[i]
			if m == nil || m.ID == msg.ID {
				continue
			}
			if protocol.IsUserLikeSender(m.From) && userRequestsImplementation(m.Content) {
				if p := preferImplementationTargetPath(a.resolveWorkspacePath(msg), m.Content, ""); p != "" {
					return p
				}
			}
		}
		return ""
	}
	return preferImplementationTargetPath(a.resolveWorkspacePath(msg), content, "")
}

// DetectAtFilePaths extracts explicit @file:path scoped edit targets from user text.
func DetectAtFilePaths(content string) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, m := range atFilePathRE.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		p := normalizeFileChangeRelPath(m[1])
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		paths = append(paths, p)
	}
	return paths
}

// preferImplementationTargetPath picks a sensible path when the model emitted a bad one.
func preferImplementationTargetPath(workspacePath, userContent, modelPath string) string {
	if isValidFileChangeRelPath(modelPath) {
		return normalizeFileChangeRelPath(modelPath)
	}
	if p := longestValidPathIn(DetectAtFilePaths(userContent)); p != "" {
		return p
	}
	if p := longestValidPathIn(DetectFilePaths(userContent)); p != "" {
		return p
	}
	if userAffirmsPendingImplementation(userContent) {
		return ""
	}
	for _, p := range implementationSeedCandidates(workspacePath, userContent, nil, nil, nil, false) {
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
