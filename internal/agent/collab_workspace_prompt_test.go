package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestAppendCollaborationWorkspaceInstructionsWarnsWithoutRepo(t *testing.T) {
	var b strings.Builder
	appendCollaborationWorkspaceInstructions(&b, CollaborationInfo{}, protocol.AgentTypeGeneral)
	out := b.String()
	if !strings.Contains(out, "No user project workspace") {
		t.Fatalf("expected missing workspace warning, got:\n%s", out)
	}
}

func TestAppendCollaborationWorkspaceInstructionsDevOpsWithoutK8s(t *testing.T) {
	var b strings.Builder
	appendCollaborationWorkspaceInstructions(&b, CollaborationInfo{
		ID:             "abc-123",
		Description:    "Standardize API schemas",
		SourceRepoPath: "/tmp/project",
		SourceWorkspaceContext: map[string]interface{}{
			"file_tree": "main.go\nREADME.md\n",
		},
	}, protocol.AgentTypeDevOps)
	out := b.String()
	if !strings.Contains(out, "collabs/abc-123") {
		t.Fatalf("expected deliverables path, got:\n%s", out)
	}
	if !strings.Contains(out, "does not show Kubernetes") {
		t.Fatalf("expected k8s guardrail, got:\n%s", out)
	}
	if !strings.Contains(out, "Collaboration goal") {
		t.Fatalf("expected goal-first instructions, got:\n%s", out)
	}
	if !strings.Contains(out, "Grounding: I loaded") {
		t.Fatalf("expected anti-grounding preamble rule, got:\n%s", out)
	}
}

func TestCollaborationTurnHandoffBodyByPhase(t *testing.T) {
	if got := collaborationTurnHandoffBody("planning"); !strings.Contains(got, "plan discussion") {
		t.Fatalf("planning handoff: %q", got)
	}
	if got := collaborationTurnHandoffBody("executing"); !strings.Contains(got, "assigned task") {
		t.Fatalf("executing handoff: %q", got)
	}
	if collaborationTurnHandoffBody("reviewing") != "" {
		t.Fatal("reviewing should not emit planning handoffs")
	}
}
