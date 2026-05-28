package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestNormalizeAndValidateTasksForExecution_DropsWeakAndDuplicates(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "a1", AgentName: "Assistant", AgentType: protocol.AgentTypeAssistant},
		{AgentID: "g1", AgentName: "Gemini", AgentType: protocol.AgentTypeCLI},
	}
	plan := `## Tasks

- Task 1: @Gemini - Write collabs/abc/findings.md with schema notes
- Task 2: @Gemini - Write collabs/abc/findings.md with schema summary
- document findings
- Task 3: @Assistant - Review current schema
`
	c := &Collaboration{
		ID:          "abc",
		Description: "Produce a markdown document",
		Agents:      agents,
		Plan:        &SharedArtifact{Content: plan},
	}
	tasks, warnings := NormalizeAndValidateTasksForExecution(c)
	if len(tasks) > 3 {
		t.Fatalf("expected merged/capped tasks, got %d: %#v", len(tasks), tasks)
	}
	if len(warnings) == 0 {
		t.Fatal("expected validation warnings")
	}
	combined := strings.Join(warnings, " ")
	if !strings.Contains(combined, "Dropped") {
		t.Fatalf("expected drop warning, got %v", warnings)
	}
}

func TestWarnMissingFileDeliverableTasks(t *testing.T) {
	c := &Collaboration{
		Description: "Write a markdown document for the API",
		Agents: []CollaborationAgent{
			{AgentID: "a1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		},
		Plan: &SharedArtifact{Content: "- Task 1: @SoftwareArchitect - Draft outline in chat only"},
	}
	_, warnings := NormalizeAndValidateTasksForExecution(c)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "concrete path") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected file-path warning, got %v", warnings)
	}

	c2 := &Collaboration{
		Description: "Write a markdown document",
		Agents:      c.Agents,
		Plan:        &SharedArtifact{Content: "- Task 1: @SoftwareArchitect - Write collabs/abc/schema.md"},
	}
	_, warnings2 := NormalizeAndValidateTasksForExecution(c2)
	for _, w := range warnings2 {
		if strings.Contains(w, "concrete path") {
			t.Fatalf("unexpected file-path warning when task has path: %v", warnings2)
		}
	}
}
