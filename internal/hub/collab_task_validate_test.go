package hub

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestApprovePlanWarnsOnMissingTaskPaths(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	_ = h.CreateChannel("path-warn", "path warn", "tester")

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "core/resource-api"), 0o755); err != nil {
		t.Fatal(err)
	}

	a1 := &protocol.AgentInfo{ID: "a1", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"schema review",
		[]string{"a1", "a2"},
		"path-warn",
		"tester",
		collaboration.DiscussionConfig{},
		collaboration.CreateOptions{SourceRepoPath: root},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := cm.UpdateArtifact(collab.ID, "a1", "SoftwareArchitect", "## Plan\n\n- Task 1: @SoftwareArchitect - Review docs/api/ schemas\n"); err != nil {
		t.Fatalf("plan: %v", err)
	}
	tasks := collaboration.ExtractTasksFromPlan(collab.Plan.Content, collab.Agents)
	if err := cm.SetTasks(collab.ID, tasks); err != nil {
		t.Fatalf("tasks: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	completePlanningRecapForHubTest(t, cm, collab.ID)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"path-warn",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/approve-plan "+collab.ID[:8],
	)
	out, err := ch.handleApprovePlan(context.Background(), msg, strings.Fields(msg.Content))
	if err != nil {
		t.Fatalf("handleApprovePlan: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "Path check") {
		t.Fatalf("expected path warning in response, got: %s", out.Content)
	}
	if !strings.Contains(out.Content, "docs/api") {
		t.Fatalf("expected docs/api in warning, got: %s", out.Content)
	}
}
