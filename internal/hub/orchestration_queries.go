package hub

import (
	"context"
	"errors"
	"time"

	"github.com/camronwood/neural-junkie/internal/orchestration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

type OrchestrationTaskMetrics struct {
	TaskID           string `json:"task_id"`
	QueueDelayMillis int64  `json:"queue_delay_ms"`
	ExecutionMillis  int64  `json:"execution_ms"`
	Retries          int    `json:"retries"`
	CacheHit         bool   `json:"cache_hit"`
	FailureReason    string `json:"failure_reason,omitempty"`
	InferenceUsage   []any  `json:"inference_usage,omitempty"`
}

type OrchestrationSnapshot struct {
	Run      *orchestration.Run           `json:"run"`
	Tasks    []orchestration.Task         `json:"tasks"`
	Attempts []orchestration.Attempt      `json:"attempts"`
	Metrics  []OrchestrationTaskMetrics   `json:"metrics"`
	Events   []orchestration.Event        `json:"events"`
	Inputs   []orchestration.InputRequest `json:"inputs,omitempty"`
	Workers  []orchestration.Worker       `json:"workers"`
	Enforced bool                         `json:"enforced"`
}

func (h *Hub) GetOrchestrationSnapshot(ctx context.Context, runID string, afterEventID int64) (*OrchestrationSnapshot, error) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		if runtime != nil && runtime.err != nil {
			return nil, runtime.err
		}
		return nil, orchestration.ErrNotFound
	}
	run, err := runtime.store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	tasks, err := runtime.store.ListTasks(ctx, runID)
	if err != nil {
		return nil, err
	}
	events, err := runtime.store.ListEvents(ctx, runID, afterEventID, 500)
	if err != nil {
		return nil, err
	}
	allInputs, err := runtime.store.ListPendingInputs(ctx)
	if err != nil {
		return nil, err
	}
	inputs := make([]orchestration.InputRequest, 0)
	for _, input := range allInputs {
		if input.RunID == runID {
			inputs = append(inputs, input)
		}
	}
	workers, err := runtime.store.ListWorkers(ctx)
	if err != nil {
		return nil, err
	}
	attempts, err := runtime.store.ListAttempts(ctx, runID)
	if err != nil {
		return nil, err
	}
	metrics := buildOrchestrationMetrics(tasks, attempts, events)
	if h.collabManager != nil {
		if collab, err := h.collabManager.GetCollaborationSnapshot(runID); err == nil {
			if messages, err := h.GetMessages(collab.Channel, 1000); err == nil {
				usageByTask := map[string][]any{}
				for _, message := range messages {
					if message == nil || message.GetCollaborationID() != runID || message.Metadata == nil {
						continue
					}
					if usage, ok := message.Metadata[protocol.MetadataInferenceUsage]; ok {
						usageByTask[message.GetTaskID()] = append(usageByTask[message.GetTaskID()], usage)
					}
				}
				for index := range metrics {
					metrics[index].InferenceUsage = usageByTask[metrics[index].TaskID]
				}
			}
		}
	}
	return &OrchestrationSnapshot{
		Run: run, Tasks: tasks, Attempts: attempts, Metrics: metrics,
		Events: events, Inputs: inputs, Workers: workers,
		Enforced: h.durableOrchestrationEnforced(),
	}, nil
}

func buildOrchestrationMetrics(
	tasks []orchestration.Task,
	attempts []orchestration.Attempt,
	events []orchestration.Event,
) []OrchestrationTaskMetrics {
	byTask := make(map[string][]orchestration.Attempt)
	for _, attempt := range attempts {
		byTask[attempt.TaskID] = append(byTask[attempt.TaskID], attempt)
	}
	cacheHits := make(map[string]bool)
	for _, event := range events {
		if event.Type == "task.cache_hit" {
			cacheHits[event.TaskID] = true
		}
	}
	now := time.Now()
	out := make([]OrchestrationTaskMetrics, 0, len(tasks))
	for _, task := range tasks {
		taskAttempts := byTask[task.ID]
		metric := OrchestrationTaskMetrics{
			TaskID: task.ID, Retries: max(0, len(taskAttempts)-1), CacheHit: cacheHits[task.ID],
		}
		for index, attempt := range taskAttempts {
			if index == 0 && !attempt.StartedAt.IsZero() && !task.CreatedAt.IsZero() {
				metric.QueueDelayMillis = max(0, attempt.StartedAt.Sub(task.CreatedAt).Milliseconds())
			}
			end := attempt.CompletedAt
			if end.IsZero() {
				end = now
			}
			if !attempt.StartedAt.IsZero() && end.After(attempt.StartedAt) {
				metric.ExecutionMillis += end.Sub(attempt.StartedAt).Milliseconds()
			}
			if attempt.Error != "" {
				metric.FailureReason = attempt.Error
			}
		}
		out = append(out, metric)
	}
	return out
}

