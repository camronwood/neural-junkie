package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "orchestration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedRunAndTasks(t *testing.T, store *Store, maxConcurrency int, tasks ...Task) {
	t.Helper()
	ctx := context.Background()
	if err := store.UpsertRun(ctx, Run{
		ID: "run-1", DefinitionID: "test", DefinitionVersion: 1,
		Status: RunRunning, MaxConcurrency: maxConcurrency,
	}); err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		task.RunID = "run-1"
		if err := store.UpsertTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClaimTaskEnforcesRunWideConcurrency(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1,
		Task{ID: "a", Status: TaskPending},
		Task{ID: "b", Status: TaskPending},
	)
	ctx := context.Background()
	now := time.Unix(1_000, 0).UTC()
	a, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{WorkerID: "one", Lease: time.Minute, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimTask(ctx, "run-1", "b", ClaimOptions{WorkerID: "two", Lease: time.Minute, Now: now}); !errors.Is(err, ErrConcurrencyFull) {
		t.Fatalf("expected concurrency limit, got %v", err)
	}
	if err := store.CompleteAttempt(ctx, a.ID, a.LeaseToken, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimTask(ctx, "run-1", "b", ClaimOptions{WorkerID: "two", Lease: time.Minute, Now: now}); err != nil {
		t.Fatalf("claim after slot released: %v", err)
	}
}

func TestRetryPolicyPersistsAttemptHistory(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{ID: "a", Status: TaskPending, MaxRetries: 1})
	ctx := context.Background()
	now := time.Unix(2_000, 0).UTC()
	attempt, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Lease: time.Minute, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	policy := RetryPolicy{MaxRetries: 1, BaseDelay: time.Second, MaxDelay: time.Second}
	retry, next, err := store.FailAttempt(ctx, attempt.ID, attempt.LeaseToken, errors.New("temporary"), policy)
	if err != nil {
		t.Fatal(err)
	}
	if !retry || next.IsZero() {
		t.Fatalf("expected scheduled retry, retry=%v next=%v", retry, next)
	}
	if _, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Now: next.Add(-time.Millisecond)}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected backoff to block claim, got %v", err)
	}
	second, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Now: next})
	if err != nil {
		t.Fatal(err)
	}
	retry, _, err = store.FailAttempt(ctx, second.ID, second.LeaseToken, errors.New("permanent"), policy)
	if err != nil {
		t.Fatal(err)
	}
	if retry {
		t.Fatal("retry budget should be exhausted")
	}
	task, err := store.GetTask(ctx, "run-1", "a")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskFailed || task.AttemptCount != 2 {
		t.Fatalf("unexpected terminal task: %#v", task)
	}
}

