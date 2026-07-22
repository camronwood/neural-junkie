package orchestration

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func reopenStore(t *testing.T, path string, store *Store) *Store {
	t.Helper()
	if store != nil {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return reopened
}

func TestCrashPointsPreserveClaimAndResultInvariants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orchestration.db")
	ctx := context.Background()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedRunAndTasks(t, store, 1, Task{
		ID: "task", Status: TaskPending, MaxRetries: 1, ExecutionKey: "execution-key",
		CachePolicy: CachePolicy{Enabled: true, Expiration: time.Hour},
	})

	// Crash before claim: durable task remains pending and no phantom attempt exists.
	store = reopenStore(t, path, store)
	attempts, err := store.ListAttempts(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 0 {
		t.Fatalf("phantom attempts before claim: %#v", attempts)
	}

	// Crash after claim/dispatch: the immutable attempt and lease survive.
	attempt, err := store.ClaimTask(ctx, "run-1", "task", ClaimOptions{Lease: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkAttemptRunning(ctx, attempt.ID, attempt.LeaseToken); err != nil {
		t.Fatal(err)
	}
	store = reopenStore(t, path, store)
	recovered, err := store.GetAttempt(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != AttemptRunning || recovered.LeaseToken != attempt.LeaseToken {
		t.Fatalf("recovered attempt=%#v", recovered)
	}

	// Crash after result persistence but before completion commit: a safe retry
	// can reuse the execution-key result instead of repeating work.
	result, err := store.PutResult(ctx, Result{
		RunID: "run-1", TaskID: "task", AttemptID: attempt.ID,
		ExecutionKey: "execution-key", Value: []byte("durable result"),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	store = reopenStore(t, path, store)
	cached, err := store.GetCachedResult(ctx, "execution-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if cached.ID != result.ID || string(cached.Value) != "durable result" {
		t.Fatalf("cached result=%#v", cached)
	}

	// Completion commit is exactly once.
	if err := store.CompleteAttempt(ctx, attempt.ID, attempt.LeaseToken, result.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteAttempt(ctx, attempt.ID, attempt.LeaseToken, result.ID); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("second completion should fail, got %v", err)
	}
	store = reopenStore(t, path, store)
	defer store.Close()
	task, err := store.GetTask(ctx, "run-1", "task")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskCompleted {
		t.Fatalf("completion was not durable: %s", task.Status)
	}
}

func TestCrashAfterInputResponseDoesNotRequestSecondDecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orchestration.db")
	ctx := context.Background()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	input, err := store.CreateInput(ctx, InputRequest{
		ID: "input", Kind: "approval", DecisionKey: "deploy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveInput(ctx, input.ID, "user", []byte(`{"approved":true}`)); err != nil {
		t.Fatal(err)
	}
	store = reopenStore(t, path, store)
	defer store.Close()
	pending, err := store.ListPendingInputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("resolved decision reappeared after restart: %#v", pending)
	}
	if _, err := store.ResolveInput(ctx, input.ID, "user", []byte(`{"approved":true}`)); !errors.Is(err, ErrAlreadyResolved) {
		t.Fatalf("response applied twice: %v", err)
	}
}
