package collaboration

import "testing"

func TestUpstreamTasksComplete_openDeps(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "A", Status: TaskInProgress},
		{ID: "t2", Title: "Compile findings", Status: TaskPending, Dependencies: []string{"t1"}},
	}
	if UpstreamTasksComplete(tasks[1], tasks, BlockedPolicyFailRun) {
		t.Fatal("expected false while upstream in progress")
	}
	tasks[0].Status = TaskCompleted
	if !UpstreamTasksComplete(tasks[1], tasks, BlockedPolicyFailRun) {
		t.Fatal("expected true when upstream completed")
	}
}

func TestUpstreamTasksComplete_noDeps(t *testing.T) {
	task := CollaborationTask{ID: "solo", Title: "Solo", Status: TaskPending}
	if !UpstreamTasksComplete(task, []CollaborationTask{task}, BlockedPolicyFailRun) {
		t.Fatal("expected true with no dependencies")
	}
}
