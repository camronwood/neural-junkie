package collaboration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferTaskContextPathsResourceAPI(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"core/resource-api", "resource-api/json_endpoints"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	task := CollaborationTask{
		Title:       "Investigate resource api schema",
		Description: "Standardize registration in resource-api/json_endpoints",
	}
	paths := InferTaskContextPaths(task, root)
	if len(paths) < 2 {
		t.Fatalf("expected resource-api paths, got %v", paths)
	}
}

func TestInferTaskContextPathsFindingsSources(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"README.md", "core/sample/main.go"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("sample\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task := CollaborationTask{
		Description: "Document findings in collabs/x/findings.md summarizing README.md and core/sample/main.go",
	}
	paths := InferTaskContextPaths(task, root)
	want := map[string]bool{"README.md": false, "core/sample/main.go": false}
	for _, p := range paths {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for p, ok := range want {
		if !ok {
			t.Fatalf("expected %q in context paths, got %v", p, paths)
		}
	}
}
