package collaboration

import (
	"testing"

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
