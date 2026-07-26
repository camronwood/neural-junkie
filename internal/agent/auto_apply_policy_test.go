package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProtectedConfigPath(t *testing.T) {
	protected := []string{
		"package.json",
		"Makefile",
		"src-tauri/tauri.conf.json",
		"vite.config.ts",
		"go.mod",
	}
	for _, p := range protected {
		if !IsProtectedConfigPath(p) {
			t.Fatalf("expected protected: %s", p)
		}
	}
	if IsProtectedConfigPath("src/App.tsx") {
		t.Fatal("src/App.tsx should be trusted, not protected")
	}
}

func TestIsTrustedAutoApplyPath(t *testing.T) {
	trusted := []string{
		"src/App.tsx",
		"internal/agent/foo.go",
		"pkg/util/helper.go",
		"cmd/server/main.go",
		"core/sample/main.go",
		"desktop/src/App.tsx",
		"lib/foo.py",
	}
	for _, p := range trusted {
		if !isTrustedAutoApplyPath(p) {
			t.Fatalf("expected trusted: %s", p)
		}
	}
	if isTrustedAutoApplyPath("package.json") {
		t.Fatal("package.json should not be trusted auto-apply")
	}
}

func TestShouldAutoApproveFileChange_absoluteUnderWorkspace(t *testing.T) {
	ws := "/Users/dev/proj/scenarios/fixtures/minimal-repo"
	abs := ws + "/core/sample/main.go"
	if !ShouldAutoApproveFileChange(abs, ws) {
		t.Fatal("absolute path under workspace should auto-approve")
	}
	if ShouldAutoApproveFileChange(abs) {
		t.Fatal("absolute path without workspace root should not auto-approve")
	}
}

func TestShouldAutoApproveFileChange_rootMakefile(t *testing.T) {
	ws := "/tmp/fixture"
	if !ShouldAutoApproveFileChange("Makefile", ws) {
		t.Fatal("root Makefile should auto-approve in implementation fixtures")
	}
	if !ShouldAutoApproveFileChange(ws+"/Makefile", ws) {
		t.Fatal("absolute root Makefile should auto-approve")
	}
	if ShouldAutoApproveFileChange("scripts/Makefile", ws) {
		t.Fatal("nested Makefile should stay protected")
	}
}

func TestShouldAutoApproveFileChange_tailwindConfig(t *testing.T) {
	ws := "/tmp/react-fixture"
	if !ShouldAutoApproveFileChange("tailwind.config.js", ws) {
		t.Fatal("root tailwind.config.js should auto-approve during implementation sessions")
	}
	if ShouldAutoApproveFileChange("package.json", ws) {
		t.Fatal("package.json should remain protected")
	}
}

func TestShouldAutoApproveFileChangeOp_greenfieldManifestCreate(t *testing.T) {
	ws := t.TempDir()
	for _, name := range []string{"Cargo.toml", "package.json", "go.mod"} {
		if ShouldAutoApproveFileChange(name, ws) {
			t.Fatalf("%s must stay protected for edit policy", name)
		}
		if !ShouldAutoApproveFileChangeOp(name, true, ws) {
			t.Fatalf("missing %s create should auto-approve (greenfield)", name)
		}
	}
	// Existing package.json stays protected even on "create" race — Stat sees it.
	pkg := filepath.Join(ws, "package.json")
	if err := os.WriteFile(pkg, []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if ShouldAutoApproveFileChangeOp("package.json", true, ws) {
		t.Fatal("existing package.json must not greenfield-auto-approve")
	}
}

func TestShouldAutoApproveFileChangeOp_greenfieldRootHTML(t *testing.T) {
	ws := t.TempDir()
	if !ShouldAutoApproveFileChangeOp("index.html", true, ws) {
		t.Fatal("missing root index.html create should auto-approve")
	}
	if ShouldAutoApproveFileChange("index.html", ws) {
		t.Fatal("index.html without greenfield create flag stays untrusted")
	}
}

func TestRelativizeFileChangePath(t *testing.T) {
	ws := "/data/minimal-repo"
	got := RelativizeFileChangePath(ws+"/core/sample/main.go", ws)
	if got != "core/sample/main.go" {
		t.Fatalf("got %q", got)
	}
	if RelativizeFileChangePath("/outside/other.go", ws) != "" {
		t.Fatal("path outside workspace should not relativize")
	}
}
func TestValidateConfigJSONContent_rejectsNull(t *testing.T) {
	if err := validateConfigJSONContent("tauri.conf.json", "null"); err == nil {
		t.Fatal("expected error for null literal")
	}
	if err := validateConfigJSONContent("tauri.conf.json", `{"build":{}}`); err != nil {
		t.Fatalf("valid json: %v", err)
	}
}
