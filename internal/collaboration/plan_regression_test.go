package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// Regression fixtures distilled from live collab failures (4ea36409, f7518f88).

func phoenixAgents() []CollaborationAgent {
	return []CollaborationAgent{
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "plat-1", AgentName: "PlatformEngineer", AgentType: protocol.AgentTypeDevOps},
		{AgentID: "asst-1", AgentName: "Assistant", AgentType: protocol.AgentTypeAssistant},
		{AgentID: "cursor-1", AgentName: "Cursor", AgentType: protocol.AgentTypeCLI},
	}
}

func TestExtractTasksFromPlan_plainTaskRowsWithoutBullets(t *testing.T) {
	plan := `## Plan

Task 1 @SoftwareArchitect Write collabs/abc/schema-outline.md
Task 2 @SoftwareArchitect Write collabs/abc/doc-standards.md
Task 3 @BackendEngineer Write collabs/abc/api-notes.md
`
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
	}
	tasks := ExtractTasksFromPlan(plan, agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
	archCount := 0
	for _, task := range tasks {
		if task.AssignedName == "SoftwareArchitect" {
			archCount++
		}
	}
	if archCount != 2 {
		t.Fatalf("expected 2 architect tasks, got %d", archCount)
	}
}

func TestExtractPlanFromTaskLists_singleStructuredRow(t *testing.T) {
	body := "Task 1: @Assistant - Write collabs/x/findings.md with three README bullets"
	got := ExtractPlanFromTaskLists(body)
	if got == "" {
		t.Fatal("expected plan from single task row")
	}
	tasks := ExtractTasksFromPlan(got, phoenixAgents())
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestExtractTasksFromPlan_f7518f88_rejectsDependencyProse(t *testing.T) {
	plan := `## Plan

Task 1: @BackendEngineer - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/api_schema.md defining the API schema and registration process.
Task 2: @SoftwareArchitect - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/markdown_doc_structure.md detailing the markdown document structure and style guide.
Task 3: @PlatformEngineer - Write collabs/f7518f88-50a4-4561-9e88-174381f3090d/ci_cd_pipeline.md outlining the CI/CD pipeline for schema and documentation updates.
- Task 1 depends on Task 2 for the markdown structure and style guide.
- Task 3 can be started independently but should reference the schema and markdown documents once they are available.
`
	tasks := ExtractTasksFromPlan(plan, phoenixAgents())
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
	for _, title := range taskTitles(tasks) {
		lower := strings.ToLower(title)
		if strings.Contains(lower, "depends on") || strings.Contains(lower, "can be started") {
			t.Fatalf("dependency prose parsed as task: %q", title)
		}
	}
}

func TestNormalizeAndValidateTasksForExecution_f7518f88_threeFileTasks(t *testing.T) {
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
	tasks, warnings := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 normalized tasks, got %d warnings=%v tasks=%v", len(tasks), warnings, taskTitles(tasks))
	}
	for _, task := range tasks {
		if !TaskRequiresFileDeliverable(task) {
			t.Fatalf("expected file deliverable task, got %q", task.Description)
		}
		if task.AssignedTo == "" {
			t.Fatalf("task missing assignee: %q", task.Description)
		}
	}
}

func TestExtractTasksFromPlan_4ea36409_keepsFindingsTask(t *testing.T) {
	plan := `## Plan

Task 1: @SoftwareArchitect - Define resource API document schema and registration approach
Task 2: @Assistant - Synthesize requirements for resource API document schema standardization/registration
Task 3: @Cursor - Implement initial draft of markdown document for resource API document schema standardization/registration
Task 4: @SoftwareArchitect - Review and finalize markdown document for resource API document schema standardization/registration
Task 5: @Assistant - Document findings and next steps in collabs/4ea36409-3abd-4760-94b5-70c1b3758a2b/findings.md
`
	tasks := ExtractTasksFromPlan(plan, phoenixAgents())
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
	last := tasks[len(tasks)-1]
	if !strings.Contains(last.Description, "findings.md") {
		t.Fatalf("task 5 missing findings path: %q", last.Description)
	}
}

