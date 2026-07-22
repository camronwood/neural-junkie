package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Store) RegisterWorker(ctx context.Context, worker Worker) error {
	if worker.ID == "" {
		return errors.New("worker id is required")
	}
	if worker.Queue == "" {
		worker.Queue = "default"
	}
	if worker.Status == "" {
		worker.Status = WorkerReady
	}
	if worker.LastHeartbeat.IsZero() {
		worker.LastHeartbeat = time.Now().UTC()
	}
	tags, err := json.Marshal(normalizeStrings(worker.Capabilities))
	if err != nil {
		return err
	}
	metadata, err := marshalObject(worker.Metadata)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_workers(
id,queue_name,capabilities_json,status,last_heartbeat_at,metadata_json)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET queue_name=excluded.queue_name,
capabilities_json=excluded.capabilities_json,status=excluded.status,
last_heartbeat_at=excluded.last_heartbeat_at,metadata_json=excluded.metadata_json`,
			worker.ID, worker.Queue, string(tags), worker.Status,
			unixMillis(worker.LastHeartbeat), metadata)
		return err
	})
}

func (s *Store) HeartbeatWorker(ctx context.Context, id string, status WorkerStatus, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if status == "" {
		status = WorkerReady
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE orchestration_workers
SET status=?,last_heartbeat_at=? WHERE id=?`, status, unixMillis(now), id)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Store) MarkStaleWorkersOffline(ctx context.Context, before time.Time) (int, error) {
	count := 0
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE orchestration_workers SET status=?
WHERE status<>? AND last_heartbeat_at<?`, WorkerOffline, WorkerOffline, unixMillis(before))
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		count = int(n)
		return nil
	})
	return count, err
}

func (s *Store) ListWorkers(ctx context.Context) ([]Worker, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,queue_name,capabilities_json,status,
last_heartbeat_at,metadata_json FROM orchestration_workers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workers []Worker
	for rows.Next() {
		var worker Worker
		var capabilities, status, metadata string
		var heartbeat int64
		if err := rows.Scan(&worker.ID, &worker.Queue, &capabilities, &status,
			&heartbeat, &metadata); err != nil {
			return nil, err
		}
		worker.Status = WorkerStatus(status)
		worker.LastHeartbeat = fromUnixMillis(heartbeat)
		worker.Metadata = unmarshalObject(metadata)
		_ = json.Unmarshal([]byte(capabilities), &worker.Capabilities)
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

// ClaimNextTask polls a queue in stable order and atomically claims the first
// task compatible with the worker.
func (s *Store) ClaimNextTask(ctx context.Context, opts ClaimOptions) (*Task, *Attempt, error) {
	if opts.Queue == "" {
		opts.Queue = "default"
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
		opts.Now = now
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.run_id,t.id
FROM orchestration_tasks t
JOIN orchestration_runs r ON r.id=t.run_id
WHERE r.status=? AND t.queue_name=? AND t.status IN ('pending','retrying')
 AND (t.next_attempt_at=0 OR t.next_attempt_at<=?)
ORDER BY t.next_attempt_at,t.created_at LIMIT 100`,
		RunRunning, opts.Queue, unixMillis(now))
	if err != nil {
		return nil, nil, err
	}
	var candidates [][2]string
	for rows.Next() {
		var candidate [2]string
		if err := rows.Scan(&candidate[0], &candidate[1]); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	for _, candidate := range candidates {
		attempt, err := s.ClaimTask(ctx, candidate[0], candidate[1], opts)
		if err != nil {
			if errors.Is(err, ErrNotReady) || errors.Is(err, ErrConcurrencyFull) || errors.Is(err, ErrConflict) {
				continue
			}
			return nil, nil, err
		}
		task, err := s.GetTask(ctx, candidate[0], candidate[1])
		return task, attempt, err
	}
	return nil, nil, ErrNotReady
}