func TestRunWideRetryBudgetCapsRetriesAcrossTasks(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if err := store.UpsertRun(ctx, Run{
		ID: "run", Status: RunRunning, Metadata: map[string]any{"retry_budget": 1},
	}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if err := store.UpsertTask(ctx, Task{RunID: "run", ID: id, Status: TaskPending, MaxRetries: 3}); err != nil {
			t.Fatal(err)
		}
	}
	policy := RetryPolicy{MaxRetries: 3, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}
	a, err := store.ClaimTask(ctx, "run", "a", ClaimOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, next, err := store.FailAttempt(ctx, a.ID, a.LeaseToken, errors.New("failed"), policy)
	if err != nil {
		t.Fatal(err)
	}
	aRetry, err := store.ClaimTask(ctx, "run", "a", ClaimOptions{Now: next})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.FailAttempt(ctx, aRetry.ID, aRetry.LeaseToken, errors.New("failed"), policy); err != nil {
		t.Fatal(err)
	}
	b, err := store.ClaimTask(ctx, "run", "b", ClaimOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, next, err = store.FailAttempt(ctx, b.ID, b.LeaseToken, errors.New("failed"), policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ClaimTask(ctx, "run", "b", ClaimOptions{Now: next}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("expected run retry budget exhaustion, got %v", err)
	}
}

func TestCrashRecoveryReconcilesExpiredLeaseAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orchestration.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedRunAndTasks(t, store, 1, Task{ID: "a", Status: TaskPending, MaxRetries: 1})
	ctx := context.Background()
	now := time.Unix(3_000, 0).UTC()
	if _, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Lease: time.Second, Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	count, err := store.ReconcileExpiredLeases(ctx, now.Add(2*time.Second), RetryPolicy{
		MaxRetries: 1, BaseDelay: time.Second, MaxDelay: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("reconciled %d attempts, want 1", count)
	}
	task, err := store.GetTask(ctx, "run-1", "a")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskRetrying {
		t.Fatalf("status=%s, want retrying", task.Status)
	}
}

func TestAttemptTimeoutCannotBeExtendedByHeartbeat(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{
		ID: "a", Status: TaskPending, Timeout: time.Second, MaxRetries: 0,
	})
	ctx := context.Background()
	now := time.Unix(3_500, 0).UTC()
	attempt, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Lease: time.Hour, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Heartbeat(ctx, attempt.ID, attempt.LeaseToken, time.Hour, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	count, err := store.ReconcileExpiredLeases(ctx, now.Add(2*time.Second), RetryPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("timed out attempts=%d, want 1", count)
	}
	reconciled, err := store.GetAttempt(ctx, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != AttemptTimedOut {
		t.Fatalf("attempt status=%s, want timed_out", reconciled.Status)
	}
}

func TestRetryPredicateCanFailClosedOnTimeout(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{
		ID: "a", Status: TaskPending, Timeout: time.Second, MaxRetries: 3,
		Metadata: map[string]any{"retry_on": []string{"error"}},
	})
	ctx := context.Background()
	now := time.Unix(3_600, 0).UTC()
	if _, err := store.ClaimTask(ctx, "run-1", "a", ClaimOptions{Lease: time.Hour, Now: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpiredLeases(ctx, now.Add(2*time.Second), RetryPolicy{MaxRetries: 3}); err != nil {
		t.Fatal(err)
	}
	task, err := store.GetTask(ctx, "run-1", "a")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskFailed {
		t.Fatalf("timeout should not retry under error-only predicate: %s", task.Status)
	}
}

func TestDurableInputSurvivesRestartAndResolvesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orchestration.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	created, err := store.CreateInput(ctx, InputRequest{
		ID: "approval-1", Kind: "approval", DecisionKey: "deploy",
		Schema: json.RawMessage(`{"type":"object"}`), ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	pending, err := store.ListPendingInputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != created.ID {
		t.Fatalf("pending=%#v", pending)
	}
	if _, err := store.ResolveInput(ctx, created.ID, "camron", json.RawMessage(`{"approved":true}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveInput(ctx, created.ID, "camron", json.RawMessage(`{"approved":true}`)); !errors.Is(err, ErrAlreadyResolved) {
		t.Fatalf("expected exactly-once resolution, got %v", err)
	}
}

func TestDurableInputValidatesResponseSchema(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	input, err := store.CreateInput(ctx, InputRequest{
		Kind: "approval",
		Schema: json.RawMessage(`{
			"type":"object",
			"required":["status"],
			"properties":{"status":{"type":"string","enum":["approved","rejected"]}}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveInput(ctx, input.ID, "camron", json.RawMessage(`{"status":"maybe"}`)); err == nil {
		t.Fatal("invalid enum should be rejected")
	}
	if _, err := store.ResolveInput(ctx, input.ID, "camron", json.RawMessage(`{"status":"approved"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestDurableInputSuspendsAndResumesRun(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if err := store.UpsertRun(ctx, Run{ID: "run", Status: RunRunning}); err != nil {
		t.Fatal(err)
	}
	input, err := store.CreateInput(ctx, InputRequest{RunID: "run", Kind: "approval"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.GetRun(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunSuspended {
		t.Fatalf("run status=%s, want suspended", run.Status)
	}
	if _, err := store.ResolveInput(ctx, input.ID, "camron", json.RawMessage(`{"approved":true}`)); err != nil {
		t.Fatal(err)
	}
	run, err = store.GetRun(ctx, "run")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunRunning {
		t.Fatalf("run status=%s, want running", run.Status)
	}
}

func TestExpireDueInputsIsDurable(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := store.CreateInput(ctx, InputRequest{
		ID: "expired", Kind: "approval", ExpiresAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	count, err := store.ExpireDueInputs(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expired=%d, want 1", count)
	}
	pending, err := store.ListPendingInputs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expired input remained pending: %#v", pending)
	}
}

func TestResultCacheHonorsExpiration(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	now := time.Unix(4_000, 0).UTC()
	_, err := store.PutResult(ctx, Result{
		ExecutionKey: "key", Value: []byte("result"),
		ExpiresAt: now.Add(time.Minute), CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := store.GetCachedResult(ctx, "key", now); err != nil || string(result.Value) != "result" {
		t.Fatalf("cache hit=%#v err=%v", result, err)
	}
	if _, err := store.GetCachedResult(ctx, "key", now.Add(2*time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected expired miss, got %v", err)
	}
}

func TestExecutionKeyIsStableForMapOrder(t *testing.T) {
	left, err := ExecutionKey(ExecutionKeyInput{
		DefinitionID: "runbook", TaskID: "task", Inputs: map[string]any{"a": 1, "b": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := ExecutionKey(ExecutionKeyInput{
		DefinitionID: "runbook", TaskID: "task", Inputs: map[string]any{"b": 2, "a": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("execution keys differ: %s != %s", left, right)
	}
}

func TestTransitionRunUsesCompareAndSwap(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if err := store.UpsertRun(ctx, Run{ID: "run", Status: RunPending}); err != nil {
		t.Fatal(err)
	}
	if err := store.TransitionRun(ctx, "run", []RunStatus{RunRunning}, RunCompleted); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := store.TransitionRun(ctx, "run", []RunStatus{RunPending}, RunRunning); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentClaimsNeverExceedRunLimit(t *testing.T) {
	store := testStore(t)
	tasks := make([]Task, 12)
	for i := range tasks {
		tasks[i] = Task{ID: "task-" + string(rune('a'+i)), Status: TaskPending}
	}
	seedRunAndTasks(t, store, 3, tasks...)
	ctx := context.Background()
	now := time.Now()
	var successes atomic.Int64
	var wg sync.WaitGroup
	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ClaimTask(ctx, "run-1", task.ID, ClaimOptions{
				WorkerID: task.ID, Lease: time.Minute, Now: now,
			}); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrConcurrencyFull) {
				t.Errorf("claim %s: %v", task.ID, err)
			}
		}()
	}
	wg.Wait()
	if got := successes.Load(); got != 3 {
		t.Fatalf("successful claims=%d, want exactly 3", got)
	}
}

func TestConcurrentInputResolutionIsExactlyOnce(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	input, err := store.CreateInput(ctx, InputRequest{Kind: "approval"})
	if err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ResolveInput(ctx, input.ID, "user", json.RawMessage(`{"approved":true}`)); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrAlreadyResolved) {
				t.Errorf("resolve: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("successful resolutions=%d, want 1", got)
	}
}

func TestAttemptRejectsStaleLeaseToken(t *testing.T) {
	store := testStore(t)
	seedRunAndTasks(t, store, 1, Task{ID: "a", Status: TaskPending})
	attempt, err := store.ClaimTask(context.Background(), "run-1", "a", ClaimOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteAttempt(context.Background(), attempt.ID, "wrong-token", ""); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expected stale lease rejection, got %v", err)
	}
	task, err := store.GetTask(context.Background(), "run-1", "a")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskRunning {
		t.Fatalf("stale completion mutated task to %s", task.Status)
	}
}
