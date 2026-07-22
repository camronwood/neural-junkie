package hub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/orchestration"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const (
	durableOrchestrationEnv = "NEURAL_JUNKIE_DURABLE_ORCHESTRATION"
	orchestrationDBEnv      = "NEURAL_JUNKIE_ORCHESTRATION_DB"
)

type hubOrchestrationRuntime struct {
	store       *orchestration.Store
	err         error
	eventCursor map[string]int64
}

var hubOrchestrationStores = struct {
	sync.Mutex
	byHub map[*Hub]*hubOrchestrationRuntime
}{byHub: make(map[*Hub]*hubOrchestrationRuntime)}

func (h *Hub) orchestrationRuntime() *hubOrchestrationRuntime {
	if h == nil {
		return nil
	}
	hubOrchestrationStores.Lock()
	defer hubOrchestrationStores.Unlock()
	if runtime, ok := hubOrchestrationStores.byHub[h]; ok {
		return runtime
	}
	path := strings.TrimSpace(os.Getenv(orchestrationDBEnv))
	if path == "" && strings.HasSuffix(os.Args[0], ".test") {
		path = ":memory:"
	}
	store, err := orchestration.Open(path)
	runtime := &hubOrchestrationRuntime{store: store, err: err, eventCursor: make(map[string]int64)}
	hubOrchestrationStores.byHub[h] = runtime
	if err != nil {
		log.Printf("[Orchestration] durable store unavailable: %v", err)
	} else {
		log.Printf("[Orchestration] durable run store ready")
		ctx := context.Background()
		_ = store.RegisterWorker(ctx, orchestration.Worker{
			ID: "local-hub", Queue: "default", Status: orchestration.WorkerReady,
			Capabilities: []string{"agent", "action", "sidecar"},
		})
		_ = store.UpsertAutomation(ctx, orchestration.Automation{
			ID: "notify-expired-leases", Name: "Notify expired task leases",
			EventType: "attempt.lease_expired", ActionType: "notify", Enabled: true,
		})
		_ = store.UpsertAutomation(ctx, orchestration.Automation{
			ID: "notify-attempt-timeout", Name: "Notify task attempt timeout",
			EventType: "attempt.timed_out", ActionType: "notify", Enabled: true,
		})
		_ = store.UpsertAutomation(ctx, orchestration.Automation{
			ID: "notify-expired-inputs", Name: "Notify expired human input",
			EventType: "input.expired", ActionType: "notify", Enabled: true,
		})
		_ = store.UpsertAutomation(ctx, orchestration.Automation{
			ID: "notify-sla-breach", Name: "Notify orchestration SLA breach",
			EventType: "run.sla_breached", ActionType: "notify", Enabled: true,
		})
		_ = store.UpsertAutomation(ctx, orchestration.Automation{
			ID: "notify-retry-exhaustion", Name: "Notify exhausted task retries",
			EventType: "attempt.failed", ActionType: "notify", Enabled: true,
		})
	}
	return runtime
}

func (h *Hub) durableOrchestrationEnforced() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(durableOrchestrationEnv)))
	return value == "1" || value == "true" || value == "yes"
}

// CloseOrchestrationStore flushes and closes the per-hub durable store.
func (h *Hub) CloseOrchestrationStore() error {
	hubOrchestrationStores.Lock()
	runtime := hubOrchestrationStores.byHub[h]
	delete(hubOrchestrationStores.byHub, h)
	hubOrchestrationStores.Unlock()
	if runtime == nil || runtime.store == nil {
		return nil
	}
	return runtime.store.Close()
}

func (h *Hub) syncOrchestrationState(ctx context.Context) {
	if h == nil || h.collabManager == nil {
		return
	}
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	for _, snapshot := range h.collabManager.Snapshot() {
		if snapshot == nil {
			continue
		}
		if err := syncOrchestrationSnapshot(ctx, runtime.store, snapshot, h.durableOrchestrationEnforced()); err != nil {
			log.Printf("[Orchestration] sync collaboration %s: %v", shortOrchestrationID(snapshot.ID), err)
		}
	}
}

