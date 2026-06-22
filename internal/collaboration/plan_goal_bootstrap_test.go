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
