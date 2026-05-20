package collaboration

import (
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func setCollabPhaseExecuting(cm *CollaborationManager, collabID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if c := cm.collaborations[collabID]; c != nil {
		c.Phase = PhaseExecuting
	}
}

func TestUpdateTaskStatusWithEffectsFailRun(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("x", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	now := time.Now()
	_ = cm.SetTasks(c.ID, []CollaborationTask{
		{ID: "t1", Title: "One", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
	})
	setCollabPhaseExecuting(cm, c.ID)
	cm.mu.Lock()
	cm.collaborations[c.ID].ExecutionPolicy.BlockedUpstreamPolicy = BlockedPolicyFailRun
	cm.mu.Unlock()

	effects, err := cm.UpdateTaskStatusWithEffects(c.ID, "t1", TaskBlocked, "stuck")
	if err != nil {
		t.Fatal(err)
	}
	if !effects.ShouldFailRun {
		t.Fatal("expected ShouldFailRun")
	}
	if effects.ShouldDispatchWave {
		t.Fatal("blocked should not dispatch wave")
	}
}

func TestUpdateTaskStatusWithEffectsDispatchOnComplete(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("x", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	now := time.Now()
	_ = cm.SetTasks(c.ID, []CollaborationTask{{ID: "t1", Title: "One", Status: TaskPending, CreatedAt: now, UpdatedAt: now}})
	setCollabPhaseExecuting(cm, c.ID)

	effects, err := cm.UpdateTaskStatusWithEffects(c.ID, "t1", TaskCompleted, "done")
	if err != nil {
		t.Fatal(err)
	}
	if !effects.ShouldDispatchWave {
		t.Fatal("expected ShouldDispatchWave when executing")
	}
}

func TestReassignTaskClearsPromptDispatched(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	h.addAgent("a2", "B", protocol.AgentTypeFrontend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	now := time.Now()
	_, err := cm.UpdateRunbook(c.ID, RunbookUpdatePayload{Tasks: []CollaborationTask{
		{ID: "t1", Title: "T", AssignedTo: "a1", AssignedName: "A", Status: TaskInProgress, PromptDispatched: true, CreatedAt: now, UpdatedAt: now},
	}})
	if err != nil {
		t.Fatal(err)
	}

	snap, err := cm.ReassignTask(c.ID, "t1", "a2")
	if err != nil {
		t.Fatal(err)
	}
	var task *CollaborationTask
	for i := range snap.Tasks {
		if snap.Tasks[i].ID == "t1" {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		t.Fatal("task not found")
	}
	if task.AssignedTo != "a2" || task.PromptDispatched || task.Status != TaskPending {
		t.Fatalf("after reassign: assignee=%s dispatched=%v status=%s", task.AssignedTo, task.PromptDispatched, task.Status)
	}
}

func TestSetDispatchPaused(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})

	snap, err := cm.SetDispatchPaused(c.ID, true)
	if err != nil || !snap.DispatchPaused {
		t.Fatalf("pause: err=%v paused=%v", err, snap.DispatchPaused)
	}
	snap, err = cm.SetDispatchPaused(c.ID, false)
	if err != nil || snap.DispatchPaused {
		t.Fatalf("resume: err=%v paused=%v", err, snap.DispatchPaused)
	}
}

func TestApproveTaskDispatchClearsGate(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("rb", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	now := time.Now()
	if _, err := cm.UpdateRunbook(c.ID, RunbookUpdatePayload{Tasks: []CollaborationTask{
		{ID: "t1", Title: "Gated", Status: TaskPending, AwaitingApproval: true, Options: &TaskExecutionOptions{RequiresApproval: true}, CreatedAt: now, UpdatedAt: now},
	}}); err != nil {
		t.Fatal(err)
	}

	snap, err := cm.ApproveTaskDispatch(c.ID, "t1")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range snap.Tasks {
		if task.ID == "t1" && task.AwaitingApproval {
			t.Fatal("approval should clear awaiting_approval")
		}
	}
}

func TestNormalizeTaskOnSaveRequiresApproval(t *testing.T) {
	task := CollaborationTask{Options: &TaskExecutionOptions{RequiresApproval: true}}
	normalizeTaskOnSave(&task)
	if !task.AwaitingApproval {
		t.Fatal("expected awaiting_approval")
	}
	if task.Kind != TaskKindAgent {
		t.Fatalf("kind = %q", task.Kind)
	}
}

func TestReadyTasksForCollabSkipsAwaitingApproval(t *testing.T) {
	now := time.Now()
	tasks := []CollaborationTask{
		{ID: "t1", Status: TaskPending, AwaitingApproval: true, Options: &TaskExecutionOptions{RequiresApproval: true}, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
	}
	c := &Collaboration{Tasks: tasks, Phase: PhaseExecuting}
	ready := ReadyTasksForCollab(c)
	if len(ready) != 1 || ready[0].ID != "t2" {
		t.Fatalf("ready = %#v", ready)
	}
}

func TestSkipBranchMarksDownstreamSkipped(t *testing.T) {
	h := newRunbookMockHub()
	h.addAgent("a1", "A", protocol.AgentTypeBackend, nil)
	cm := NewCollaborationManager(h)
	c, _ := cm.CreateRunbook("x", []string{"a1"}, "general", "u", DiscussionConfig{}, CreateOptions{})
	now := time.Now()
	_ = cm.SetTasks(c.ID, []CollaborationTask{
		{ID: "t1", Title: "Up", Status: TaskPending, CreatedAt: now, UpdatedAt: now},
		{ID: "t2", Title: "Down", Status: TaskPending, Dependencies: []string{"t1"}, CreatedAt: now, UpdatedAt: now},
	})
	cm.mu.Lock()
	cm.collaborations[c.ID].ExecutionPolicy.BlockedUpstreamPolicy = BlockedPolicySkipBranch
	cm.mu.Unlock()
	_, _ = cm.UpdateTaskStatusWithEffects(c.ID, "t1", TaskBlocked, "blocked")

	snap, _ := cm.GetCollaborationSnapshot(c.ID)
	for _, task := range snap.Tasks {
		if task.ID == "t2" && !task.SkippedDueToBlocked {
			t.Fatal("downstream should be marked skipped_due_to_blocked")
		}
	}
}
