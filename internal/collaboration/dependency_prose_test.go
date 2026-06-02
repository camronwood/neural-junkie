package collaboration

import (
	"strings"
	"testing"
)

func TestParseHardDependencyProse(t *testing.T) {
	cases := []struct {
		line     string
		from     int
		to       []int
		ok       bool
	}{
		{"- Task 1 depends on Task 2 for the markdown structure and style guide.", 1, []int{2}, true},
		{"Task 3 depends on Task 1, Task 2", 3, []int{1, 2}, true},
		{"- Task 3 can be started independently but should reference the schema.", 0, nil, false},
		{"- Task 1 depends on Task 1 for context", 0, nil, false},
		{"Task 1: @BackendEngineer - Write collabs/x/a.md", 0, nil, false},
	}
	for _, tc := range cases {
		from, to, ok := parseHardDependencyProse(tc.line)
		if ok != tc.ok {
			t.Errorf("parseHardDependencyProse(%q) ok=%v want %v", tc.line, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if from != tc.from {
			t.Errorf("parseHardDependencyProse(%q) from=%d want %d", tc.line, from, tc.from)
		}
		if len(to) != len(tc.to) {
			t.Errorf("parseHardDependencyProse(%q) to=%v want %v", tc.line, to, tc.to)
			continue
		}
		for i := range to {
			if to[i] != tc.to[i] {
				t.Errorf("parseHardDependencyProse(%q) to=%v want %v", tc.line, to, tc.to)
				break
			}
		}
	}
}

func TestApplyPlanDependencyProse_f7518f88(t *testing.T) {
	plan := `## Plan

Task 1: @BackendEngineer - Write collabs/f7518f88/api_schema.md
Task 2: @SoftwareArchitect - Write collabs/f7518f88/markdown_doc_structure.md
Task 3: @PlatformEngineer - Write collabs/f7518f88/ci_cd_pipeline.md
- Task 1 depends on Task 2 for the markdown structure and style guide.
- Task 3 can be started independently but should reference the schema.
`
	tasks := ExtractTasksFromPlan(plan, phoenixAgents())
	tasks = mergeNearDuplicateTasks(tasks)
	for i := range tasks {
		if tasks[i].ID == "" {
			tasks[i].ID = "task-" + string(rune('a'+i))
		}
	}
	warnings := ApplyPlanDependencyProse(plan, tasks)
	if len(warnings) > 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if len(tasks[0].Dependencies) != 1 || tasks[0].Dependencies[0] != tasks[1].ID {
		t.Fatalf("task 1 should depend on task 2, deps=%v", tasks[0].Dependencies)
	}
	if len(tasks[2].Dependencies) != 0 {
		t.Fatalf("task 3 should have no deps, got %v", tasks[2].Dependencies)
	}
}

func TestApplyPlanDependencyProse_cycleDropped(t *testing.T) {
	plan := `- Task 1 depends on Task 2
- Task 2 depends on Task 1
`
	tasks := []CollaborationTask{
		{ID: "a", Title: "One"},
		{ID: "b", Title: "Two"},
	}
	warnings := ApplyPlanDependencyProse(plan, tasks)
	if len(warnings) == 0 {
		t.Fatal("expected cycle warning")
	}
	if !strings.Contains(warnings[0], "cycle") {
		t.Fatalf("expected cycle in warning, got %q", warnings[0])
	}
	if len(tasks[0].Dependencies) != 0 || len(tasks[1].Dependencies) != 0 {
		t.Fatalf("deps should be reverted on cycle, got %v %v", tasks[0].Dependencies, tasks[1].Dependencies)
	}
}

func TestNormalizeAndValidateTasksForExecution_f7518f88_dependencies(t *testing.T) {
	plan := `## Plan

Task 1: @BackendEngineer - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/api_schema.md defining the API schema and registration process.
Task 2: @SoftwareArchitect - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/markdown_doc_structure.md detailing the markdown document structure and style guide.
Task 3: @PlatformEngineer - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/ci_cd_pipeline.md outlining the CI/CD pipeline for schema and documentation updates.
- Task 1 depends on Task 2 for the markdown structure and style guide.
- Task 3 can be started independently but should reference the schema and markdown documents once they are available.
`
	c := &Collaboration{
		ID:             "f7518f88-50a4-4561-9e88-174381f3090d",
		Description:    "Investigate resource api document schema standardization/registration",
		Agents:         phoenixAgents(),
		SourceRepoPath: "/Users/test/Phoenix",
		Plan:           &SharedArtifact{Content: plan},
	}
	tasks, _ := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	apiTask := tasks[0]
	mdTask := tasks[1]
	if !strings.Contains(apiTask.Description, "api_schema") {
		t.Fatalf("expected api_schema task first, got %q", apiTask.Description)
	}
	if !strings.Contains(mdTask.Description, "markdown_doc_structure") {
		t.Fatalf("expected markdown task second, got %q", mdTask.Description)
	}
	if len(apiTask.Dependencies) != 1 || apiTask.Dependencies[0] != mdTask.ID {
		t.Fatalf("api task should depend on markdown task, deps=%v want %q", apiTask.Dependencies, mdTask.ID)
	}
}
