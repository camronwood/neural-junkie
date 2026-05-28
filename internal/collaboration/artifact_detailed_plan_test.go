package collaboration

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestExtractTasksFromPlanIgnoresDetailedPlanSection(t *testing.T) {
	agents := []CollaborationAgent{
		{AgentID: "arch-1", AgentName: "SoftwareArchitect", AgentType: protocol.AgentTypeArchitecture},
		{AgentID: "gem-1", AgentName: "Gemini", AgentType: protocol.AgentTypeCLI},
		{AgentID: "asst-1", AgentName: "Assistant", AgentType: protocol.AgentTypeAssistant},
	}

	planContent := `#### Task Breakdown and Initial Assignments

1. **Review Current Schema**
   - **@SoftwareArchitect**: Inspect docs/api/schema.yml and docs/api/resource-api.md.

2. **Research Standardization Practices**
   - **@Gemini**: Research API schema standards.

3. **Propose and Validate Schema Changes**
   - **@Assistant**: Draft docs/api/new-schema.yml.

### Detailed Plan

#### Task 1: Review Current Schema (SoftwareArchitect)

1. **Files to Review**:
   - docs/api/schema.yml

2. **Specific Actions**:
   - Check schema definitions

3. **Example Actions**:
   - Line 10: Check data types
`

	tasks := ExtractTasksFromPlan(planContent, agents)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks from overview, got %d: %#v", len(tasks), tasks)
	}
	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Title), "specific actions") {
			t.Fatalf("unexpected nested heading as task: %q", task.Title)
		}
	}
}
