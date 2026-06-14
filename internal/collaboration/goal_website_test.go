package collaboration

import "testing"

func TestGoalLooksLikeWebsiteBuild(t *testing.T) {
	cases := map[string]bool{
		"I want you to make me a website":                    true,
		"Design a website, colors black white gray blue red": true,
		"Three pages: home page, about page, contact page":   true,
		"Investigate API schema standardization":             false,
		"Write findings.md about main.go":                    false,
	}
	for goal, want := range cases {
		if got := GoalLooksLikeWebsiteBuild(goal); got != want {
			t.Errorf("GoalLooksLikeWebsiteBuild(%q) = %v want %v", goal, got, want)
		}
	}
}

func TestValidatePlanForCollaboration_rejectsSpecsOnlyWebsitePlan(t *testing.T) {
	c := &Collaboration{
		Description: "Design a website with home, about, and contact pages",
		Agents:      websiteCollabAgents(),
	}
	plan := `## Plan

- Task 1: @SoftwareArchitect - Write collabs/x/setup.md defining tech stack
- Task 2: @FrontendEngineer - Write collabs/x/ui-spec.md color palette
- Task 3: @Assistant - Write collabs/x/pages-wireframe.md wireframes`
	ok, reason := ValidatePlanForCollaboration(c, plan)
	if ok {
		t.Fatal("expected specs-only website plan to be rejected")
	}
	if reason == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestValidatePlanForCollaboration_acceptsWebsiteWithHTML(t *testing.T) {
	c := &Collaboration{
		Description: "Make me a website called Collaboration Station",
		Agents:      websiteCollabAgents(),
	}
	plan := `## Plan

- Task 1: @FrontendEngineer - Write collabs/x/plan.md architecture notes
- Task 2: @Gemini - Create collabs/x/index.html, about.html, contact.html, and style.css`
	ok, reason := ValidatePlanForCollaboration(c, plan)
	if !ok {
		t.Fatalf("expected HTML plan to pass, reason=%q", reason)
	}
}

func TestWarnWebsitePlanMissingHTML(t *testing.T) {
	c := &Collaboration{Description: "Build a website with 3 pages"}
	tasks := []CollaborationTask{
		{Description: "Write collabs/x/setup.md"},
	}
	if w := WarnWebsitePlanMissingHTML(c, tasks); w == "" {
		t.Fatal("expected warning for specs-only tasks")
	}
	tasks = []CollaborationTask{
		{Description: "Create collabs/x/index.html and style.css"},
	}
	if w := WarnWebsitePlanMissingHTML(c, tasks); w != "" {
		t.Fatalf("unexpected warning: %s", w)
	}
}
