package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollaborationProactiveWorkspaceScanSkipsFocusScopedCollabTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "assigned task")
	msg.SetCollaborationID("cid")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata[MetadataContextScope] = ContextScopeFocus
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("focus-scoped collaboration_task should not bulk-scan the repo")
	}
	if collaborationWorkspaceGroundingLine(msg, info) {
		t.Fatal("collaboration_task should not force grounding opener")
	}
}

func TestCollaborationProactiveWorkspaceScanAllowsCollabTaskWithoutFocusScope(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "assigned task")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if !collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("collaboration_task without focus scope may scan")
	}
}

func TestCollaborationWorkspaceGroundingLineDisabledForCollab(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "task work")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if collaborationWorkspaceGroundingLine(msg, info) {
		t.Fatal("collaboration should not require grounding opener")
	}
}

func TestCollaborationPlanningSkipsScanEvenWithPathTokens(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "BE"}, "Write collabs/x/findings.md summarizing README.md and core/sample/main.go only")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{
		ID:                     "cid",
		Phase:                  "planning",
		SourceWorkspaceContext: map[string]interface{}{"file_tree": "README.md\ncore/sample/main.go\n"},
	}
	if collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("planning should not bulk-scan when task text mentions .go paths")
	}
}

func TestCollaborationPlanningWithoutWorkspaceSkipsScanAndGrounding(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "BE"}, "Write collabs/x/findings.md")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "planning"}
	if collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("planning without bound repo should not scan open editor tree")
	}
	if collaborationWorkspaceGroundingLine(msg, info) {
		t.Fatal("planning without bound repo should not force grounding line")
	}
}

func TestCollaborationPlanningWithBoundRepoSkipsGroundingLine(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "BE"}, "Review internal/hub/hub.go for schema")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{
		ID:                     "cid",
		Phase:                  "planning",
		SourceWorkspaceContext: map[string]interface{}{"README.md": "hello"},
	}
	if collaborationWorkspaceGroundingLine(msg, info) {
		t.Fatal("planning with bound repo should not force grounding line")
	}
}

func TestCollaborationRestrictsDiscoveryToolsForFocusScope(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "BE"}, "Write findings.md")
	msg.SetCollaborationID("cid")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata[MetadataContextScope] = ContextScopeFocus
	if !collaborationRestrictsDiscoveryTools(msg) {
		t.Fatal("focus-scoped collab task should restrict discovery tools")
	}
	allow := effectiveMCPToolAllowlist(&Agent{}, msg)
	for _, name := range allow {
		if collaborationDiscoveryToolNames[name] {
			t.Fatalf("discovery tool %q must not be allowlisted for focus scope", name)
		}
	}
	foundRead := false
	for _, name := range allow {
		if name == "read_file" {
			foundRead = true
		}
	}
	if !foundRead {
		t.Fatalf("expected read_file in focus allowlist, got %v", allow)
	}
	if !collaborationDiscoveryToolBlocked(msg, "list_dir") {
		t.Fatal("list_dir should be blocked for focus-scoped task")
	}
	open := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "BE"}, "Implement API")
	open.SetCollaborationID("cid")
	if collaborationRestrictsDiscoveryTools(open) {
		t.Fatal("collab task without focus scope should allow discovery tools")
	}
}

func TestCollaborationProactiveWorkspaceScanAllowsExplicitReview(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "please review internal/hub/hub.go")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if !collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("explicit file review should allow scan")
	}
}

func TestCollaborationFocusReadPathAllowed(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "be", Name: "BackendEngineer"}, "Write findings")
	msg.SetCollaborationID("cid")
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata[MetadataContextScope] = ContextScopeFocus
	msg.Metadata["task_context_paths"] = []string{"README.md", "core/sample/main.go"}

	if !collaborationFocusReadPathAllowed(msg, "", "README.md") {
		t.Fatal("README.md should be allowed")
	}
	if !collaborationFocusReadPathAllowed(msg, "", "core/sample/main.go") {
		t.Fatal("core/sample/main.go should be allowed")
	}
	if collaborationFocusReadPathAllowed(msg, "", "core/server/main.go") {
		t.Fatal("core/server/main.go must be blocked under focus scope")
	}
	if collaborationFocusReadPathAllowed(msg, "", "src/App.tsx") {
		t.Fatal("src/App.tsx must be blocked under focus scope")
	}
	if !collaborationFocusReadPathAllowed(msg, "", "collabs/abc/findings.md") {
		t.Fatal("collabs deliverable path should be allowed")
	}

	open := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "be", Name: "BE"}, "Implement")
	open.SetCollaborationID("cid")
	if !collaborationFocusReadPathAllowed(open, "", "core/server/main.go") {
		t.Fatal("non-focus task should allow arbitrary reads")
	}
}
