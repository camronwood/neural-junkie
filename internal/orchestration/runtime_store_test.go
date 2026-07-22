package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWorkerClaimsCompatibleQueuedTask(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	seedRunAndTasks(t, store, 2, Task{
		ID: "gpu-task", Status: TaskPending, Queue: "training",
		CapabilityTags: []string{"gpu"},
	})
	if err := store.RegisterWorker(ctx, Worker{
		ID: "worker", Queue: "training", Capabilities: []string{"gpu"}, Status: WorkerReady,
	}); err != nil {
		t.Fatal(err)
	}
	task, attempt, err := store.ClaimNextTask(ctx, ClaimOptions{
		WorkerID: "worker", Queue: "training", Capabilities: []string{"gpu"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != "gpu-task" || attempt.WorkerID != "worker" {
		t.Fatalf("task=%#v attempt=%#v", task, attempt)
	}
}

func TestWorkerCannotClaimMissingCapability(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{
		ID: "gpu-task", Status: TaskPending, Queue: "training",
		CapabilityTags: []string{"gpu"},
	})
	_, _, err := store.ClaimNextTask(context.Background(), ClaimOptions{
		WorkerID: "cpu", Queue: "training", Capabilities: []string{"cpu"},
	})
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected no compatible work, got %v", err)
	}
}

func TestSpawnTaskHonorsBound(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 0, Task{ID: "root", Status: TaskPending})
	ctx := context.Background()
	if err := store.SpawnTask(ctx, "root", Task{RunID: "run-1", ID: "child", Status: TaskPending}, 2); err != nil {
		t.Fatal(err)
	}
	if err := store.SpawnTask(ctx, "root", Task{RunID: "run-1", ID: "overflow", Status: TaskPending}, 2); !errors.Is(err, ErrConcurrencyFull) {
		t.Fatalf("expected task bound, got %v", err)
	}
}

func TestDueDeploymentAndTrigger(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := store.UpsertDeployment(ctx, Deployment{
		ID: "daily", DefinitionID: "report", Enabled: true, NextRunAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	due, err := store.DueDeployments(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != "daily" {
		t.Fatalf("due=%#v", due)
	}
	if err := store.MarkDeploymentTriggered(ctx, "daily", now, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	due, err = store.DueDeployments(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("deployment should no longer be due: %#v", due)
	}
}

func TestNextScheduleTime(t *testing.T) {
	after := time.Date(2026, 7, 21, 16, 0, 0, 0, time.UTC)
	next, err := NextScheduleTime("@every 15m", after)
	if err != nil || !next.Equal(after.Add(15*time.Minute)) {
		t.Fatalf("next=%v err=%v", next, err)
	}
	next, err = NextScheduleTime("@daily 15:30", after)
	if err != nil || !next.Equal(time.Date(2026, 7, 22, 15, 30, 0, 0, time.UTC)) {
		t.Fatalf("daily next=%v err=%v", next, err)
	}
}

type deploymentLauncherFunc func(context.Context, Deployment, map[string]any) (string, error)

func (fn deploymentLauncherFunc) LaunchDeployment(ctx context.Context, deployment Deployment, parameters map[string]any) (string, error) {
	return fn(ctx, deployment, parameters)
}

func TestProcessDueDeploymentLaunchesExactlyOncePerSchedule(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := store.UpsertDeployment(ctx, Deployment{
		ID: "scheduled", DefinitionID: "definition", Schedule: "@every 1h",
		Enabled: true, NextRunAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	launches := 0
	launcher := deploymentLauncherFunc(func(context.Context, Deployment, map[string]any) (string, error) {
		launches++
		return "launched-run", nil
	})
	count, err := store.ProcessDueDeployments(ctx, now, launcher)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || launches != 1 {
		t.Fatalf("count=%d launches=%d", count, launches)
	}
	count, err = store.ProcessDueDeployments(ctx, now, launcher)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || launches != 1 {
		t.Fatalf("deployment relaunched count=%d launches=%d", count, launches)
	}
}

type countingAutomationHandler struct {
	mu    sync.Mutex
	count int
}

func (h *countingAutomationHandler) HandleAutomation(context.Context, Automation, Event) error {
	h.mu.Lock()
	h.count++
	h.mu.Unlock()
	return nil
}

func TestAutomationDeliveryIsIdempotent(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if err := store.UpsertAutomation(ctx, Automation{
		ID: "notify", Name: "notify", EventType: "task.*", ActionType: "notify", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendEvent(ctx, Event{RunID: "run", Type: "task.failed"}); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListEvents(ctx, "run", 0, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	handler := &countingAutomationHandler{}
	if err := store.DispatchAutomations(ctx, events[0], handler); err != nil {
		t.Fatal(err)
	}
	if err := store.DispatchAutomations(ctx, events[0], handler); err != nil {
		t.Fatal(err)
	}
	if handler.count != 1 {
		t.Fatalf("automation delivered %d times, want 1", handler.count)
	}
}

func TestIdempotentSideEffectLeaseRequiresManualReconciliation(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{
		ID: "write", Status: TaskPending, MaxRetries: 3,
		Metadata: map[string]any{"idempotency_required": true},
	})
	ctx := context.Background()
	now := time.Unix(5_000, 0).UTC()
	if _, err := store.ClaimTask(ctx, "run-1", "write", ClaimOptions{Now: now, Lease: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpiredLeases(ctx, now.Add(2*time.Second), RetryPolicy{MaxRetries: 3}); err != nil {
		t.Fatal(err)
	}
	task, err := store.GetTask(ctx, "run-1", "write")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskFailed {
		t.Fatalf("idempotent side effect should fail closed, got %s", task.Status)
	}
}
