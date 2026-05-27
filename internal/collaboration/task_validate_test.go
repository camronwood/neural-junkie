package collaboration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCollaborationPathsMissingDocsAPI(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "core/resource-api"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Collaboration{
		SourceRepoPath: root,
		Plan: &SharedArtifact{
			Content: "## Plan\n\nReview docs/api/ and schema/.\n\n- Task 1: @Arch - Read docs/api/resource.md",
		},
		Tasks: []CollaborationTask{
			{
				Title:       "Review schemas",
				Description: "Inspect docs/api/ and resource-api/json_endpoints/",
			},
		},
	}
	issues := ValidateCollaborationPaths(c)
	if len(issues) == 0 {
		t.Fatal("expected missing path warnings")
	}
	foundDocs := false
	for _, iss := range issues {
		if strings.Contains(iss.Path, "docs/api") {
			foundDocs = true
		}
	}
	if !foundDocs {
		t.Fatalf("expected docs/api warning, got %#v", issues)
	}
}

func TestValidateCollaborationPathsSkipsExistingAndCollabs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "core/resource-api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core/resource-api/README.md"), []byte("#"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Collaboration{
		SourceRepoPath: root,
		Tasks: []CollaborationTask{{
			Title:       "Write deliverable",
			Description: "Output to collabs/abc-123/findings.md and read core/resource-api/README.md",
		}},
	}
	issues := ValidateCollaborationPaths(c)
	for _, iss := range issues {
		if strings.HasPrefix(iss.Path, "collabs/") || iss.Path == "core/resource-api/README.md" {
			t.Fatalf("unexpected issue: %#v", iss)
		}
	}
}