func syncOrchestrationSnapshot(
	ctx context.Context,
	store *orchestration.Store,
	snapshot *collaboration.Collaboration,
	authoritative bool,
) error {
	if store == nil || snapshot == nil || snapshot.ID == "" {
		return nil
	}
	runStatus := orchestrationRunStatus(snapshot.Phase, snapshot.DispatchPaused)
	if authoritative && runStatus == orchestration.RunRunning {
		if persisted, err := store.GetRun(ctx, snapshot.ID); err == nil && persisted.Status == orchestration.RunSuspended {
			runStatus = orchestration.RunSuspended
		}
	}
	runMetadata := map[string]any{
		"title":        snapshot.Title,
		"description":  snapshot.Description,
		"channel":      snapshot.Channel,
		"source":       snapshot.Source,
		"sla_seconds":  snapshot.EffectiveExecutionPolicy().SLASeconds,
		"retry_budget": snapshot.EffectiveExecutionPolicy().RetryBudget,
	}
	if err := store.UpsertRun(ctx, orchestration.Run{
		ID: snapshot.ID, DefinitionID: snapshot.DefinitionID,
		DefinitionVersion: snapshot.DefinitionVersion, Status: runStatus,
		MaxConcurrency: snapshot.EffectiveExecutionPolicy().MaxConcurrentTasks,
		Metadata:       runMetadata, CreatedAt: snapshot.CreatedAt, UpdatedAt: snapshot.UpdatedAt,
	}); err != nil {
		return err
	}
	for _, task := range snapshot.Tasks {
		status := orchestrationTaskStatus(task)
		maxRetries := 0
		timeout := time.Duration(collaboration.ExecutionTimeoutSeconds(task, 0)) * time.Second
		provider, model, queue := "", "", "default"
		var capabilityTags []string
		cachePolicy := orchestration.CachePolicy{}
		if task.Options != nil {
			maxRetries = task.Options.MaxRetries
			if task.Options.TimeoutSeconds > 0 {
				timeout = time.Duration(task.Options.TimeoutSeconds) * time.Second
			}
			provider = task.Options.ProviderID
			model = task.Options.ExpectedModel
			if strings.TrimSpace(task.Options.Queue) != "" {
				queue = strings.TrimSpace(task.Options.Queue)
			}
			capabilityTags = append([]string(nil), task.Options.CapabilityTags...)
			cachePolicy.Enabled = task.Options.CachePolicy == "result"
			cachePolicy.Refresh = task.Options.RefreshCache
			if task.Options.CacheExpirationSecs > 0 {
				cachePolicy.Expiration = time.Duration(task.Options.CacheExpirationSecs) * time.Second
			}
		}
		definitionID := snapshot.DefinitionID
		if definitionID == "" {
			definitionID = "collaboration"
		}
		key, err := orchestration.ExecutionKey(orchestration.ExecutionKeyInput{
			DefinitionID: definitionID, DefinitionVersion: snapshot.DefinitionVersion,
			TaskID: task.ID, Inputs: stringMapToAny(snapshot.RunInputs),
			ContextHash: orchestrationContextHash(snapshot, task), Provider: provider,
			Model: model, PolicyVersion: "collaboration-v1",
		})
		if err != nil {
			return err
		}
		metadata := map[string]any{
			"assigned_to": task.AssignedTo, "kind": task.EffectiveKind(),
			"requires_approval": task.AwaitingApproval,
		}
		if task.Options != nil {
			metadata["idempotency_required"] = task.Options.IdempotencyRequired
			metadata["retry_on"] = append([]string(nil), task.Options.RetryOn...)
		}
		if err := store.UpsertTask(ctx, orchestration.Task{
			ID: task.ID, RunID: snapshot.ID, Title: task.Title, Status: status,
			Queue: queue, CapabilityTags: capabilityTags,
			MaxRetries: maxRetries, Timeout: timeout, ExecutionKey: key, CachePolicy: cachePolicy,
			Metadata: metadata, CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
		}); err != nil {
			return err
		}
		persisted, err := store.GetTask(ctx, snapshot.ID, task.ID)
		if err != nil {
			return err
		}
		terminalImport := status == orchestration.TaskCompleted || status == orchestration.TaskFailed ||
			status == orchestration.TaskCancelled || status == orchestration.TaskSkipped ||
			status == orchestration.TaskAwaitingInput
		if persisted.Status != status && (!authoritative || terminalImport) {
			if err := store.SyncTaskStatus(ctx, snapshot.ID, task.ID, status, task.Output); err != nil {
				return err
			}
		}
		if status == orchestration.TaskCompleted && cachePolicy.Enabled && strings.TrimSpace(task.Output) != "" {
			_, cacheErr := store.GetCachedResult(ctx, key, time.Now())
			if cachePolicy.Refresh || errors.Is(cacheErr, orchestration.ErrNotFound) {
				expiresAt := time.Time{}
				if cachePolicy.Expiration > 0 {
					expiresAt = time.Now().Add(cachePolicy.Expiration)
				}
				_, err = store.PutResult(ctx, orchestration.Result{
					RunID: snapshot.ID, TaskID: task.ID, ExecutionKey: key,
					Value: []byte(task.Output), ContentType: "text/plain",
					ExpiresAt: expiresAt, Metadata: map[string]any{"source": "collaboration_task"},
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *Hub) claimOrchestrationTask(
	ctx context.Context,
	snapshot *collaboration.Collaboration,
	task collaboration.CollaborationTask,
) (*orchestration.Attempt, error) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		if runtime != nil && runtime.err != nil {
			return nil, runtime.err
		}
		return nil, errors.New("durable orchestration store unavailable")
	}
	if err := syncOrchestrationSnapshot(ctx, runtime.store, snapshot, h.durableOrchestrationEnforced()); err != nil {
		return nil, err
	}
	timeout := time.Duration(collaboration.ExecutionTimeoutSeconds(task, h.collabExecutionTimeoutOverride())) * time.Second
	if timeout < 30*time.Second {
		timeout = 30 * time.Second
	}
	attempt, err := runtime.store.ClaimTask(ctx, snapshot.ID, task.ID, orchestration.ClaimOptions{
		WorkerID: "local-hub", Lease: timeout + 30*time.Second, Queue: "default",
	})
	if err != nil {
		return nil, err
	}
	return attempt, runtime.store.MarkAttemptRunning(ctx, attempt.ID, attempt.LeaseToken)
}

func (h *Hub) activeOrchestrationAttempt(ctx context.Context, runID, taskID string) (*orchestration.Attempt, error) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		if runtime != nil && runtime.err != nil {
			return nil, runtime.err
		}
		return nil, errors.New("durable orchestration store unavailable")
	}
	return runtime.store.ActiveAttempt(ctx, runID, taskID)
}

func (h *Hub) cachedOrchestrationTaskResult(ctx context.Context, snapshot *collaboration.Collaboration, task collaboration.CollaborationTask) ([]byte, bool) {
	if task.Options == nil || task.Options.CachePolicy != "result" || task.Options.RefreshCache {
		return nil, false
	}
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return nil, false
	}
	if err := syncOrchestrationSnapshot(ctx, runtime.store, snapshot, h.durableOrchestrationEnforced()); err != nil {
		return nil, false
	}
	persisted, err := runtime.store.GetTask(ctx, snapshot.ID, task.ID)
	if err != nil || persisted.ExecutionKey == "" {
		return nil, false
	}
	result, err := runtime.store.GetCachedResult(ctx, persisted.ExecutionKey, time.Now())
	if err != nil {
		return nil, false
	}
	_ = runtime.store.AppendEvent(ctx, orchestration.Event{
		RunID: snapshot.ID, TaskID: task.ID, Type: "task.cache_hit",
		Payload: map[string]any{"result_id": result.ID, "execution_key": persisted.ExecutionKey},
	})
	return append([]byte(nil), result.Value...), true
}

