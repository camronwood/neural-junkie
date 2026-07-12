package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendOutlineEntries_skipsNodeModules(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.js"), []byte("x"), 0644); err != nil {
		t.Fatalf("write node_modules file: %v", err)
	}

	var b strings.Builder
	appendOutlineEntries(&b, root, root, 0, 2)
	tree := b.String()
	if !strings.Contains(tree, "README.md") {
		t.Fatalf("expected README.md in outline, got:\n%s", tree)
	}
	if strings.Contains(tree, "node_modules") {
		t.Fatalf("node_modules should be excluded, got:\n%s", tree)
	}
}

func TestBuildCollabOutlineFileTree_PrioritizesResourceAPI(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{
		"resource-api/json_endpoints/products.json",
		"core/sample/main.go",
		"docs/tim/README.md",
	} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	tree := buildCollabOutlineFileTree(root, "Investigate resource api document schema standardization", 2)
	if !strings.Contains(tree, "Key project paths (start here):") {
		t.Fatalf("expected priority section, got:\n%s", tree)
	}
	idxEndpoints := strings.Index(tree, "resource-api/json_endpoints")
	idxMain := strings.Index(tree, "core/sample/main.go")
	if idxEndpoints < 0 {
		t.Fatalf("expected json_endpoints in tree:\n%s", tree)
	}
	if idxMain >= 0 && idxMain < idxEndpoints {
		t.Fatalf("expected json_endpoints before main.go, got:\n%s", tree)
	}
}

func TestInferOutlinePriorityPaths_SkipsMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "resource-api/json_endpoints"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	paths := inferOutlinePriorityPaths(root, "resource api schema registration")
	if len(paths) == 0 || paths[0] != "resource-api/json_endpoints" {
		t.Fatalf("paths = %v", paths)
	}
}
