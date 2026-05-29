package agent

import (
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestCollaborationProactiveWorkspaceScanSkipsCollabTask(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabTask, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "assigned task")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if !collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("collaboration_task should trigger targeted workspace scan")
	}
	if collaborationWorkspaceGroundingLine(msg, info) {
		t.Fatal("collaboration_task should not force grounding opener")
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

func TestCollaborationProactiveWorkspaceScanAllowsExplicitReview(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeCollabDiscussion, "collab-ch", protocol.AgentInfo{ID: "a1", Name: "Arch"}, "please review internal/hub/hub.go")
	msg.SetCollaborationID("cid")
	info := CollaborationInfo{ID: "cid", Phase: "executing"}
	if !collaborationProactiveWorkspaceScan(msg, info) {
		t.Fatal("explicit file review should allow scan")
	}
}