func (h *Hub) orchestrationDispatchFailed(ctx context.Context, attempt *orchestration.Attempt, cause error, task collaboration.CollaborationTask) {
	if attempt == nil {
		return
	}
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	maxRetries := 0
	if task.Options != nil {
		maxRetries = task.Options.MaxRetries
	}
	retryDispatch := task.Options == nil || len(task.Options.RetryOn) == 0
	if task.Options != nil {
		for _, category := range task.Options.RetryOn {
			if category == "dispatch" {
				retryDispatch = true
				break
			}
		}
	}
	_, _, err := runtime.store.FailAttempt(ctx, attempt.ID, attempt.LeaseToken, cause, orchestration.RetryPolicy{
		MaxRetries: maxRetries, BaseDelay: time.Second, MaxDelay: time.Minute, JitterFactor: 0.2,
		RetryIf: func(error) bool { return retryDispatch },
	})
	if err != nil {
		log.Printf("[Orchestration] fail attempt %s: %v", shortOrchestrationID(attempt.ID), err)
	}
}

func (h *Hub) reconcileOrchestration(ctx context.Context, now time.Time) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil {
		return
	}
	_ = runtime.store.HeartbeatWorker(ctx, "local-hub", orchestration.WorkerReady, now)
	_, _ = runtime.store.MarkStaleWorkersOffline(ctx, now.Add(-45*time.Second))
	if expired, err := runtime.store.ExpireDueInputs(ctx, now); err != nil {
		log.Printf("[Orchestration] expire due inputs: %v", err)
	} else if expired > 0 {
		log.Printf("[Orchestration] expired %d overdue input request(s)", expired)
	}
	count, err := runtime.store.ReconcileExpiredLeases(ctx, now, orchestration.RetryPolicy{
		MaxRetries: 2, BaseDelay: time.Second, MaxDelay: time.Minute, JitterFactor: 0.2,
	})
	if err != nil {
		log.Printf("[Orchestration] reconcile expired leases: %v", err)
	} else if count > 0 {
		log.Printf("[Orchestration] reconciled %d expired attempt lease(s)", count)
	}
	h.syncOrchestrationState(ctx)
	if h.collabManager != nil {
		for _, snapshot := range h.collabManager.ListActive() {
			if snapshot == nil || snapshot.Phase != collaboration.PhaseExecuting {
				continue
			}
			sla := snapshot.EffectiveExecutionPolicy().SLASeconds
			if sla > 0 && now.Sub(snapshot.CreatedAt) > time.Duration(sla)*time.Second {
				_, _ = runtime.store.AppendEventOnce(ctx, orchestration.Event{
					RunID: snapshot.ID, Type: "run.sla_breached", CreatedAt: now,
					Payload: map[string]any{"sla_seconds": sla},
				})
			}
		}
	}
	h.dispatchOrchestrationAutomations(ctx)
	if _, err := runtime.store.ProcessDueDeployments(ctx, now, h); err != nil {
		log.Printf("[Orchestration] process scheduled deployments: %v", err)
	}
}