func TestTaskPassesExecutionQuality_findingsTask(t *testing.T) {
	task := CollaborationTask{
		Title:       "Document findings and next steps in collabs/4ea36409/findings.md",
		Description: "Document findings and next steps in collabs/4ea36409-3abd-4760-94b5-70c1b3758a2b/findings.md",
		AssignedTo:  "asst-1",
	}
	if !TaskRequiresFileDeliverable(task) {
		t.Fatal("expected file deliverable")
	}
	if isWeakTaskFragment(task.Description) {
		t.Fatal("findings task should not be weak")
	}
	if !taskPassesExecutionQuality(task) {
		t.Fatal("findings task should pass execution quality")
	}
}

func TestNormalizeAndValidateTasksForExecution_4ea36409_fiveTasks(t *testing.T) {
	plan := `## Plan

Task 1: @SoftwareArchitect - Define resource API document schema and registration approach
Task 2: @Assistant - Synthesize requirements for resource API document schema standardization/registration
Task 3: @Cursor - Implement initial draft of markdown document for resource API document schema standardization/registration
Task 4: @SoftwareArchitect - Review and finalize markdown document for resource API document schema standardization/registration
Task 5: @Assistant - Document findings and next steps in collabs/4ea36409-3abd-4760-94b5-70c1b3758a2b/findings.md
`
	agents := phoenixAgents()
	extracted := ExtractTasksFromPlan(plan, agents)
	if len(extracted) != 5 {
		t.Fatalf("extract: expected 5, got %d", len(extracted))
	}
	last := extracted[len(extracted)-1]
	if !taskPassesExecutionQuality(last) {
		t.Fatalf("extracted task 5 failed quality: %+v", last)
	}
	filtered := make([]CollaborationTask, 0, len(extracted))
	for _, task := range extracted {
		if taskPassesExecutionQuality(task) {
			filtered = append(filtered, task)
		}
	}
	if len(filtered) != 5 {
		t.Fatalf("filtered: expected 5, got %d titles=%v", len(filtered), taskTitles(filtered))
	}
	merged := mergeNearDuplicateTasks(filtered)
	if len(merged) != 5 {
		t.Fatalf("merged: expected 5, got %d titles=%v", len(merged), taskTitles(merged))
	}

	c := &Collaboration{
		ID:             "4ea36409-3abd-4760-94b5-70c1b3758a2b",
		Description:    "Produce a markdown document",
		Agents:         phoenixAgents(),
		SourceRepoPath: "/Users/test/Phoenix",
		Plan:           &SharedArtifact{Content: plan},
	}
	tasks, _ := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) != 5 {
		t.Fatalf("expected 5 normalized tasks, got %d: %v", len(tasks), taskTitles(tasks))
	}
}

func TestIsTaskDependencyProse(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"- Task 1 depends on Task 2 for the markdown structure and style guide.", true},
		{"- Task 3 can be started independently but should reference the schema.", true},
		{"Task 1: @BackendEngineer - Write collabs/x/api_schema.md", false},
		{"- Task 1: @BackendEngineer - Write collabs/x/api_schema.md", false},
		{"depends on Task 2 for context", true},
	}
	for _, tc := range cases {
		if got := isTaskDependencyProse(tc.line); got != tc.want {
			t.Errorf("isTaskDependencyProse(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestExtractPlanFromTaskLists_ignoresDependencyBullets(t *testing.T) {
	content := `Here is the plan:

- Task 1: @BackendEngineer - Write collabs/x/a.md
- Task 2: @SoftwareArchitect - Write collabs/x/b.md
- Task 1 depends on Task 2 for style.
`
	plan := ExtractPlanFromTaskLists(content)
	if plan == "" {
		t.Fatal("expected plan from task lists")
	}
	tasks := ExtractTasksFromPlan(plan, phoenixAgents())
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks in synthesized plan, got %d: %v", len(tasks), taskTitles(tasks))
	}
}

func TestNormalizeAndValidateTasksForExecution_distinctDeliverablesSameAssignee(t *testing.T) {
	plan := `## Plan

Task 2: @Assistant - Synthesize requirements for resource API document schema standardization/registration
Task 5: @Assistant - Document findings and next steps in collabs/abc/findings.md
`
	c := &Collaboration{
		ID:          "abc",
		Description: "Produce a markdown document",
		Agents:      phoenixAgents(),
		Plan:        &SharedArtifact{Content: plan},
	}
	tasks, _ := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 distinct Assistant tasks, got %d: %v", len(tasks), taskTitles(tasks))
	}
}

func taskTitles(tasks []CollaborationTask) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.Title
	}
	return out
}
