package collaboration

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestClimb0302GoalTaskCounts(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "be", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
		{AgentID: "sa", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "cl", AgentName: "Claude", AgentType: protocol.AgentTypeAssistant},
		{AgentID: "pe", AgentName: "PlatformEngineer", AgentType: protocol.AgentTypeDevOps},
		{AgentID: "fe", AgentName: "FrontendEngineer", AgentType: protocol.AgentTypeFrontend},
	}
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	cases := []struct {
		name string
		goal string
		min  int
		max  int
		pin  bool
	}{
		{
			name: "plan-combined",
			goal: `@BackendEngineer @SoftwareArchitect @Claude Investigate resource API document schema standardization. Plan exactly: Task 1 @BackendEngineer Write collabs/<id>/api_schema.md; Task 2 @SoftwareArchitect Write collabs/<id>/markdown_doc_structure.md; Task 3 @Claude Write collabs/<id>/ci_cd_pipeline.md describing CI/CD in prose (not running docker/npm); Task 4 Document findings in collabs/<id>/findings.md. After tasks add dependency bullets like '- Task 1 depends on Task 2' — those are notes, NOT separate tasks.`,
			min: 3, max: 5, pin: true,
		},
		{
			name: "schema-planning",
			goal: `@SoftwareArchitect @PlatformEngineer @Claude Investigate resource api document schema standardization/registration. Produce EXACTLY two tasks and no others, using these exact lines:
- Task 1: @SoftwareArchitect - Write collabs/<id>/schema-standards.md covering schema/doc standards from resource-api/json_endpoints and docs/tim
- Task 2: @PlatformEngineer - Write collabs/<id>/ci-release-docs.md covering CI/release of docs (not schema design). @Claude synthesizes the plan only.`,
			min: 2, max: 2, pin: true,
		},
		{
			name: "schema-regression",
			goal: `@BackendEngineer @FrontendEngineer @Claude Investigate resource api document schema standardization/registration. Produce EXACTLY two tasks and no others, using these exact lines:
- Task 1: @BackendEngineer - Define Scope with Deliverable collabs/<id>/scope.md
- Task 2: @BackendEngineer - Review API docs with Deliverable collabs/<id>/existing-schema.md
Produce real markdown deliverables, not chat-only summaries. @FrontendEngineer and @Claude discuss only.`,
			min: 2, max: 2, pin: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks := ExtractTasksFromCollaborationGoal(tc.goal, id, agents)
			if len(tasks) < tc.min || len(tasks) > tc.max {
				t.Fatalf("tasks=%d want [%d,%d]: %#v", len(tasks), tc.min, tc.max, tasks)
			}
			if goalPinsExactTaskList(tc.goal, len(tasks)) != tc.pin {
				t.Fatalf("pin=%v want %v (tasks=%d)", !tc.pin, tc.pin, len(tasks))
			}
		})
	}
}
