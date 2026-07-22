package orchestration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPropertyRetryBudgetProducesAtMostMaxRetriesPlusOneAttempts(t *testing.T) {
	for maxRetries := 0; maxRetries <= 6; maxRetries++ {
		t.Run(fmt.Sprintf("max_%d", maxRetries), func(t *testing.T) {
			store := testStore(t)
			seedRunAndTasks(t, store, 1, Task{
				ID: "task", Status: TaskPending, MaxRetries: maxRetries,
			})
			ctx := context.Background()
			now := time.Now()
			policy := RetryPolicy{
				MaxRetries: maxRetries, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond,
			}
			for number := 1; ; number++ {
				attempt, err := store.ClaimTask(ctx, "run-1", "task", ClaimOptions{Now: now})
				if err != nil {
					t.Fatalf("claim attempt %d: %v", number, err)
				}
				retry, next, err := store.FailAttempt(ctx, attempt.ID, attempt.LeaseToken, errors.New("failed"), policy)
				if err != nil {
					t.Fatal(err)
				}
				if !retry {
					if number != maxRetries+1 {
						t.Fatalf("exhausted after %d attempts, want %d", number, maxRetries+1)
					}
					break
				}
				now = next
			}
			attempts, err := store.ListAttempts(ctx, "run-1")
			if err != nil {
				t.Fatal(err)
			}
			if len(attempts) != maxRetries+1 {
				t.Fatalf("attempt history=%d, want %d", len(attempts), maxRetries+1)
			}
		})
	}
}

func TestPropertyTerminalTaskCannotBeClaimedAgain(t *testing.T) {
	for _, terminal := range []TaskStatus{TaskCompleted, TaskFailed, TaskCancelled, TaskSkipped} {
		t.Run(string(terminal), func(t *testing.T) {
			store := testStore(t)
			seedRunAndTasks(t, store, 1, Task{ID: "task", Status: terminal})
			if _, err := store.ClaimTask(context.Background(), "run-1", "task", ClaimOptions{}); !errors.Is(err, ErrNotReady) {
				t.Fatalf("terminal %s was claimable: %v", terminal, err)
			}
		})
	}
}