// SpawnTask adds a bounded runtime task to an active run.
func (s *Store) SpawnTask(ctx context.Context, parentTaskID string, task Task, maxTasks int) error {
	if task.RunID == "" || task.ID == "" {
		return errors.New("run id and task id are required")
	}
	if maxTasks <= 0 {
		maxTasks = 100
	}
	now := time.Now().UTC()
	if task.Status == "" {
		task.Status = TaskPending
	}
	if task.Queue == "" {
		task.Queue = "default"
	}
	tags, err := json.Marshal(normalizeStrings(task.CapabilityTags))
	if err != nil {
		return err
	}
	metadata, err := marshalObject(task.Metadata)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var runStatus string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM orchestration_runs WHERE id=?`, task.RunID).Scan(&runStatus); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if RunStatus(runStatus) != RunRunning {
			return ErrNotReady
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM orchestration_tasks WHERE run_id=?`, task.RunID).Scan(&count); err != nil {
			return err
		}
		if count >= maxTasks {
			return ErrConcurrencyFull
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_tasks(
id,run_id,title,status,queue_name,capability_tags_json,max_retries,timeout_ms,
attempt_count,next_attempt_at,execution_key,cache_enabled,cache_expiration_ms,
cache_refresh,version,metadata_json,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,0,0,?,?,?, ?,1,?,?,?)`,
			task.ID, task.RunID, task.Title, task.Status, task.Queue, string(tags),
			task.MaxRetries, task.Timeout.Milliseconds(), task.ExecutionKey,
			boolInt(task.CachePolicy.Enabled), task.CachePolicy.Expiration.Milliseconds(),
			boolInt(task.CachePolicy.Refresh), metadata, unixMillis(now), unixMillis(now))
		if err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: task.RunID, TaskID: task.ID, Type: "task.spawned", CreatedAt: now,
			Payload: map[string]any{"parent_task_id": parentTaskID},
		})
	})
}

func (s *Store) UpsertDeployment(ctx context.Context, deployment Deployment) error {
	if deployment.ID == "" || deployment.DefinitionID == "" {
		return errors.New("deployment id and definition id are required")
	}
	if deployment.Queue == "" {
		deployment.Queue = "default"
	}
	now := time.Now().UTC()
	if deployment.CreatedAt.IsZero() {
		deployment.CreatedAt = now
	}
	deployment.UpdatedAt = now
	filter, err := marshalObject(deployment.EventFilter)
	if err != nil {
		return err
	}
	parameters, err := marshalObject(deployment.Parameters)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_deployments(
id,definition_id,definition_version,queue_name,schedule,event_filter_json,enabled,
parameters_json,next_run_at,last_triggered_at,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET definition_id=excluded.definition_id,
definition_version=excluded.definition_version,queue_name=excluded.queue_name,
schedule=excluded.schedule,event_filter_json=excluded.event_filter_json,
enabled=excluded.enabled,parameters_json=excluded.parameters_json,
next_run_at=excluded.next_run_at,updated_at=excluded.updated_at`,
			deployment.ID, deployment.DefinitionID, deployment.DefinitionVersion,
			deployment.Queue, deployment.Schedule, filter, boolInt(deployment.Enabled),
			parameters, unixMillis(deployment.NextRunAt), unixMillis(deployment.LastTriggeredAt),
			unixMillis(deployment.CreatedAt), unixMillis(now))
		return err
	})
}

func (s *Store) DueDeployments(ctx context.Context, now time.Time) ([]Deployment, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,definition_id,definition_version,queue_name,
schedule,event_filter_json,enabled,parameters_json,next_run_at,last_triggered_at,created_at,updated_at
FROM orchestration_deployments WHERE enabled=1 AND next_run_at>0 AND next_run_at<=?
ORDER BY next_run_at`, unixMillis(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deployments []Deployment
	for rows.Next() {
		var item Deployment
		var filter, parameters string
		var enabled int
		var next, last, created, updated int64
		if err := rows.Scan(&item.ID, &item.DefinitionID, &item.DefinitionVersion, &item.Queue,
			&item.Schedule, &filter, &enabled, &parameters, &next, &last, &created, &updated); err != nil {
			return nil, err
		}
		item.Enabled = enabled != 0
		item.EventFilter, item.Parameters = unmarshalObject(filter), unmarshalObject(parameters)
		item.NextRunAt, item.LastTriggeredAt = fromUnixMillis(next), fromUnixMillis(last)
		item.CreatedAt, item.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
		deployments = append(deployments, item)
	}
	return deployments, rows.Err()
}

func (s *Store) MarkDeploymentTriggered(ctx context.Context, id string, triggeredAt, nextRunAt time.Time) error {
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE orchestration_deployments
SET last_triggered_at=?,next_run_at=?,updated_at=? WHERE id=?`,
			unixMillis(triggeredAt), unixMillis(nextRunAt), unixMillis(triggeredAt), id)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return ErrNotFound
		}
		return nil
	})
}

