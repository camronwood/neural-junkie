package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollabPlanningSuppressMCPTools_DevOpsNoK8sTree(t *testing.T) {
	info := CollaborationInfo{
		Phase: "planning",
		SourceWorkspaceContext: map[string]interface{}{
			"file_tree": "README.md\ncore/sample/main.go\n",
		},
	}
	if !collabPlanningSuppressMCPTools(info, protocol.AgentTypeDevOps) {
		t.Fatal("expected MCP tools suppressed for devops doc-only tree")
	}
	if collabPlanningSuppressMCPTools(info, protocol.AgentTypeArchitecture) {
		t.Fatal("architect has no MCP catalog to suppress")
	}
}

func TestCollabPlanningSuppressMCPTools_DevOpsWithK8s(t *testing.T) {
	info := CollaborationInfo{
		Phase: "planning",
		SourceWorkspaceContext: map[string]interface{}{
			"file_tree": "deploy/helm/Chart.yaml\nk8s/deployment.yaml\n",
		},
	}
	if collabPlanningSuppressMCPTools(info, protocol.AgentTypeDevOps) {
		t.Fatal("expected MCP tools available when tree signals kubernetes")
	}
}

func TestSanitizeCollabDiscussionResponse_ToolJSON(t *testing.T) {
	raw := "```json\n{\"name\": \"kubectl_query\", \"arguments\": {\"namespace\": \"default\"}}\n```"
	out := sanitizeCollabDiscussionResponse(raw, CollaborationInfo{Phase: "planning"}, protocol.AgentTypeDevOps)
	if strings.Contains(out, "kubectl") || strings.Contains(out, `"arguments"`) {
		t.Fatalf("expected replacement prose, got %q", out)
	}
	if !strings.Contains(out, "Task N") {
		t.Fatalf("expected task format hint, got %q", out)
	}
}
