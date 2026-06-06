package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectHasTypeScript_localBin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"typescript":"^5.0.0"}}`)
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, binDir, "tsc", "#!/bin/sh\n")
	if !ProjectHasTypeScript(dir) {
		t.Fatal("expected true when node_modules/.bin/tsc exists")
	}
}

func TestProjectHasTypeScript_dependencyOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"typescript":"^5.0.0"}}`)
	if !ProjectHasTypeScript(dir) {
		t.Fatal("expected true when typescript is in devDependencies")
	}
}

func TestProjectHasTypeScript_missing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"vite":"^5.0.0"}}`)
	if ProjectHasTypeScript(dir) {
		t.Fatal("expected false without typescript")
	}
}

func TestTypeScriptCheckShellCommand_prefersLocalBin(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, binDir, "tsc", "#!/bin/sh\n")
	got := TypeScriptCheckShellCommand(dir)
	if got != "./node_modules/.bin/tsc --noEmit" {
		t.Fatalf("got %q", got)
	}
}

func TestTypeScriptCheckShellCommand_npmExecFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"typescript":"^5.0.0"}}`)
	got := TypeScriptCheckShellCommand(dir)
	if got != "npm exec -- tsc --noEmit" {
		t.Fatalf("got %q", got)
	}
}
