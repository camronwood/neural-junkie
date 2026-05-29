package collaboration

import "testing"

func TestMergeTaskContextIntoDescriptionAddsWriteVerb(t *testing.T) {
	desc := mergeTaskContextIntoDescription("Define Scope and Objectives", []string{
		"- **Deliverable:** `collabs/abc/scope.md`",
	})
	if !TaskRequiresFileDeliverable(CollaborationTask{Description: desc}) {
		t.Fatalf("expected file deliverable task, got %q", desc)
	}
	if want := "Write collabs/abc/scope.md"; !contains(desc, want) {
		t.Fatalf("desc=%q want substring %q", desc, want)
	}
}

func TestSplitPlanTaskSections(t *testing.T) {
	plan := `### Task 1: Define Scope and Objectives
- **Deliverable:** ` + "`collabs/abc/scope.md`" + `
`
	sections := splitPlanTaskSections(plan)
	key := normalizeTaskSectionKey("Define Scope and Objectives")
	body, ok := sections[key]
	if !ok {
		t.Fatalf("missing section for %q, keys=%v", key, sections)
	}
	if body == "" {
		t.Fatal("empty section body")
	}
}

func TestEnrichTasksWithPlanDeliverables(t *testing.T) {
	plan := `### Task 1: Define Scope and Objectives
- **Deliverable:** ` + "`collabs/abc/scope.md`" + `
- **Details:** List APIs in scope

### Task 2: Review Existing API Documentation
- **Deliverable:** ` + "`collabs/abc/existing-schema.md`" + `
`
	tasks := []CollaborationTask{{
		Title:        "Define Scope and Objectives",
		Description:  "Define Scope and Objectives",
		AssignedName: "BackendEngineer",
	}}
	EnrichTasksWithPlanDeliverables(plan, tasks)
	if !TaskRequiresFileDeliverable(tasks[0]) {
		t.Fatalf("task 0 should require file deliverable: %q", tasks[0].Description)
	}
}

func TestNormalizeTaskDeliverablePathsForSandbox(t *testing.T) {
	c := &Collaboration{ID: "abc-1234-5678-90ab-cdef00000000"}
	tasks := []CollaborationTask{{
		Description: "Write collabs/abc-1234/scope.md with goals",
	}}
	NormalizeTaskDeliverablePathsForSandbox(c, tasks)
	if contains(tasks[0].Description, "collabs/") {
		t.Fatalf("expected flattened sandbox path, got %q", tasks[0].Description)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
