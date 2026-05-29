package collaboration

import "testing"

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
