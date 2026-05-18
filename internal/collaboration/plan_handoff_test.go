package collaboration

import "testing"

func TestSyncTaskStatusFromPlanHandoff(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "One", Status: TaskPending},
		{ID: "t2", Title: "Two", Status: TaskInProgress},
	}
	content := `PLAN (V2)

Task 1 (@RustExpert) — Complete
Task 2 (@Cursor) - Complete
`
	updated := SyncTaskStatusFromPlanHandoff(content, tasks)
	if len(updated) != 2 {
		t.Fatalf("expected 2 updates, got %d (%v)", len(updated), updated)
	}
	if updated[0] != "t1" || updated[1] != "t2" {
		t.Fatalf("unexpected ids: %v", updated)
	}
}

func TestSyncTaskStatusFromPlanHandoffSkipsAlreadyComplete(t *testing.T) {
	tasks := []CollaborationTask{
		{ID: "t1", Title: "One", Status: TaskCompleted},
	}
	content := "Task 1 — Complete\n"
	updated := SyncTaskStatusFromPlanHandoff(content, tasks)
	if len(updated) != 0 {
		t.Fatalf("expected no updates, got %v", updated)
	}
}
