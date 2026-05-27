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

func TestPersistCollaborationReviewAssetsWritesPlanAndRecaps(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })
	_ = h.CreateChannel("review-assets", "review assets", "tester")

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("persist review assets", []string{"a1", "a2"}, "review-assets", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	if err := cm.UpdateArtifact(collab.ID, "a1", "Gemini", "## Plan\n\n- Task 1: @Gemini - Review schemas"); err != nil {
		t.Fatalf("update artifact: %v", err)
	}
	if err := cm.CompletePlanningRecap(collab.ID, "a2", "## Planning summary\n\n- The plan is ready."); err != nil {
		t.Fatalf("planning recap: %v", err)
	}
	if err := cm.CompleteSessionRecap(collab.ID, "a2", "## Session summary\n\n- The work is complete."); err != nil {
		t.Fatalf("session recap: %v", err)
	}

	h.persistCollaborationReviewAssets(collab.ID)

	paths := collaboration.ReviewAssetPathsFor(assetsRoot, collab.ID)
	assertHubFileContains(t, paths.Plan, "## Plan")
	assertHubFileContains(t, paths.PlanningSummary, "## Planning summary")
	assertHubFileContains(t, paths.SessionSummary, "## Session summary")
	assertHubFileContains(t, paths.Index, "persist review assets")
}

func TestPersistCollaborationReviewAssetsWritesToProjectCollabDir(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })
	_ = h.CreateChannel("proj-collab", "proj collab", "tester")

	sourceWorkspace := t.TempDir()
	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("project assets", []string{"a1", "a2"}, "proj-collab", "tester", collaboration.DiscussionConfig{}, collaboration.CreateOptions{
		SourceRepoPath: sourceWorkspace,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := cm.UpdateArtifact(collab.ID, "a1", "Gemini", "## Plan\n\n- Task 1: @Gemini - Draft"); err != nil {
		t.Fatalf("artifact: %v", err)
	}
	h.persistCollaborationReviewAssets(collab.ID)

	paths := collaboration.CollabAssetPaths(collab, assetsRoot)
	wantDir := filepath.Join(sourceWorkspace, collaboration.ProjectCollabsDirName, collab.ID)
	if paths.Directory != wantDir {
		t.Fatalf("directory=%q want %q", paths.Directory, wantDir)
	}
	assertHubFileContains(t, paths.Plan, "## Plan")
}

func TestCollaborateCapturesOutboundWorkspaceContextAsSourceWorkspace(t *testing.T) {
	h := NewHub()
	ch, err := NewCommandHandler(h)
	if err != nil {
		t.Fatalf("command handler: %v", err)
	}
	h.commandHandler = ch

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	sourceWorkspace := t.TempDir()
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"general",
		protocol.AgentInfo{ID: "human", Name: "tester", Type: protocol.AgentTypeGeneral},
		"/collaborate @Gemini @Assistant investigate schemas",
	)
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{
			"workspace_name": "source",
			"workspace_path": sourceWorkspace,
			"file_tree":      "schema-definition.js\n",
			"open_files":     []interface{}{},
		},
	}

	if out, err := ch.handleCollaborate(context.Background(), msg, strings.Fields(msg.Content)); err != nil || out != nil {
		t.Fatalf("handleCollaborate out=%v err=%v", out, err)
	}

	active := h.GetCollaborationManager().ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active collaboration, got %d", len(active))
	}
	if got := active[0].SourceRepoPath; got != sourceWorkspace {
		t.Fatalf("source workspace = %q, want %q", got, sourceWorkspace)
	}

	if active[0].ExecutionMode == collaboration.ExecutionModeWorktree {
		t.Fatal("plain /collaborate should keep sandbox execution mode")
	}
}