func NextScheduleTime(schedule string, after time.Time) (time.Time, error) {
	schedule = strings.TrimSpace(schedule)
	if schedule == "" {
		return time.Time{}, nil
	}
	if strings.HasPrefix(schedule, "@every ") {
		duration, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(schedule, "@every ")))
		if err != nil || duration <= 0 {
			return time.Time{}, fmt.Errorf("invalid @every schedule %q", schedule)
		}
		return after.Add(duration), nil
	}
	if strings.HasPrefix(schedule, "@daily ") {
		parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(schedule, "@daily ")), ":")
		if len(parts) != 2 {
			return time.Time{}, fmt.Errorf("invalid @daily schedule %q", schedule)
		}
		hour, hourErr := strconv.Atoi(parts[0])
		minute, minuteErr := strconv.Atoi(parts[1])
		if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return time.Time{}, fmt.Errorf("invalid @daily schedule %q", schedule)
		}
		next := time.Date(after.Year(), after.Month(), after.Day(), hour, minute, 0, 0, after.Location())
		if !next.After(after) {
			next = next.Add(24 * time.Hour)
		}
		return next, nil
	}
	return time.Time{}, fmt.Errorf("unsupported schedule %q; use @every or @daily", schedule)
}

func (s *Store) ProcessDueDeployments(ctx context.Context, now time.Time, launcher DeploymentLauncher) (int, error) {
	if launcher == nil {
		return 0, errors.New("deployment launcher is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	deployments, err := s.DueDeployments(ctx, now)
	if err != nil {
		return 0, err
	}
	launched := 0
	for _, deployment := range deployments {
		runID, err := launcher.LaunchDeployment(ctx, deployment, deployment.Parameters)
		if err != nil {
			return launched, err
		}
		next, err := NextScheduleTime(deployment.Schedule, now)
		if err != nil {
			return launched, err
		}
		if err := s.MarkDeploymentTriggered(ctx, deployment.ID, now, next); err != nil {
			return launched, err
		}
		_ = s.AppendEvent(ctx, Event{
			RunID: runID, Type: "deployment.triggered", CreatedAt: now,
			Payload: map[string]any{"deployment_id": deployment.ID, "next_run_at": next},
		})
		launched++
	}
	return launched, nil
}

func (s *Store) EventDeployments(ctx context.Context, event Event) ([]Deployment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,definition_id,definition_version,queue_name,
schedule,event_filter_json,enabled,parameters_json,next_run_at,last_triggered_at,created_at,updated_at
FROM orchestration_deployments WHERE enabled=1 AND event_filter_json<>'{}'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deployments []Deployment
	for rows.Next() {
		var item Deployment
		var filter, parameters string
		var enabled int
		var next, last, created, updated int64
		if err := rows.Scan(&item.ID, &item.DefinitionID, &item.DefinitionVersion, &item.Queue,
			&item.Schedule, &filter, &enabled, &parameters, &next, &last, &created, &updated); err != nil {
			return nil, err
		}
		item.EventFilter, item.Parameters = unmarshalObject(filter), unmarshalObject(parameters)
		pattern, _ := item.EventFilter["event_type"].(string)
		matched, matchErr := path.Match(pattern, event.Type)
		if pattern == "" || matchErr != nil || !matched {
			continue
		}
		item.Enabled = enabled != 0
		item.NextRunAt, item.LastTriggeredAt = fromUnixMillis(next), fromUnixMillis(last)
		item.CreatedAt, item.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
		deployments = append(deployments, item)
	}
	return deployments, rows.Err()
}

func (s *Store) ProcessEventDeployments(ctx context.Context, event Event, launcher DeploymentLauncher) (int, error) {
	if launcher == nil {
		return 0, errors.New("deployment launcher is required")
	}
	deployments, err := s.EventDeployments(ctx, event)
	if err != nil {
		return 0, err
	}
	launched := 0
	now := time.Now().UTC()
	for _, deployment := range deployments {
		parameters := make(map[string]any, len(deployment.Parameters)+1)
		for key, value := range deployment.Parameters {
			parameters[key] = value
		}
		parameters["trigger_event"] = event
		runID, err := launcher.LaunchDeployment(ctx, deployment, parameters)
		if err != nil {
			return launched, err
		}
		if err := s.MarkDeploymentTriggered(ctx, deployment.ID, now, deployment.NextRunAt); err != nil {
			return launched, err
		}
		_ = s.AppendEvent(ctx, Event{
			RunID: runID, Type: "deployment.event_triggered", CreatedAt: now,
			Payload: map[string]any{"deployment_id": deployment.ID, "source_event_id": event.ID},
		})
		launched++
	}
	return launched, nil
}

func (s *Store) UpsertAutomation(ctx context.Context, automation Automation) error {
	if automation.ID == "" {
		automation.ID = uuid.NewString()
	}
	if automation.Name == "" || automation.EventType == "" || automation.ActionType == "" {
		return errors.New("automation name, event type, and action type are required")
	}
	if automation.Posture == "" {
		automation.Posture = "reactive"
	}
	if automation.Threshold <= 0 {
		automation.Threshold = 1
	}
	now := time.Now().UTC()
	if automation.CreatedAt.IsZero() {
		automation.CreatedAt = now
	}
	action, err := marshalObject(automation.Action)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_automations(
id,name,event_type,posture,threshold,within_ms,action_type,action_json,enabled,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name,event_type=excluded.event_type,
posture=excluded.posture,threshold=excluded.threshold,within_ms=excluded.within_ms,
action_type=excluded.action_type,action_json=excluded.action_json,
enabled=excluded.enabled,updated_at=excluded.updated_at`,
			automation.ID, automation.Name, automation.EventType, automation.Posture,
			automation.Threshold, automation.Within.Milliseconds(), automation.ActionType,
			action, boolInt(automation.Enabled), unixMillis(automation.CreatedAt), unixMillis(now))
		return err
	})
}

func (s *Store) MatchingAutomations(ctx context.Context, eventType string) ([]Automation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,event_type,posture,threshold,within_ms,
action_type,action_json,enabled,created_at,updated_at
FROM orchestration_automations WHERE enabled=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var automations []Automation
	for rows.Next() {
		var item Automation
		var within, created, updated int64
		var action string
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.EventType, &item.Posture,
			&item.Threshold, &within, &item.ActionType, &action, &enabled,
			&created, &updated); err != nil {
			return nil, err
		}
		matched, matchErr := path.Match(item.EventType, eventType)
		if matchErr != nil || !matched {
			continue
		}
		item.Within = time.Duration(within) * time.Millisecond
		item.Action = unmarshalObject(action)
		item.Enabled = enabled != 0
		item.CreatedAt, item.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
		automations = append(automations, item)
	}
	return automations, rows.Err()
}

// DispatchAutomations executes matching actions outside the state transaction.
func (s *Store) DispatchAutomations(ctx context.Context, event Event, handler AutomationActionHandler) error {
	if handler == nil {
		return nil
	}
	automations, err := s.MatchingAutomations(ctx, event.Type)
	if err != nil {
		return err
	}
	var failures []string
	for _, automation := range automations {
		claimed, err := s.claimAutomationDelivery(ctx, automation.ID, event.ID)
		if err != nil {
			failures = append(failures, automation.Name+": "+err.Error())
			continue
		}
		if !claimed {
			continue
		}
		handleErr := handler.HandleAutomation(ctx, automation, event)
		if err := s.finishAutomationDelivery(ctx, automation.ID, event.ID, handleErr); err != nil {
			failures = append(failures, automation.Name+": "+err.Error())
			continue
		}
		if handleErr != nil {
			failures = append(failures, automation.Name+": "+handleErr.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func (s *Store) claimAutomationDelivery(ctx context.Context, automationID string, eventID int64) (bool, error) {
	if eventID <= 0 {
		return true, nil
	}
	claimed := false
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC()
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO orchestration_automation_deliveries(
automation_id,event_id,status,created_at,updated_at) VALUES(?,?,'running',?,?)`,
			automationID, eventID, unixMillis(now), unixMillis(now))
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		claimed = count == 1
		return nil
	})
	return claimed, err
}

func (s *Store) finishAutomationDelivery(ctx context.Context, automationID string, eventID int64, handleErr error) error {
	if eventID <= 0 {
		return nil
	}
	status, errorText := "completed", ""
	if handleErr != nil {
		status, errorText = "failed", handleErr.Error()
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE orchestration_automation_deliveries
SET status=?,error_text=?,updated_at=? WHERE automation_id=? AND event_id=?`,
			status, errorText, unixMillis(time.Now().UTC()), automationID, eventID)
		return err
	})
}
