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

func TestCollabPlanningSuppressMCPTools_DevOpsExecutionDocCollab(t *testing.T) {
	info := CollaborationInfo{
		Phase: "executing",
		SourceWorkspaceContext: map[string]interface{}{
			"file_tree": "resource-api/json_endpoints/\ncollabs/abc/api_schema.md\n",
		},
	}
	if !collabPlanningSuppressMCPTools(info, protocol.AgentTypeDevOps) {
		t.Fatal("expected MCP tools suppressed during executing doc collab without k8s assets")
	}
}

func TestIsCollabTurnPromptForAgent_MentionOnly(t *testing.T) {
	stub := collabSystemTurnStub{agentID: "assistant-id"}
	msg := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		"collab-test",
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Collaboration turn handoff: next participant, please continue the plan discussion and refine task assignments.",
	)
	msg.SetCollaborationID("550e8400-e29b-41d4-a716-446655440000")
	msg.Mentions = []string{"assistant-id"}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["collab_internal_event"] = true

	if !isCollabTurnPromptForAgent(msg, msg.GetCollaborationID(), "assistant-id", stub) {
		t.Fatal("mentioned agent should match turn handoff prompt")
	}
	if isCollabTurnPromptForAgent(msg, msg.GetCollaborationID(), "moderator-id", stub) {
		t.Fatal("non-mentioned agent must not match turn handoff prompt when IsAgentTurn is true")
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
