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

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snap.WorkspaceAcknowledged {
		t.Fatal("expected workspace auto-acknowledged for sandbox with bound repo")
	}
	if !snap.TasksDispatched {
		t.Fatal("expected tasks dispatched after auto-ack on approve")
	}
	msgs, _ := h.GetMessages("path-warn", 100)
	if countMessageType(msgs, protocol.MessageTypeCollabTask) == 0 {
		t.Fatal("expected collaboration_task messages after approve with bound repo")
	}
	if !strings.Contains(out.Content, "auto-confirmed") {
		t.Fatalf("expected auto-confirmed notice, got: %s", out.Content)
	}
}

func TestApprovePlan_dropsDependencyProseTasks(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	_ = h.CreateChannel("dep-prose", "dep prose", "tester")

	a1 := &protocol.AgentInfo{ID: "be-1", Name: "BackendEngineer", Type: protocol.AgentTypeBackend, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "arch-1", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	a3 := &protocol.AgentInfo{ID: "plat-1", Name: "PlatformEngineer", Type: protocol.AgentTypeDevOps, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)
	_ = h.RegisterAgent(a3)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration(
		"schema docs",
		[]string{"be-1", "arch-1", "plat-1"},
		"dep-prose",
		"tester",
		collaboration.DiscussionConfig{},
	)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	plan := `## Plan

Task 1: @BackendEngineer - Write collabs/test/api_schema.md defining the API schema.
Task 2: @SoftwareArchitect - Write collabs/test/markdown_doc_structure.md detailing structure.
Task 3: @PlatformEngineer - Write collabs/test/ci_cd_pipeline.md outlining CI/CD.
- Task 1 depends on Task 2 for the markdown structure and style guide.
- Task 3 can be started independently but should reference the schema documents.
`
	if err := cm.UpdateArtifact(collab.ID, "arch-1", "SoftwareArchitect", plan); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	completePlanningRecapForHubTest(t, cm, collab.ID)

	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dep-prose",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/approve-plan "+collab.ID[:8],
	)
	if _, err := ch.handleApprovePlan(context.Background(), msg, strings.Fields(msg.Content)); err != nil {
		t.Fatalf("handleApprovePlan: %v", err)
	}
	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.Tasks) != 3 {
		t.Fatalf("expected 3 tasks after approve, got %d: %#v", len(snap.Tasks), snap.Tasks)
	}
	for _, task := range snap.Tasks {
		lower := strings.ToLower(task.Title + " " + task.Description)
		if strings.Contains(lower, "depends on") || strings.Contains(lower, "can be started") {
			t.Fatalf("dependency prose became task: %+v", task)
		}
	}
}
