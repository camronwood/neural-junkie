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