func TestSandboxTaskPromptUsesSourceWorkspaceAndOutputPath(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })
	_ = h.CreateChannel("source-context", "source context", "tester")

	sourceWorkspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceWorkspace, "schema-definition.js"), []byte("export const schema = {};\n"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "Assistant", Type: protocol.AgentTypeAssistant, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("inspect source", []string{"a1", "a2"}, "source-context", "tester", collaboration.DiscussionConfig{}, collaboration.CreateOptions{
		SourceRepoPath: sourceWorkspace,
	})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	task := collaboration.CollaborationTask{
		ID:           "task-1",
		Title:        "Inspect source",
		Description:  "Inspect source schemas",
		AssignedTo:   "a1",
		AssignedName: "Gemini",
		Status:       collaboration.TaskPending,
	}
	if err := cm.SetTasks(collab.ID, []collaboration.CollaborationTask{task}); err != nil {
		t.Fatalf("set tasks: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	if err := cm.CompletePlanningRecap(collab.ID, "a2", "ready"); err != nil {
		t.Fatalf("planning recap: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("executing: %v", err)
	}
	if _, _, err := cm.AcknowledgeWorkspace(collab.ID); err != nil {
		t.Fatalf("ack workspace: %v", err)
	}
	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	h.dispatchCollabTaskMessages(snap, nil, false)

	messages, _ := h.GetMessages("source-context", 50)
	var taskMsg *protocol.Message
	for _, m := range messages {
		if m != nil && m.Type == protocol.MessageTypeCollabTask {
			taskMsg = m
			break
		}
	}
	if taskMsg == nil {
		t.Fatal("expected collaboration task message")
	}
	ctx := workspaceContextFromMessage(taskMsg)
	if ctx == nil {
		t.Fatal("expected task workspace_context")
	}
	if got := ctx["workspace_path"]; got != sourceWorkspace {
		t.Fatalf("workspace_path=%v want %s", got, sourceWorkspace)
	}
	if got := ctx["collaboration_output_path"]; got != snap.WorkingDirectory {
		t.Fatalf("collaboration_output_path=%v want %s", got, snap.WorkingDirectory)
	}
	wantOut := filepath.Join(sourceWorkspace, collaboration.ProjectCollabsDirName, collab.ID)
	if snap.WorkingDirectory != wantOut {
		t.Fatalf("working_directory=%q want %q", snap.WorkingDirectory, wantOut)
	}
	if got, _ := ctx["review_assets_path"].(string); got != wantOut {
		t.Fatalf("review_assets_path=%q want %q", got, wantOut)
	}
	if !strings.Contains(taskMsg.Content, "Write deliverables under") {
		t.Fatalf("task prompt missing source/output guidance:\n%s", taskMsg.Content)
	}
}

func TestExecutionReplyDoesNotOverwriteApprovedPlanArtifact(t *testing.T) {
	h := NewHub()
	assetsRoot := t.TempDir()
	h.SetCollaborationAssetsRootResolver(func() string { return assetsRoot })

	a1 := &protocol.AgentInfo{ID: "a1", Name: "Gemini", Type: protocol.AgentTypeCLI, Status: "active"}
	a2 := &protocol.AgentInfo{ID: "a2", Name: "SoftwareArchitect", Type: protocol.AgentTypeArchitecture, Status: "active"}
	_ = h.RegisterAgent(a1)
	_ = h.RegisterAgent(a2)

	cm := h.GetCollaborationManager()
	collab, err := cm.CreateCollaboration("protect plan", []string{"a1", "a2"}, "general", "tester", collaboration.DiscussionConfig{})
	if err != nil {
		t.Fatalf("create collaboration: %v", err)
	}
	chName := "collab-" + collab.ID
	h.CreateChannelWithType(chName, "collab", "general", protocol.ChannelTypeCollaboration, "tester")
	if err := cm.BindCollaborationChannel(collab.ID, chName); err != nil {
		t.Fatalf("bind channel: %v", err)
	}

	approvedPlan := "## Plan\n\n- Task 1: @Gemini - Review current schema\n- Task 2: @SoftwareArchitect - Document standards"
	if err := cm.UpdateArtifact(collab.ID, "a1", "Gemini", approvedPlan); err != nil {
		t.Fatalf("update artifact: %v", err)
	}
	tasks := collaboration.ExtractTasksFromPlan(approvedPlan, collab.Agents)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 extracted tasks, got %d", len(tasks))
	}
	if err := cm.SetTasks(collab.ID, tasks); err != nil {
		t.Fatalf("set tasks: %v", err)
	}
	if _, err := cm.TransitionToReviewing(collab.ID); err != nil {
		t.Fatalf("reviewing: %v", err)
	}
	if err := cm.CompletePlanningRecap(collab.ID, "a2", "ready"); err != nil {
		t.Fatalf("planning recap: %v", err)
	}
	if _, err := cm.ApprovePlan(collab.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := cm.TransitionToExecuting(collab.ID); err != nil {
		t.Fatalf("executing: %v", err)
	}

	reply := protocol.NewMessage(
		protocol.MessageTypeCollabDiscussion,
		chName,
		*a1,
		"### Task 1: Understand Current Schema Implementation\n\nThis task is blocked because the workspace is empty.",
	)
	reply.SetCollaborationID(collab.ID)
	h.maybeIngestPlanArtifact(reply, collab.ID)

	snap, err := cm.GetCollaborationSnapshot(collab.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := strings.TrimSpace(snap.Plan.Content); got != approvedPlan {
		t.Fatalf("plan was overwritten:\n%s", got)
	}
	if len(snap.Tasks) != 2 {
		t.Fatalf("tasks were overwritten, got %d", len(snap.Tasks))
	}
}

func assertHubFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, string(data))
	}
}

func workspaceContextFromMessage(m *protocol.Message) map[string]interface{} {
	if m == nil || m.Metadata == nil {
		return nil
	}
	ctx, _ := m.Metadata["workspace_context"].(map[string]interface{})
	return ctx
}
