package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExtractTasksFromCollaborationGoal_distinctDeliverables(t *testing.T) {
	goal := "Produce exactly three tasks: Task 1 @SoftwareArchitect Write collabs/<id>/schema-outline.md; Task 2 @SoftwareArchitect Write collabs/<id>/doc-standards.md; Task 3 @BackendEngineer Write collabs/<id>/api-notes.md."
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
	}
	tasks := ExtractTasksFromCollaborationGoal(goal, "abc-123", agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
	arch := 0
	for _, task := range tasks {
		if task.AssignedName == "SoftwareArchitect" {
			arch++
		}
	}
	if arch != 2 {
		t.Fatalf("expected 2 architect tasks, got %d", arch)
	}
}

func TestExtractTasksFromCollaborationGoal_compoundInlineTasks(t *testing.T) {
	goal := "Use exactly: - Task 1: @SoftwareArchitect - Write collabs/<id>/design.md. - Task 2: @BackendEngineer - Write collabs/<id>/filter.go."
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
	}
	tasks := ExtractTasksFromCollaborationGoal(goal, "abc-123", agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
}

func TestExtractTasksFromCollaborationGoal_fileAssigneeList(t *testing.T) {
	goal := "Produce a short plan with exactly three file tasks under collabs/<id>/: api_schema.md (@BackendEngineer), markdown_doc_structure.md (@SoftwareArchitect), ci_cd_pipeline.md (@PlatformEngineer)."
	agents := []CollaborationAgent{
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "plat-1", AgentName: "PlatformEngineer", AgentType: protocol.AgentTypeDevOps},
	}
	tasks := ExtractTasksFromCollaborationGoal(goal, "f7518f88-50a4-4561-9e88-174381f3090d", agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %#v", len(tasks), taskTitles(tasks))
	}
}

func TestExtractTasksFromCollaborationGoal_unassignedFindingsTask(t *testing.T) {
	goal := "Task 1 @BackendEngineer Write collabs/<id>/api_schema.md; Task 4 Document findings in collabs/<id>/findings.md (any assignee)."
	agents := []CollaborationAgent{
		{AgentID: "asst-1", AgentName: "Assistant", AgentType: protocol.AgentTypeGeneral},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
	}
	tasks := ExtractTasksFromCollaborationGoal(goal, "abc-123", agents)
	var hasFindings bool
	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Description), "findings.md") {
			hasFindings = true
		}
	}
	if !hasFindings {
		t.Fatalf("expected findings.md task in %#v", taskTitles(tasks))
	}
}

func TestExtractTasksFromCollaborationGoal_documentFindingsExecutionGoal(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
		{AgentID: "sa-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
	}
	goal := "@BackendEngineer @BackendEngineer Plan one task using this exact line: - Task 1: @BackendEngineer - Document findings in collabs/<id>/findings.md summarizing README.md and core/sample/main.go."
	tasks := ExtractTasksFromCollaborationGoal(goal, "ac310c77-9d42-4981-9b53-c9e814243809", agents)
	if len(tasks) > 2 {
		t.Fatalf("expected <=2 tasks from goal, got %d: %#v", len(tasks), tasks)
	}
}

func TestGoalPinsExactTaskList(t *testing.T) {
	if !goalPinsExactTaskList("Produce EXACTLY three tasks and no others, using these exact lines:\n- Task 1...", 3) {
		t.Fatal("expected EXACTLY three + exact lines to pin")
	}
	if !goalPinsExactTaskList("Plan one task using this exact line: - Task 1: @BE - Write findings.md", 1) {
		t.Fatal("expected single exact-line pin")
	}
	if !goalPinsExactTaskList("Investigate schema. Plan exactly: Task 1 @BE Write a.md; Task 2 @SA Write b.md", 2) {
		t.Fatal("expected Plan exactly: to pin")
	}
	if goalPinsExactTaskList("Write a flexible plan with a few tasks", 3) {
		t.Fatal("freestyle goal should not pin")
	}
}

func TestEnsurePlanTasksFromGoal_replacesFreestyleWhenPinned(t *testing.T) {
	cm := NewCollaborationManager(nil)
	agents := []CollaborationAgent{
		{AgentID: "fe-1", AgentName: "FrontendEngineer", AgentType: protocol.AgentTypeFrontend},
		{AgentID: "sa-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "cl-1", AgentName: "Claude", AgentType: protocol.AgentTypeAssistant},
	}
	c := &Collaboration{
		ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Description: "@FrontendEngineer @SoftwareArchitect @Claude Produce EXACTLY three tasks and no others, using these exact lines:\n" +
			"- Task 1: @SoftwareArchitect - Write collabs/<id>/site-structure.md (navigation)\n" +
			"- Task 2: @FrontendEngineer - Write collabs/<id>/design-system.md (palette)\n" +
			"- Task 3: @FrontendEngineer - Write collabs/<id>/layout-specs.md (templates)",
		Agents: agents,
		Plan: &SharedArtifact{Content: `## Plan

Task 1: @SoftwareArchitect - Review existing HTML structure for Collaboration Station
Task 2: @FrontendEngineer - Sketch color tokens for black/white/gray/blue/red
Task 3: @Claude - Propose page inventory for home/about/contact
`},
	}
	cm.mu.Lock()
	cm.ensurePlanTasksFromGoalLocked(c)
	cm.mu.Unlock()
	if len(c.Tasks) != 3 {
		t.Fatalf("expected pinned goal list of 3, got %d: %#v", len(c.Tasks), taskTitles(c.Tasks))
	}
	joined := strings.ToLower(c.Plan.Content)
	for _, needle := range []string{"site-structure.md", "design-system.md", "layout-specs.md"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("plan missing %s: %s", needle, c.Plan.Content)
		}
	}
	if strings.Contains(joined, "page inventory") || strings.Contains(joined, "color tokens") {
		t.Fatalf("freestyle discussion tasks should not remain after pin: %s", c.Plan.Content)
	}
}
