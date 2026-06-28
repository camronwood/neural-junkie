package agent

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// isTrustedAutoApplyPath reports application source paths safe for auto_apply_edits.
func isTrustedAutoApplyPath(path string) bool {
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(path))
	if strings.HasSuffix(lower, "_test.go") {
		return true
	}
	parts := strings.Split(lower, "/")
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case "src", "internal", "pkg", "lib", "cmd", "app", "apps", "components", "pages", "tests", "test", "core":
		return true
	}
	if len(parts) >= 2 && parts[0] == "desktop" && parts[1] == "src" {
		return true
	}
	if strings.HasSuffix(lower, ".rs") && (parts[0] == "src" || parts[0] == "benches") {
		return true
	}
	if strings.HasSuffix(lower, ".py") && !isProtectedConfigPath(path) {
		base := filepath.Base(lower)
		if base != "__init__.py" && !strings.HasPrefix(base, "setup") {
			return true
		}
	}
	return false
}

// IsProtectedConfigPath reports manifest and build config paths that require human approval.
func IsProtectedConfigPath(path string) bool {
	return isProtectedConfigPath(path)
}

func isProtectedConfigPath(path string) bool {
	path = normalizeFileChangeRelPath(path)
	if path == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "package.json", "package-lock.json", "go.mod", "go.sum", "cargo.toml", "cargo.lock",
		"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt", "makefile",
		"dockerfile", "docker-compose.yml", "docker-compose.yaml":
		return true
	}
	lower := strings.ToLower(filepath.ToSlash(path))
	if strings.Contains(lower, "tauri.conf.json") {
		return true
	}
	if strings.HasPrefix(base, "vite.config.") || strings.HasPrefix(base, "vitest.config.") {
		return true
	}
	if strings.HasPrefix(base, "webpack.config.") || strings.HasPrefix(base, "rollup.config.") {
		return true
	}
	if strings.HasPrefix(lower, ".github/") {
		return true
	}
	return false
}

// ShouldAutoApproveFileChange gates hub auto-approve for protected config paths and non-source paths.
// workspaceRoot relativizes absolute change paths before policy checks (hub passes absolute paths).
func ShouldAutoApproveFileChange(path string, workspaceRoot ...string) bool {
	ws := ""
	if len(workspaceRoot) > 0 {
		ws = workspaceRoot[0]
	}
	path = RelativizeFileChangePath(path, ws)
	if path == "" {
		return false
	}
	if isProtectedConfigPath(path) && !isRootMakefile(path) {
		return false
	}
	return isTrustedAutoApplyPath(path) || isRootMakefile(path)
}

func isRootMakefile(path string) bool {
	parts := strings.Split(filepath.ToSlash(strings.TrimSpace(path)), "/")
	if len(parts) != 1 {
		return false
	}
	return strings.EqualFold(parts[0], "makefile")
}

func validateConfigJSONContent(path, content string) error {
	path = strings.ToLower(normalizeFileChangeRelPath(path))
	if !strings.HasSuffix(path, ".json") {
		return nil
	}
	trim := strings.TrimSpace(content)
	if trim == "" || trim == "null" {
		return errInvalidConfigJSON
	}
	if !json.Valid([]byte(content)) {
		return errInvalidConfigJSON
	}
	return nil
}

var errInvalidConfigJSON = &configJSONError{}

type configJSONError struct{}

func (e *configJSONError) Error() string {
	return "config JSON content must be valid and not empty/null"
}