func (h *Hub) dispatchOrchestrationAutomations(ctx context.Context) {
	runtime := h.orchestrationRuntime()
	if runtime == nil || runtime.store == nil || h.collabManager == nil {
		return
	}
	for runID := range h.collabManager.Snapshot() {
		after := runtime.eventCursor[runID]
		for {
			events, err := runtime.store.ListEvents(ctx, runID, after, 200)
			if err != nil {
				log.Printf("[Orchestration] list events for %s: %v", shortOrchestrationID(runID), err)
				break
			}
			for _, event := range events {
				if err := runtime.store.DispatchAutomations(ctx, event, h); err != nil {
					log.Printf("[Orchestration] automation event %d: %v", event.ID, err)
				}
				if _, err := runtime.store.ProcessEventDeployments(ctx, event, h); err != nil {
					log.Printf("[Orchestration] event deployment %d: %v", event.ID, err)
				}
				after = event.ID
				runtime.eventCursor[runID] = after
			}
			if len(events) < 200 {
				break
			}
		}
	}
}

// HandleAutomation implements orchestration.AutomationActionHandler.
func (h *Hub) HandleAutomation(_ context.Context, automation orchestration.Automation, event orchestration.Event) error {
	if automation.ID == "notify-retry-exhaustion" {
		if retry, _ := event.Payload["retry"].(bool); retry {
			return nil
		}
	}
	if automation.ActionType == "log" {
		log.Printf("[Orchestration] automation %s observed %s for run %s",
			automation.Name, event.Type, shortOrchestrationID(event.RunID))
		return nil
	}
	if automation.ActionType == "cleanup" {
		runtime := h.orchestrationRuntime()
		if runtime == nil || runtime.store == nil {
			return errors.New("orchestration store unavailable")
		}
		_, err := runtime.store.DeleteExpiredResults(context.Background(), time.Now())
		return err
	}
	if automation.ActionType != "notify" && automation.ActionType != "escalate" {
		return errors.New("unsupported automation action: " + automation.ActionType)
	}
	channel := "general"
	if h.collabManager != nil && event.RunID != "" {
		if snapshot, err := h.collabManager.GetCollaborationSnapshot(event.RunID); err == nil && snapshot.Channel != "" {
			channel = snapshot.Channel
		}
	}
	message := protocol.NewMessage(
		protocol.MessageTypeSystemInfo,
		channel,
		protocol.AgentInfo{ID: "system", Name: "System", Type: protocol.AgentTypeGeneral},
		"Orchestration "+automation.ActionType+": "+event.Type+" ("+shortOrchestrationID(event.RunID)+")",
	)
	message.Metadata = map[string]any{
		"orchestration_run_id": event.RunID, "orchestration_task_id": event.TaskID,
		"orchestration_attempt_id": event.AttemptID, "orchestration_event_id": event.ID,
		"automation_id": automation.ID,
	}
	return h.SendMessage(message)
}

