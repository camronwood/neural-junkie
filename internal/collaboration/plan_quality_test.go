package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func websiteCollabAgents() []CollaborationAgent {
	return []CollaborationAgent{
		{AgentID: "fe-1", AgentName: "FrontendEngineer", AgentType: protocol.AgentTypeFrontend},
		{AgentID: "g-1", AgentName: "Gemini", AgentType: protocol.AgentTypeCLI},
		{AgentID: "be-1", AgentName: "BackendEngineer", AgentType: protocol.AgentTypeBackend},
	}
}

func TestValidatePlanContent_rejectsCorruptV4(t *testing.T) {
	plan := "## Plan\n\n- Task 2 consolidates all code implementation to @Gemini's Implementation & Code lane to avoid duplicate work.\n- Task 2 consolidates all code implementation to @Gemini's Implementation & Code lane to avoid duplicate work."
	ok, reason := ValidatePlanContent(plan, websiteCollabAgents())
	if ok {
		t.Fatal("expected corrupt v4 plan to be rejected")
	}
	if reason == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestValidatePlanContent_acceptsV3ThreeTaskPlan(t *testing.T) {
	plan := `## Plan

1.  Task 1: @FrontendEngineer - Write collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/frontend_architecture_plan.md defining the HTML structure, CSS palette (black/white/gray/blue/red), and component strategy.
    - depends: none
2.  Task 2: @Gemini - Create collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/index.html, collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/about.html, collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/contact.html, and collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/style.css with content and layout based on the frontend_architecture_plan.md.
    - depends: 1
3.  Task 3: @FrontendEngineer - Write collabs/b222bffe-39e8-4b00-91ca-ee1c555b9592/findings.md reviewing the implementation consistency, styling alignment, and cross-browser basics.
    - depends: 2`
	ok, _ := ValidatePlanContent(plan, websiteCollabAgents())
	if !ok {
		t.Fatal("expected v3 plan to pass validation")
	}
}

func TestSynthesizePlanFromDiscussion_prefersV3OverMergedUnion(t *testing.T) {
	v1 := `## Plan

- Task 1: @FrontendEngineer - Write collabs/b222bffe/frontend_architecture_plan.md outlining HTML structure.
- Task 2: @Gemini - Create collabs/b222bffe/index.html and style.css based on the plan in Task 1.
- Task 3: @Gemini - Populate collabs/b222bffe/index.html with placeholder text.
- Task 4: @Gemini - Populate collabs/b222bffe/about.html with placeholder text.
- Task 5: @Gemini - Populate collabs/b222bffe/contact.html with placeholder text.
- Task 6: @FrontendEngineer - Review the implemented pages.`
	v3 := `## Plan

1.  Task 1: @FrontendEngineer - Write collabs/b222bffe/frontend_architecture_plan.md defining HTML structure and CSS palette.
    - depends: none
2.  Task 2: @Gemini - Create collabs/b222bffe/index.html, about.html, contact.html, and style.css based on frontend_architecture_plan.md.
    - depends: 1
3.  Task 3: @FrontendEngineer - Write collabs/b222bffe/findings.md reviewing implementation consistency.
    - depends: 2`
	v2extra := `## Plan

1. Task 2: @FrontendEngineer - Create collabs/b222bffe/index.html, about.html, contact.html, and style.css with full content.
2. Task 3: @FrontendEngineer - Review and validate implementation in findings.md.`

	agents := websiteCollabAgents()
	c := &Collaboration{
		Agents: agents,
		Discussion: &DiscussionSession{
			Messages: []*protocol.Message{
				{From: protocol.AgentInfo{Name: "Gemini"}, Content: v1},
				{From: protocol.AgentInfo{Name: "BackendEngineer"}, Content: v2extra},
				{From: protocol.AgentInfo{Name: "Gemini"}, Content: v3},
			},
		},
	}
	plan, tasks := SynthesizePlanFromDiscussion(c)
	if !strings.Contains(plan, "findings.md") {
		t.Fatalf("expected v3 plan, got: %s", plan)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks from v3, got %d: %#v", len(tasks), tasks)
	}
}

func TestMergeConflictingDeliverableTasks_dropsCrossAssigneeDupes(t *testing.T) {
	tasks := []CollaborationTask{
		{AssignedName: "Gemini", Title: "Create index.html", Description: "Create collabs/x/index.html based on plan"},
		{AssignedName: "FrontendEngineer", Title: "Create index too", Description: "Create collabs/x/index.html with layout"},
		{AssignedName: "FrontendEngineer", Title: "Write plan", Description: "Write collabs/x/frontend_architecture_plan.md"},
	}
	out, warnings := mergeConflictingDeliverableTasks(tasks)
	if len(out) != 2 {
		t.Fatalf("expected 2 tasks after conflict merge, got %d: %#v", len(out), out)
	}
	if len(warnings) == 0 {
		t.Fatal("expected conflict warning")
	}
}

func TestInferDepsFromTaskDescriptions_basedOnPlanInTask1(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", AssignedName: "FrontendEngineer", Description: "Write collabs/x/plan.md"},
		{ID: "t2", AssignedName: "Gemini", Description: "Create collabs/x/index.html based on the plan in Task 1"},
	}
	inferDepsFromTaskDescriptions(tasks)
	NormalizeDependencies(tasks)
	if len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != "t1" {
		t.Fatalf("task 2 should depend on task 1, deps=%v", tasks[1].Dependencies)
	}
}
