package agent

import "testing"

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

func TestValidateConfigJSONContent_rejectsNull(t *testing.T) {
	if err := validateConfigJSONContent("tauri.conf.json", "null"); err == nil {
		t.Fatal("expected error for null literal")
	}
	if err := validateConfigJSONContent("tauri.conf.json", `{"build":{}}`); err != nil {
		t.Fatalf("valid json: %v", err)
	}
}
