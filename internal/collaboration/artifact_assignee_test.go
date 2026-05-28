package collaboration

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAssignRoundRobinToUnassignedTasks(t *testing.T) {
	tasks := []CollaborationTask{
		{Title: "Review schema", Description: "Review schema"},
		{Title: "Write doc", Description: "Write doc", AssignedTo: "a2", AssignedName: "Architect"},
	}
	agents := []CollaborationAgent{
		{AgentID: "a1", AgentName: "Assistant"},
		{AgentID: "a2", AgentName: "Architect"},
	}
	AssignRoundRobinToUnassignedTasks(tasks, agents)
	if tasks[0].AssignedTo != "a1" {
		t.Fatalf("task0 assignee = %q, want a1", tasks[0].AssignedTo)
	}
	if tasks[1].AssignedTo != "a2" {
		t.Fatalf("task1 assignee changed unexpectedly")
	}
}

func TestExtractTasksFromPlan_RequiresAssigneeOnNumberedBold(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "a1", AgentName: "Assistant"},
		{AgentID: "a2", AgentName: "SoftwareArchitect"},
	}
	plan := `## Tasks

1. **Review and Understand the Current Schema** - ` + "`pending`"
	tasks := ExtractTasksFromPlan(plan, agents)
	if len(tasks) != 0 {
		t.Fatalf("expected no tasks without assignee, got %d", len(tasks))
	}

	plan2 := `## Tasks

1. **Review json_endpoints** - ` + "`pending` (@SoftwareArchitect)"
	tasks2 := ExtractTasksFromPlan(plan2, agents)
	if len(tasks2) != 1 || tasks2[0].AssignedName != "SoftwareArchitect" {
		t.Fatalf("tasks2 = %+v", tasks2)
	}
}

func TestSelectRecapFacilitator_PrefersAssistantForFinal(t *testing.T) {
	c := &Collaboration{
		Agents: []CollaborationAgent{
			{AgentID: "be", AgentName: "BackendEngineer"},
			{AgentID: "as", AgentName: "Assistant"},
		},
		Discussion: &DiscussionSession{
			Messages: []*protocol.Message{
				{From: protocol.AgentInfo{ID: "be", Name: "BackendEngineer"}},
			},
		},
	}
	id := SelectRecapFacilitator(c, RecapKindFinal)
	if id != "as" {
		t.Fatalf("final recap facilitator = %q, want Assistant (as)", id)
	}
}
