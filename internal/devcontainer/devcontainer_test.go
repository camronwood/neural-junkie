package devcontainer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDevcontainer(t *testing.T) {
	dir := t.TempDir()
	dcDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"name":"my-dev","image":"mcr.microsoft.com/devcontainers/go","workspaceFolder":"/workspaces/app"}`
	if err := os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Image == "" || got.WorkspaceFolder != "/workspaces/app" {
		t.Fatalf("cfg: %+v", got)
	}
	plan := PlanFromConfig(dir, got)
	if plan.SidecarPort != 19876 {
		t.Fatalf("plan: %+v", plan)
	}
}
