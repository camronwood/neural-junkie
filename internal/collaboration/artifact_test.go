package collaboration

import (
	"strings"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExtractTasksFromPlanSupportsKebabMentions(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "a-1", AgentName: "agent-a", AgentType: protocol.AgentTypeBackend},
		{AgentID: "b-1", AgentName: "agent-b", AgentType: protocol.AgentTypeFrontend},
	}

	planContent := `## Tasks

- @agent-a: implement backend parser support
- @agent-b: add UI wiring for collaborations
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].AssignedName != "agent-a" || tasks[1].AssignedName != "agent-b" {
		t.Fatalf("expected assignments to resolve kebab-case mentions, got %+v", tasks)
	}
}

func TestExtractTasksFromPlanSupportsHeadingWithAssignedLine(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "rust-1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust},
		{AgentID: "sec-1", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity},
	}

	planContent := `## Plan

### Task 1: Build CLI command interface
- Assigned to: @RustExpert
- Acceptance: command parses args and prints help

### Task 2: Add encryption key handling
- Assigned to: @SecurityExpert
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].AssignedTo != "rust-1" {
		t.Fatalf("expected task 1 assigned to rust-1, got %s", tasks[0].AssignedTo)
	}
	if tasks[1].AssignedTo != "sec-1" {
		t.Fatalf("expected task 2 assigned to sec-1, got %s", tasks[1].AssignedTo)
	}
}

func TestExtractTasksFromPlanParsesDependencies(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "a1", AgentName: "RustExpert", AgentType: protocol.AgentTypeRust},
		{AgentID: "a2", AgentName: "SecurityExpert", AgentType: protocol.AgentTypeSecurity},
		{AgentID: "a3", AgentName: "GoExpert", AgentType: protocol.AgentTypeBackend},
	}

	planContent := `## Plan

- Task 1: @RustExpert - Scaffold CLI
- Task 2: @SecurityExpert - Threat model
- Task 3: @GoExpert - Integration tests
  - depends: 1, 2
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if len(tasks[2].Dependencies) != 2 {
		t.Fatalf("task 3 deps: %#v", tasks[2].Dependencies)
	}
	if tasks[2].Dependencies[0] != tasks[0].ID || tasks[2].Dependencies[1] != tasks[1].ID {
		t.Fatalf("expected deps on task 1 and 2 ids, got %#v", tasks[2].Dependencies)
	}
	if err := ValidateDAG(tasks); err != nil {
		t.Fatalf("ValidateDAG: %v", err)
	}
}

func TestExtractTasksFromPlanIgnoresBareMentionSummary(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "gemini-1", AgentName: "Gemini", AgentType: protocol.AgentTypeCLI},
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
	}

	planContent := `## Plan

@Gemini has proposed a structured plan and should update later tasks.
- Task 1: @SoftwareArchitect - Investigate existing API definitions
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 1 {
		t.Fatalf("expected only the real task, got %d: %#v", len(tasks), tasks)
	}
	if tasks[0].AssignedName != "SoftwareArchitect" {
		t.Fatalf("expected SoftwareArchitect assignment, got %+v", tasks[0])
	}
}

func TestExtractTasksFromPlanIgnoresNumberedPlanSteps(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
	}
	planContent := `## Plan

1. @SoftwareArchitect, please begin by reviewing the current API documentation and schema files.
1. @SoftwareArchitect, please begin by reviewing the current API documentation and schema files located at docs/api/.
- Task 1: @SoftwareArchitect - Review resource-api JSON schemas and registration
`
	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task from structured line, got %d: %#v", len(tasks), tasks)
	}
}

func TestDedupeTasksCollapsesSimilarAssignments(t *testing.T) {
	now := time.Now()
	tasks := []CollaborationTask{
		{ID: "1", Title: "Review API docs and schema files", AssignedTo: "a1", Description: "Review API docs", CreatedAt: now, UpdatedAt: now},
		{ID: "2", Title: "Review API docs and schema files", AssignedTo: "a1", Description: "Review API docs", CreatedAt: now, UpdatedAt: now},
		{ID: "3", Title: "Investigate CI", AssignedTo: "a2", Description: "CI", CreatedAt: now, UpdatedAt: now},
	}
	out := DedupeTasks(tasks)
	if len(out) != 2 {
		t.Fatalf("expected 2 after dedupe, got %d", len(out))
	}
}

func TestExtractTasksFromPlanParsesMarkdownNumberedBoldTasks(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
	}

	planContent := `## Tasks

1. **Compile industry standards for API schema documentation.** - ` + "`pending`" + ` (@SoftwareArchitect)
   - **Dependencies**: None
   - **Milestones**:
     - Research industry standards
     - Document findings

2. **Perform code analysis on resource-api JSON registration.** - ` + "`pending`" + ` (@BackendEngineer)
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %#v", len(tasks), tasks)
	}
	if tasks[0].AssignedName != "SoftwareArchitect" {
		t.Fatalf("task 0 assignee: %+v", tasks[0])
	}
	if tasks[0].Title != "Compile industry standards for API schema documentation." {
		t.Fatalf("task 0 title: %q", tasks[0].Title)
	}
	if strings.HasPrefix(strings.ToLower(tasks[0].Title), "task is to") {
		t.Fatalf("unexpected fragment title: %q", tasks[0].Title)
	}
}

func TestExtractTasksFromPlanRejectsWeakFragmentBullets(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
	}
	planContent := `## Plan

- task is to compile industry standards and best practices
- Task 1: @SoftwareArchitect - Investigate resource-api JSON schemas
`
	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %#v", len(tasks), tasks)
	}
}

func TestExtractTasksFromPlan_keepsDocumentFindingsWithDeliverablePath(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "asst-1", AgentName: "Assistant", AgentType: protocol.AgentTypeAssistant},
	}
	planContent := `## Plan

Task 5: @Assistant - Document findings and next steps in collabs/abc-123/findings.md
`
	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %#v", len(tasks), tasks)
	}
	if isWeakTaskFragment(tasks[0].Description) {
		t.Fatalf("task with collabs/ path should not be weak: %q", tasks[0].Description)
	}
}

func TestNormalizeAndValidateTasksForExecution_keepsDocumentFindingsTask(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "asst-1", AgentName: "Assistant", AgentType: protocol.AgentTypeAssistant},
	}
	plan := `## Plan

Task 5: @Assistant - Document findings and next steps in collabs/abc-123/findings.md
`
	c := &Collaboration{
		ID:             "abc-123",
		Description:    "Produce a markdown document",
		Agents:         agents,
		SourceRepoPath: "/tmp/project",
		Plan:           &SharedArtifact{Content: plan},
	}
	tasks, _ := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d: %#v", len(tasks), tasks)
	}
}

func TestExtractTasksFromPlanSanitizesAssetMentionAssignee(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "gemini-1", AgentName: "Gemini", AgentType: protocol.AgentTypeCLI},
	}

	planContent := `## Plan

- Task 1: @assets/icons/Gemini_Generated_Image_7ofmua7ofmua7ofm.png - Analyze current API definitions
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].AssignedName != "Gemini" || tasks[0].AssignedTo != "gemini-1" {
		t.Fatalf("bogus @mention should round-robin to participant: %+v", tasks[0])
	}
	if tasks[0].Description != "Analyze current API definitions" {
		t.Fatalf("unexpected sanitized description %q", tasks[0].Description)
	}
}
