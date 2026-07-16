package collaboration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentReplyContainsStalePlanning(t *testing.T) {
	content := "Looks good.\n\n**Approve the Plan:**\nUse the command /approve-plan to proceed.\n\nTASK_STATUS: completed\n"
	if !AgentReplyContainsStalePlanning(content) {
		t.Fatal("expected stale planning detection")
	}
	clean := SanitizeCollabExecutionResponse(content, string(PhaseExecuting))
	if AgentReplyContainsStalePlanning(clean) {
		t.Fatalf("sanitize left stale planning: %q", clean)
	}
}

func TestMaterializePlanDeliverableStubsSandbox(t *testing.T) {
	root := t.TempDir()
	c := &Collaboration{
		ID:               "abc-1234-5678-90ab-cdef00000000",
		WorkingDirectory: root,
		Plan: &SharedArtifact{Content: `### Task 1: Scope
- **Deliverable:** collabs/abc-1234/scope.md
`},
		Tasks: []CollaborationTask{{
			Title:       "Scope",
			Description: "Write collabs/abc-1234/scope.md",
		}},
	}
	created, err := MaterializePlanDeliverableStubs(c)
	if err != nil {
		t.Fatalf("MaterializePlanDeliverableStubs: %v", err)
	}
	if len(created) == 0 {
		t.Fatal("expected stub files")
	}
}

func TestIsDeliverableStubContent(t *testing.T) {
	body := []byte("# plan.md\n\n" + deliverableStubMarker + ". Replace with task output._\n")
	if !IsDeliverableStubContent(body) {
		t.Fatal("expected stub marker detection")
	}
	if IsDeliverableStubContent([]byte("# real\n\nactual content here\n")) {
		t.Fatal("real content should not be stub")
	}
}

func TestMaterializePlanDeliverableStubs_skipsHTMLAndTruncatedPaths(t *testing.T) {
	root := t.TempDir()
	c := &Collaboration{
		ID:               "abc-1234-5678-90ab-cdef00000000",
		WorkingDirectory: root,
		Tasks: []CollaborationTask{{
			Title:       "Create `collabs/abc-1234/index....`",
			Description: "Create collabs/abc-1234/index.html",
		}},
	}
	created, err := MaterializePlanDeliverableStubs(c)
	if err != nil {
		t.Fatalf("MaterializePlanDeliverableStubs: %v", err)
	}
	for _, rel := range created {
		if filepath.Base(rel) == "index...." {
			t.Fatalf("should not create truncated stub %q", rel)
		}
		if filepath.Ext(rel) == ".html" {
			t.Fatalf("HTML deliverables must not get plan-approval stubs, got %q", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); !os.IsNotExist(err) {
		t.Fatalf("expected no HTML stub on disk, stat err=%v", err)
	}
}
