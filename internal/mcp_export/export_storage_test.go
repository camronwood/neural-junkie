package mcp_export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExportsBaseDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MCP_EXPORTS_DIR", dir)

	base, err := resolveExportsBaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if base != dir {
		t.Fatalf("expected %q, got %q", dir, base)
	}
}

func TestNewExportStorageTildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	sub := "nj-exports-test-" + t.Name()
	t.Setenv("MCP_EXPORTS_DIR", "~/"+sub)

	storage, err := NewExportStorage()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, sub)
	if storage.baseDir != expected {
		t.Fatalf("expected base %q, got %q", expected, storage.baseDir)
	}
	_ = os.RemoveAll(expected)
}