func (h *Hub) UpsertOrchestrationDeployment(ctx context.Context, deployment orchestration.Deployment) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		if runtime != nil && runtime.err != nil {
			return runtime.err
		}
		return orchestration.ErrNotFound
	}
	return runtime.store.UpsertDeployment(ctx, deployment)
}

func (h *Hub) UpsertOrchestrationAutomation(ctx context.Context, automation orchestration.Automation) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		if runtime != nil && runtime.err != nil {
			return runtime.err
		}
		return orchestration.ErrNotFound
	}
	return runtime.store.UpsertAutomation(ctx, automation)
}

func (h *Hub) RegisterOrchestrationWorker(ctx context.Context, worker orchestration.Worker) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	return runtime.store.RegisterWorker(ctx, worker)
}

func (h *Hub) HeartbeatOrchestrationWorker(ctx context.Context, workerID string) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	return runtime.store.HeartbeatWorker(ctx, workerID, orchestration.WorkerReady, time.Now())
}

func (h *Hub) ClaimOrchestrationWork(ctx context.Context, options orchestration.ClaimOptions) (*orchestration.Task, *orchestration.Attempt, error) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return nil, nil, orchestration.ErrNotFound
	}
	task, attempt, err := runtime.store.ClaimNextTask(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	if err := runtime.store.MarkAttemptRunning(ctx, attempt.ID, attempt.LeaseToken); err != nil {
		return nil, nil, err
	}
	return task, attempt, nil
}

func (h *Hub) HeartbeatOrchestrationWork(
	ctx context.Context,
	attemptID, leaseToken string,
	extend time.Duration,
	progress map[string]any,
) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	if err := runtime.store.Heartbeat(ctx, attemptID, leaseToken, extend, time.Now()); err != nil {
		return err
	}
	if len(progress) > 0 {
		attempt, err := runtime.store.GetAttempt(ctx, attemptID)
		if err != nil {
			return err
		}
		return runtime.store.AppendEvent(ctx, orchestration.Event{
			RunID: attempt.RunID, TaskID: attempt.TaskID, AttemptID: attemptID,
			Type: "attempt.progress", Payload: progress,
		})
	}
	return nil
}

func (h *Hub) SpawnOrchestrationTask(
	ctx context.Context,
	parentTaskID string,
	task orchestration.Task,
	maxTasks int,
) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	if maxTasks <= 0 || maxTasks > 100 {
		maxTasks = 100
	}
	return runtime.store.SpawnTask(ctx, parentTaskID, task, maxTasks)
}

func (h *Hub) CompleteOrchestrationWork(
	ctx context.Context,
	attemptID, leaseToken string,
	value []byte,
	contentType string,
	metadata map[string]any,
) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	var resultID string
	if value != nil {
		attempt, err := runtime.store.GetAttempt(ctx, attemptID)
		if err != nil {
			return err
		}
		task, err := runtime.store.GetTask(ctx, attempt.RunID, attempt.TaskID)
		if err != nil {
			return err
		}
		expiresAt := time.Time{}
		if task.CachePolicy.Expiration > 0 {
			expiresAt = time.Now().Add(task.CachePolicy.Expiration)
		}
		result, err := runtime.store.PutResult(ctx, orchestration.Result{
			RunID: attempt.RunID, TaskID: attempt.TaskID, AttemptID: attemptID,
			ExecutionKey: task.ExecutionKey, Value: value, ContentType: contentType,
			Metadata: metadata, ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}
		resultID = result.ID
	}
	return runtime.store.CompleteAttempt(ctx, attemptID, leaseToken, resultID)
}

func (h *Hub) FailOrchestrationWork(ctx context.Context, attemptID, leaseToken, message string) error {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return orchestration.ErrNotFound
	}
	attempt, err := runtime.store.GetAttempt(ctx, attemptID)
	if err != nil {
		return err
	}
	task, err := runtime.store.GetTask(ctx, attempt.RunID, attempt.TaskID)
	if err != nil {
		return err
	}
	_, _, err = runtime.store.FailAttempt(ctx, attemptID, leaseToken, errors.New(message), orchestration.RetryPolicy{
		MaxRetries: task.MaxRetries, BaseDelay: time.Second, MaxDelay: time.Minute, JitterFactor: 0.2,
		RetryIf: func(error) bool {
			return orchestration.RetryCategoryAllowed(task.Metadata, "error")
		},
	})
	return err
}
