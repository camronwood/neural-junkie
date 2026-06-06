package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectHasESLint_noConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"demo","devDependencies":{"vite":"^5.0.0"}}`)
	if ProjectHasESLint(dir) {
		t.Fatal("expected false without eslint config or dependency")
	}
}

func TestProjectHasESLint_configFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "eslint.config.js", "export default [];\n")
	if !ProjectHasESLint(dir) {
		t.Fatal("expected true when eslint.config.js exists")
	}
}

func TestProjectHasESLint_dependency(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"eslint":"^9.0.0"}}`)
	if !ProjectHasESLint(dir) {
		t.Fatal("expected true when eslint is in devDependencies")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
