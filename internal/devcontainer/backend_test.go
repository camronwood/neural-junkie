package devcontainer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/camronwood/neural-junkie/internal/workspacebackend"
)

func TestLoadViaBackendLocal(t *testing.T) {
	dir := t.TempDir()
	dcDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"test-dev","image":"mcr.microsoft.com/devcontainers/go","workspaceFolder":"/workspaces/app"}`
	if err := os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	local := workspacebackend.NewLocal(dir)
	got, err := LoadViaBackend(context.Background(), local)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "test-dev" {
		t.Fatalf("name=%q", got.Name)
	}
	if got.WorkspaceFolder != "/workspaces/app" {
		t.Fatalf("workspace folder=%q", got.WorkspaceFolder)
	}
}
