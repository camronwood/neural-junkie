package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var eslintConfigMarkers = []string{
	"eslint.config.js",
	"eslint.config.mjs",
	"eslint.config.cjs",
	".eslintrc.js",
	".eslintrc.cjs",
	".eslintrc.json",
	".eslintrc.yml",
	".eslintrc.yaml",
	".eslintrc",
}

// ProjectHasESLint reports whether the project has ESLint configured (not just npx-downloadable).
func ProjectHasESLint(projectRoot string) bool {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return false
	}
	for _, name := range eslintConfigMarkers {
		if PathExists(filepath.Join(projectRoot, name)) {
			return true
		}
	}
	pkgPath := filepath.Join(projectRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return false
	}
	for _, deps := range []map[string]string{pkg.Dependencies, pkg.DevDependencies} {
		for name := range deps {
			if isESLintPackageName(name) {
				return true
			}
		}
	}
	return false
}

func isESLintPackageName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "eslint" || strings.HasPrefix(name, "eslint-") || strings.HasPrefix(name, "@eslint/")
}

// ESLintNotConfiguredMessage explains why lint was skipped.
func ESLintNotConfiguredMessage(projectRoot string) string {
	return "ESLint is not configured in this project (" + projectRoot + "). " +
		"No eslint.config.* or .eslintrc.* file and eslint is not listed in package.json. " +
		"Use run_typescript_check (tsc --noEmit) or npm run build instead, or add ESLint to the project first."
}