// LaunchDeployment implements orchestration.DeploymentLauncher by creating and
// starting an existing runbook definition.
func (h *Hub) LaunchDeployment(
	_ context.Context,
	deployment orchestration.Deployment,
	parameters map[string]any,
) (string, error) {
	request := RunbookCreateRequest{
		Channel:        stringParameter(parameters, "channel", "general"),
		CreatedBy:      stringParameter(parameters, "created_by", "orchestration"),
		ExecutionMode:  stringParameter(parameters, "execution_mode", ""),
		SourceRepoPath: stringParameter(parameters, "source_repo_path", ""),
		RunInputs:      stringMapParameter(parameters, "run_inputs"),
		AgentIDs:       stringSliceParameter(parameters, "agent_ids"),
	}
	result, err := h.InstantiateDefinition(deployment.DefinitionID, deployment.DefinitionVersion, request)
	if err != nil {
		return "", err
	}
	if _, err := h.SubmitRunbookForReview(result.CollaborationID); err != nil {
		return "", err
	}
	if _, err := h.StartRunbook(result.CollaborationID, request.RunInputs); err != nil {
		return "", err
	}
	return result.CollaborationID, nil
}

func stringParameter(parameters map[string]any, key, fallback string) string {
	if value, ok := parameters[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func stringMapParameter(parameters map[string]any, key string) map[string]string {
	raw, ok := parameters[key]
	if !ok {
		return nil
	}
	out := map[string]string{}
	switch values := raw.(type) {
	case map[string]string:
		for itemKey, value := range values {
			out[itemKey] = value
		}
	case map[string]any:
		for itemKey, value := range values {
			if text, ok := value.(string); ok {
				out[itemKey] = text
			}
		}
	}
	return out
}

func stringSliceParameter(parameters map[string]any, key string) []string {
	raw, ok := parameters[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	default:
		return nil
	}
}

func orchestrationRunStatus(phase collaboration.CollaborationPhase, paused bool) orchestration.RunStatus {
	if paused {
		return orchestration.RunSuspended
	}
	switch phase {
	case collaboration.PhaseExecuting:
		return orchestration.RunRunning
	case collaboration.PhaseCompleted:
		return orchestration.RunCompleted
	case collaboration.PhaseCancelled:
		return orchestration.RunCancelled
	default:
		return orchestration.RunPending
	}
}

func orchestrationTaskStatus(task collaboration.CollaborationTask) orchestration.TaskStatus {
	if task.AwaitingApproval {
		return orchestration.TaskAwaitingInput
	}
	switch task.Status {
	case collaboration.TaskInProgress:
		return orchestration.TaskRunning
	case collaboration.TaskCompleted:
		return orchestration.TaskCompleted
	case collaboration.TaskBlocked:
		return orchestration.TaskFailed
	default:
		return orchestration.TaskPending
	}
}

func orchestrationContextHash(snapshot *collaboration.Collaboration, task collaboration.CollaborationTask) string {
	payload := map[string]any{
		"source_repo_path":  snapshot.SourceRepoPath,
		"working_directory": snapshot.WorkingDirectory,
		"context_paths":     collaboration.InferTaskContextPaths(task, snapshot.SourceRepoPath),
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func stringMapToAny(values map[string]string) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func shortOrchestrationID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
