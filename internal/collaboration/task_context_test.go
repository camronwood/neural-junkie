package collaboration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferTaskContextPaths_trailingPunctuation(t *testing.T) {
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
		Description: "Document findings in collabs/x/findings.md summarizing README.md and core/sample/main.go.",
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

func TestInferTaskContextPaths_shortCollabPrefixExpands(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "collabs", "b222bffe-39e8-4b00-91ca-ee1c555b9592")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "style.css", "contact.html"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<html>xss contact</html>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	task := CollaborationTask{
		Title:       "Security audit",
		Description: "Write collabs/x/security-audit.md reviewing b222bffe HTML/CSS for XSS and contact-form handling",
	}
	paths := InferTaskContextPaths(task, root)
	want := map[string]bool{
		"collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/index.html":   false,
		"collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/style.css":    false,
		"collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/contact.html": false,
	}
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
