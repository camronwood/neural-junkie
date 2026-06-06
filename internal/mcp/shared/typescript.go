package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalTscBin returns the path to the project's tsc binary, or "" if missing.
func LocalTscBin(projectRoot string) string {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return ""
	}
	bin := filepath.Join(projectRoot, "node_modules", ".bin", "tsc")
	if PathExists(bin) {
		return bin
	}
	return ""
}

// ProjectHasTypeScript reports whether the project can run a local TypeScript check.
func ProjectHasTypeScript(projectRoot string) bool {
	if LocalTscBin(projectRoot) != "" {
		return true
	}
	return projectHasTypeScriptDep(projectRoot)
}

func projectHasTypeScriptDep(projectRoot string) bool {
	data, err := os.ReadFile(filepath.Join(projectRoot, "package.json"))
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
		if _, ok := deps["typescript"]; ok {
			return true
		}
	}
	return false
}

// TypeScriptNotConfiguredMessage explains why the TypeScript check was skipped.
func TypeScriptNotConfiguredMessage(projectRoot string) string {
	return "TypeScript is not available in this project (" + projectRoot + "). " +
		"Add typescript to devDependencies and run npm install, or use npm run build if it runs tsc."
}

// TypeScriptCheckShellCommand returns a shell command for tsc --noEmit, or "" if unavailable.
func TypeScriptCheckShellCommand(projectRoot string) string {
	if LocalTscBin(projectRoot) != "" {
		return "./node_modules/.bin/tsc --noEmit"
	}
	if projectHasTypeScriptDep(projectRoot) {
		return "npm exec -- tsc --noEmit"
	}
	return ""
}

// RunTypeScriptCheck runs tsc --noEmit using the project's TypeScript compiler.
// Never uses bare "npx tsc", which resolves to the wrong npm package (tsc@2.0.4).
func RunTypeScriptCheck(ctx context.Context, projectRoot string) (string, error) {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return "", fmt.Errorf("project path is required")
	}
	if bin := LocalTscBin(projectRoot); bin != "" {
		return RunCommand(ctx, projectRoot, bin, "--noEmit")
	}
	if projectHasTypeScriptDep(projectRoot) {
		return RunCommand(ctx, projectRoot, "npm", "exec", "--", "tsc", "--noEmit")
	}
	return TypeScriptNotConfiguredMessage(projectRoot), nil
}
