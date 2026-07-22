package hub

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/orchestration"
)

func TestAuthoritativeSyncDoesNotRegressRetryState(t *testing.T) {
	store, err := orchestration.Open(filepath.Join(t.TempDir(), "orchestration.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now()
	snapshot := &collaboration.Collaboration{
		ID: "run", Phase: collaboration.PhaseExecuting, CreatedAt: now, UpdatedAt: now,
		Tasks: []collaboration.CollaborationTask{{
			ID: "task", Title: "Task", Status: collaboration.TaskPending,
			CreatedAt: now, UpdatedAt: now,
			Options: &collaboration.TaskExecutionOptions{MaxRetries: 1},
		}},
	}
	ctx := context.Background()
	if err := syncOrchestrationSnapshot(ctx, store, snapshot, false); err != nil {
		t.Fatal(err)
	}
	attempt, err := store.ClaimTask(ctx, "run", "task", orchestration.ClaimOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.FailAttempt(ctx, attempt.ID, attempt.LeaseToken, errors.New("retry"), orchestration.RetryPolicy{
		MaxRetries: 1, BaseDelay: time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	if err := syncOrchestrationSnapshot(ctx, store, snapshot, true); err != nil {
		t.Fatal(err)
	}
	task, err := store.GetTask(ctx, "run", "task")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != orchestration.TaskRetrying {
		t.Fatalf("authoritative retry state regressed to %s", task.Status)
	}
}

func TestAuthoritativeSyncImportsTerminalLegacyCompletion(t *testing.T) {
	store, err := orchestration.Open(filepath.Join(t.TempDir(), "orchestration.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now()
	snapshot := &collaboration.Collaboration{
		ID: "run", Phase: collaboration.PhaseExecuting, CreatedAt: now, UpdatedAt: now,
		Tasks: []collaboration.CollaborationTask{{
			ID: "task", Title: "Task", Status: collaboration.TaskPending,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
	ctx := context.Background()
	if err := syncOrchestrationSnapshot(ctx, store, snapshot, false); err != nil {
		t.Fatal(err)
	}
	snapshot.Tasks[0].Status = collaboration.TaskCompleted
	snapshot.Tasks[0].Output = "done"
	if err := syncOrchestrationSnapshot(ctx, store, snapshot, true); err != nil {
		t.Fatal(err)
	}
	task, err := store.GetTask(ctx, "run", "task")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != orchestration.TaskCompleted {
		t.Fatalf("terminal completion was not imported: %s", task.Status)
	}
}
