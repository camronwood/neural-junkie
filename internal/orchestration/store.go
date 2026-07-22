package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "orchestration.db"), nil
}

func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite is the local-first backend. A single connection plus explicit
	// transactions gives us serializable claims without process-local races.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=FULL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS orchestration_runs (
  id TEXT PRIMARY KEY,
  definition_id TEXT NOT NULL DEFAULT '',
  definition_version INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  max_concurrency INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 1,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS orchestration_tasks (
  id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  queue_name TEXT NOT NULL DEFAULT 'default',
  capability_tags_json TEXT NOT NULL DEFAULT '[]',
  max_retries INTEGER NOT NULL DEFAULT 0,
  timeout_ms INTEGER NOT NULL DEFAULT 0,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  next_attempt_at INTEGER NOT NULL DEFAULT 0,
  execution_key TEXT NOT NULL DEFAULT '',
  cache_enabled INTEGER NOT NULL DEFAULT 0,
  cache_expiration_ms INTEGER NOT NULL DEFAULT 0,
  cache_refresh INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 1,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(run_id, id),
  FOREIGN KEY(run_id) REFERENCES orchestration_runs(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orch_tasks_ready
  ON orchestration_tasks(run_id, status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_orch_tasks_queue
  ON orchestration_tasks(queue_name, status, next_attempt_at);
CREATE TABLE IF NOT EXISTS orchestration_attempts (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  attempt_number INTEGER NOT NULL,
  status TEXT NOT NULL,
  worker_id TEXT NOT NULL DEFAULT '',
  lease_token TEXT NOT NULL,
  lease_expires_at INTEGER NOT NULL,
  heartbeat_at INTEGER NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER NOT NULL DEFAULT 0,
  error_text TEXT NOT NULL DEFAULT '',
  result_id TEXT NOT NULL DEFAULT '',
  UNIQUE(run_id, task_id, attempt_number),
  FOREIGN KEY(run_id, task_id) REFERENCES orchestration_tasks(run_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orch_attempts_active
  ON orchestration_attempts(run_id, status, lease_expires_at);
CREATE TABLE IF NOT EXISTS orchestration_inputs (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  schema_json TEXT NOT NULL DEFAULT '{}',
  initial_json TEXT NOT NULL DEFAULT 'null',
  decision_key TEXT NOT NULL DEFAULT '',
  requester TEXT NOT NULL DEFAULT '',
  approver TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  response_json TEXT NOT NULL DEFAULT 'null',
  expires_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  resolved_at INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orch_inputs_decision
  ON orchestration_inputs(run_id, decision_key)
  WHERE decision_key <> '' AND status = 'pending';
CREATE TABLE IF NOT EXISTS orchestration_results (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  attempt_id TEXT NOT NULL DEFAULT '',
  execution_key TEXT NOT NULL DEFAULT '',
  value_blob BLOB NOT NULL,
  content_type TEXT NOT NULL DEFAULT 'application/json',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  expires_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orch_results_key
  ON orchestration_results(execution_key, created_at DESC);
CREATE TABLE IF NOT EXISTS orchestration_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL DEFAULT '',
  task_id TEXT NOT NULL DEFAULT '',
  attempt_id TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orch_events_run ON orchestration_events(run_id, id);
CREATE TABLE IF NOT EXISTS orchestration_workers (
  id TEXT PRIMARY KEY,
  queue_name TEXT NOT NULL DEFAULT 'default',
  capabilities_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL,
  last_heartbeat_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS orchestration_deployments (
  id TEXT PRIMARY KEY,
  definition_id TEXT NOT NULL,
  definition_version INTEGER NOT NULL DEFAULT 0,
  queue_name TEXT NOT NULL DEFAULT 'default',
  schedule TEXT NOT NULL DEFAULT '',
  event_filter_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  parameters_json TEXT NOT NULL DEFAULT '{}',
  next_run_at INTEGER NOT NULL DEFAULT 0,
  last_triggered_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS orchestration_automations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  event_type TEXT NOT NULL,
  posture TEXT NOT NULL DEFAULT 'reactive',
  threshold INTEGER NOT NULL DEFAULT 1,
  within_ms INTEGER NOT NULL DEFAULT 0,
  action_type TEXT NOT NULL,
  action_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS orchestration_automation_deliveries (
  automation_id TEXT NOT NULL,
  event_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  error_text TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(automation_id,event_id),
  FOREIGN KEY(automation_id) REFERENCES orchestration_automations(id) ON DELETE CASCADE,
  FOREIGN KEY(event_id) REFERENCES orchestration_events(id) ON DELETE CASCADE
);`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	for _, statement := range []string{
		`ALTER TABLE orchestration_deployments ADD COLUMN next_run_at INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE orchestration_deployments ADD COLUMN last_triggered_at INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.Exec(statement); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	return nil
}

func (s *Store) transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) UpsertRun(ctx context.Context, run Run) error {
	if strings.TrimSpace(run.ID) == "" {
		return errors.New("run id is required")
	}
	if run.Status == "" {
		run.Status = RunPending
	}
	now := time.Now().UTC()
	if !run.UpdatedAt.IsZero() {
		now = run.UpdatedAt.UTC()
	}
	created := run.CreatedAt.UTC()
	if created.IsZero() {
		created = now
	}
	raw, err := marshalObject(run.Metadata)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var previous string
		lookupErr := tx.QueryRowContext(ctx, `SELECT status FROM orchestration_runs WHERE id=?`, run.ID).Scan(&previous)
		if lookupErr != nil && lookupErr != sql.ErrNoRows {
			return lookupErr
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_runs(
id, definition_id, definition_version, status, max_concurrency, version, metadata_json, created_at, updated_at)
VALUES(?,?,?,?,?,1,?,?,?)
ON CONFLICT(id) DO UPDATE SET
 definition_id=excluded.definition_id,
 definition_version=excluded.definition_version,
 status=excluded.status,
 max_concurrency=excluded.max_concurrency,
 metadata_json=excluded.metadata_json,
 updated_at=excluded.updated_at,
 version=orchestration_runs.version+1
WHERE orchestration_runs.definition_id<>excluded.definition_id
 OR orchestration_runs.definition_version<>excluded.definition_version
 OR orchestration_runs.status<>excluded.status
 OR orchestration_runs.max_concurrency<>excluded.max_concurrency
 OR orchestration_runs.metadata_json<>excluded.metadata_json`,
			run.ID, run.DefinitionID, run.DefinitionVersion, run.Status, run.MaxConcurrency,
			raw, unixMillis(created), unixMillis(now))
		if err != nil {
			return err
		}
		if lookupErr == sql.ErrNoRows {
			return appendEventTx(ctx, tx, Event{RunID: run.ID, Type: "run.created", CreatedAt: now})
		}
		if previous != string(run.Status) {
			return appendEventTx(ctx, tx, Event{
				RunID: run.ID, Type: "run.external_transition", CreatedAt: now,
				Payload: map[string]any{"from": previous, "to": run.Status},
			})
		}
		return nil
	})
}

func (s *Store) GetRun(ctx context.Context, id string) (*Run, error) {
	var run Run
	var status, metadata string
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT id, definition_id, definition_version, status,
max_concurrency, version, metadata_json, created_at, updated_at
FROM orchestration_runs WHERE id=?`, id).Scan(
		&run.ID, &run.DefinitionID, &run.DefinitionVersion, &status,
		&run.MaxConcurrency, &run.Version, &metadata, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	run.Status = RunStatus(status)
	run.Metadata = unmarshalObject(metadata)
	run.CreatedAt, run.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
	return &run, nil
}

func (s *Store) TransitionRun(ctx context.Context, id string, from []RunStatus, to RunStatus) error {
	if to == "" {
		return errors.New("target run status is required")
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var current string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM orchestration_runs WHERE id=?`, id).Scan(&current); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if len(from) > 0 && !containsRunStatus(from, RunStatus(current)) {
			return fmt.Errorf("%w: run %s is %s", ErrConflict, id, current)
		}
		now := time.Now().UTC()
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_runs SET status=?, version=version+1, updated_at=? WHERE id=?`,
			to, unixMillis(now), id); err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: id, Type: "run.transitioned", CreatedAt: now,
			Payload: map[string]any{"from": current, "to": to},
		})
	})
}

func (s *Store) UpsertTask(ctx context.Context, task Task) error {
	if task.RunID == "" || task.ID == "" {
		return errors.New("run id and task id are required")
	}
	if task.Status == "" {
		task.Status = TaskPending
	}
	if task.Queue == "" {
		task.Queue = "default"
	}
	now := time.Now().UTC()
	if !task.UpdatedAt.IsZero() {
		now = task.UpdatedAt.UTC()
	}
	created := task.CreatedAt.UTC()
	if created.IsZero() {
		created = now
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
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
SELECT 1 FROM orchestration_tasks WHERE run_id=? AND id=?)`, task.RunID, task.ID).Scan(&exists); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_tasks(
id, run_id, title, status, queue_name, capability_tags_json, max_retries, timeout_ms,
attempt_count, next_attempt_at, execution_key, cache_enabled, cache_expiration_ms,
cache_refresh, version, metadata_json, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?, ?,?,?,1,?,?,?)
ON CONFLICT(run_id,id) DO UPDATE SET
 title=excluded.title,
 queue_name=excluded.queue_name,
 capability_tags_json=excluded.capability_tags_json,
 max_retries=excluded.max_retries,
 timeout_ms=excluded.timeout_ms,
 execution_key=excluded.execution_key,
 cache_enabled=excluded.cache_enabled,
 cache_expiration_ms=excluded.cache_expiration_ms,
 cache_refresh=excluded.cache_refresh,
 metadata_json=excluded.metadata_json,
 updated_at=excluded.updated_at,
 version=orchestration_tasks.version+1
WHERE orchestration_tasks.title<>excluded.title
 OR orchestration_tasks.queue_name<>excluded.queue_name
 OR orchestration_tasks.capability_tags_json<>excluded.capability_tags_json
 OR orchestration_tasks.max_retries<>excluded.max_retries
 OR orchestration_tasks.timeout_ms<>excluded.timeout_ms
 OR orchestration_tasks.execution_key<>excluded.execution_key
 OR orchestration_tasks.cache_enabled<>excluded.cache_enabled
 OR orchestration_tasks.cache_expiration_ms<>excluded.cache_expiration_ms
 OR orchestration_tasks.cache_refresh<>excluded.cache_refresh
 OR orchestration_tasks.metadata_json<>excluded.metadata_json`,
			task.ID, task.RunID, task.Title, task.Status, task.Queue, string(tags),
			task.MaxRetries, task.Timeout.Milliseconds(), task.AttemptCount, unixMillis(task.NextAttemptAt),
			task.ExecutionKey, boolInt(task.CachePolicy.Enabled), task.CachePolicy.Expiration.Milliseconds(),
			boolInt(task.CachePolicy.Refresh), metadata, unixMillis(created), unixMillis(now))
		if err != nil {
			return err
		}
		if exists == 0 {
			return appendEventTx(ctx, tx, Event{
				RunID: task.RunID, TaskID: task.ID, Type: "task.created", CreatedAt: now,
			})
		}
		return nil
	})
}

func (s *Store) GetTask(ctx context.Context, runID, taskID string) (*Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, run_id, title, status, queue_name,
capability_tags_json, max_retries, timeout_ms, attempt_count, next_attempt_at,
execution_key, cache_enabled, cache_expiration_ms, cache_refresh, version,
metadata_json, created_at, updated_at
FROM orchestration_tasks WHERE run_id=? AND id=?`, runID, taskID)
	return scanTask(row)
}

func (s *Store) ListTasks(ctx context.Context, runID string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, run_id, title, status, queue_name,
capability_tags_json, max_retries, timeout_ms, attempt_count, next_attempt_at,
execution_key, cache_enabled, cache_expiration_ms, cache_refresh, version,
metadata_json, created_at, updated_at
FROM orchestration_tasks WHERE run_id=? ORDER BY created_at,id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

// SyncTaskStatus imports a status observed by a legacy executor during the
// shadow-write rollout. Once durable scheduling is authoritative, callers
// should use ClaimTask, CompleteAttempt, and FailAttempt instead.
func (s *Store) SyncTaskStatus(ctx context.Context, runID, taskID string, status TaskStatus, output string) error {
	if status == "" {
		return errors.New("task status is required")
	}
	now := time.Now().UTC()
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var current string
		err := tx.QueryRowContext(ctx, `SELECT status FROM orchestration_tasks WHERE run_id=? AND id=?`,
			runID, taskID).Scan(&current)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if current == string(status) {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,version=version+1,updated_at=? WHERE run_id=? AND id=?`,
			status, unixMillis(now), runID, taskID); err != nil {
			return err
		}
		if status == TaskCompleted || status == TaskFailed || status == TaskCancelled || status == TaskSkipped {
			attemptStatus := AttemptSucceeded
			if status != TaskCompleted {
				attemptStatus = AttemptFailed
			}
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts SET
status=?,completed_at=?,error_text=CASE WHEN ?='' THEN error_text ELSE ? END
WHERE id=(SELECT id FROM orchestration_attempts
  WHERE run_id=? AND task_id=? AND status IN ('claimed','running')
  ORDER BY attempt_number DESC LIMIT 1)`,
				attemptStatus, unixMillis(now), output, output, runID, taskID); err != nil {
				return err
			}
		}
		return appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, Type: "task.external_transition", CreatedAt: now,
			Payload: map[string]any{"from": current, "to": status},
		})
	})
}

func (s *Store) ActiveAttempt(ctx context.Context, runID, taskID string) (*Attempt, error) {
	var attempt Attempt
	var status string
	var leaseExpires, heartbeat, started, completed int64
	err := s.db.QueryRowContext(ctx, `SELECT id,run_id,task_id,attempt_number,status,worker_id,
lease_token,lease_expires_at,heartbeat_at,started_at,completed_at,error_text,result_id
FROM orchestration_attempts
WHERE run_id=? AND task_id=? AND status IN ('claimed','running')
ORDER BY attempt_number DESC LIMIT 1`, runID, taskID).Scan(
		&attempt.ID, &attempt.RunID, &attempt.TaskID, &attempt.Number, &status, &attempt.WorkerID,
		&attempt.LeaseToken, &leaseExpires, &heartbeat, &started, &completed,
		&attempt.Error, &attempt.ResultID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	attempt.Status = AttemptStatus(status)
	attempt.LeaseExpiresAt, attempt.HeartbeatAt, attempt.StartedAt, attempt.CompletedAt =
		fromUnixMillis(leaseExpires), fromUnixMillis(heartbeat), fromUnixMillis(started), fromUnixMillis(completed)
	return &attempt, nil
}

func (s *Store) GetAttempt(ctx context.Context, attemptID string) (*Attempt, error) {
	var attempt Attempt
	var status string
	var leaseExpires, heartbeat, started, completed int64
	err := s.db.QueryRowContext(ctx, `SELECT id,run_id,task_id,attempt_number,status,worker_id,
lease_token,lease_expires_at,heartbeat_at,started_at,completed_at,error_text,result_id
FROM orchestration_attempts WHERE id=?`, attemptID).Scan(
		&attempt.ID, &attempt.RunID, &attempt.TaskID, &attempt.Number, &status, &attempt.WorkerID,
		&attempt.LeaseToken, &leaseExpires, &heartbeat, &started, &completed,
		&attempt.Error, &attempt.ResultID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	attempt.Status = AttemptStatus(status)
	attempt.LeaseExpiresAt, attempt.HeartbeatAt, attempt.StartedAt, attempt.CompletedAt =
		fromUnixMillis(leaseExpires), fromUnixMillis(heartbeat), fromUnixMillis(started), fromUnixMillis(completed)
	return &attempt, nil
}

func (s *Store) ListAttempts(ctx context.Context, runID string) ([]Attempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,run_id,task_id,attempt_number,status,worker_id,
lease_token,lease_expires_at,heartbeat_at,started_at,completed_at,error_text,result_id
FROM orchestration_attempts WHERE run_id=? ORDER BY task_id,attempt_number`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attempts []Attempt
	for rows.Next() {
		var attempt Attempt
		var status string
		var leaseExpires, heartbeat, started, completed int64
		if err := rows.Scan(
			&attempt.ID, &attempt.RunID, &attempt.TaskID, &attempt.Number, &status, &attempt.WorkerID,
			&attempt.LeaseToken, &leaseExpires, &heartbeat, &started, &completed,
			&attempt.Error, &attempt.ResultID,
		); err != nil {
			return nil, err
		}
		attempt.Status = AttemptStatus(status)
		attempt.LeaseExpiresAt, attempt.HeartbeatAt, attempt.StartedAt, attempt.CompletedAt =
			fromUnixMillis(leaseExpires), fromUnixMillis(heartbeat), fromUnixMillis(started), fromUnixMillis(completed)
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

type rowScanner interface {
	Scan(...any) error
}

func scanTask(row rowScanner) (*Task, error) {
	var task Task
	var status, tags, metadata string
	var timeoutMS, nextAttempt, cacheExpiration, created, updated int64
	var cacheEnabled, cacheRefresh int
	err := row.Scan(&task.ID, &task.RunID, &task.Title, &status, &task.Queue,
		&tags, &task.MaxRetries, &timeoutMS, &task.AttemptCount, &nextAttempt,
		&task.ExecutionKey, &cacheEnabled, &cacheExpiration, &cacheRefresh,
		&task.Version, &metadata, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	task.Status = TaskStatus(status)
	task.Timeout = time.Duration(timeoutMS) * time.Millisecond
	task.NextAttemptAt = fromUnixMillis(nextAttempt)
	task.CachePolicy = CachePolicy{
		Enabled: cacheEnabled != 0, Expiration: time.Duration(cacheExpiration) * time.Millisecond,
		Refresh: cacheRefresh != 0,
	}
	_ = json.Unmarshal([]byte(tags), &task.CapabilityTags)
	task.Metadata = unmarshalObject(metadata)
	task.CreatedAt, task.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
	return &task, nil
}

// ClaimTask atomically enforces run-wide concurrency and creates an immutable
// attempt. The caller must send work only after this method succeeds.
func (s *Store) ClaimTask(ctx context.Context, runID, taskID string, opts ClaimOptions) (*Attempt, error) {
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if opts.Lease <= 0 {
		opts.Lease = 2 * time.Minute
	}
	if opts.WorkerID == "" {
		opts.WorkerID = "local-hub"
	}
	var claimed *Attempt
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		var runStatus, runMetadata string
		var maxConcurrency int
		if err := tx.QueryRowContext(ctx, `SELECT status,max_concurrency,metadata_json
FROM orchestration_runs WHERE id=?`, runID).Scan(&runStatus, &maxConcurrency, &runMetadata); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if RunStatus(runStatus) != RunRunning {
			return fmt.Errorf("%w: run is %s", ErrNotReady, runStatus)
		}
		retryBudget, _ := unmarshalObject(runMetadata)["retry_budget"].(float64)
		var taskStatus, queue, tagsJSON string
		var attemptCount, maxRetries int
		var nextAttempt int64
		if err := tx.QueryRowContext(ctx, `SELECT status,queue_name,capability_tags_json,
attempt_count,max_retries,next_attempt_at FROM orchestration_tasks WHERE run_id=? AND id=?`,
			runID, taskID).Scan(&taskStatus, &queue, &tagsJSON, &attemptCount, &maxRetries, &nextAttempt); err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
		if TaskStatus(taskStatus) != TaskPending && TaskStatus(taskStatus) != TaskRetrying {
			return fmt.Errorf("%w: task is %s", ErrNotReady, taskStatus)
		}
		if nextAttempt > 0 && nextAttempt > unixMillis(now) {
			return ErrNotReady
		}
		if retryBudget > 0 && attemptCount > 0 {
			var retriesUsed int
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(
CASE WHEN attempt_count>1 THEN attempt_count-1 ELSE 0 END),0)
FROM orchestration_tasks WHERE run_id=?`, runID).Scan(&retriesUsed); err != nil {
				return err
			}
			if retriesUsed >= int(retryBudget) {
				return fmt.Errorf("%w: run retry budget exhausted", ErrNotReady)
			}
		}
		if opts.Queue != "" && opts.Queue != queue {
			return ErrNotReady
		}
		var required []string
		_ = json.Unmarshal([]byte(tagsJSON), &required)
		if !hasCapabilities(opts.Capabilities, required) {
			return ErrNotReady
		}
		if attemptCount > maxRetries {
			return fmt.Errorf("%w: retry budget exhausted", ErrNotReady)
		}
		if maxConcurrency > 0 {
			var active int
			err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM orchestration_attempts
WHERE run_id=? AND status IN ('claimed','running') AND lease_expires_at>?`,
				runID, unixMillis(now)).Scan(&active)
			if err != nil {
				return err
			}
			if active >= maxConcurrency {
				return ErrConcurrencyFull
			}
		}
		number := attemptCount + 1
		claim := &Attempt{
			ID: uuid.NewString(), RunID: runID, TaskID: taskID, Number: number,
			Status: AttemptClaimed, WorkerID: opts.WorkerID, LeaseToken: uuid.NewString(),
			LeaseExpiresAt: now.Add(opts.Lease), HeartbeatAt: now, StartedAt: now,
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_attempts(
id,run_id,task_id,attempt_number,status,worker_id,lease_token,lease_expires_at,
heartbeat_at,started_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			claim.ID, runID, taskID, number, claim.Status, claim.WorkerID, claim.LeaseToken,
			unixMillis(claim.LeaseExpiresAt), unixMillis(now), unixMillis(now))
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,attempt_count=?,next_attempt_at=0,version=version+1,updated_at=?
WHERE run_id=? AND id=? AND status IN ('pending','retrying')`,
			TaskRunning, number, unixMillis(now), runID, taskID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return ErrConflict
		}
		if err := appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, AttemptID: claim.ID, Type: "attempt.claimed",
			CreatedAt: now, Payload: map[string]any{"number": number, "worker_id": claim.WorkerID},
		}); err != nil {
			return err
		}
		claimed = claim
		return nil
	})
	return claimed, err
}

func (s *Store) MarkAttemptRunning(ctx context.Context, attemptID, leaseToken string) error {
	return s.updateOwnedAttempt(ctx, attemptID, leaseToken, "claimed", AttemptRunning, "attempt.started", nil)
}

func (s *Store) Heartbeat(ctx context.Context, attemptID, leaseToken string, extend time.Duration, now time.Time) error {
	if extend <= 0 {
		extend = 2 * time.Minute
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return s.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts
SET heartbeat_at=?, lease_expires_at=?
WHERE id=? AND lease_token=? AND status IN ('claimed','running')`,
			unixMillis(now), unixMillis(now.Add(extend)), attemptID, leaseToken)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return ErrLeaseLost
		}
		return nil
	})
}

func (s *Store) CompleteAttempt(ctx context.Context, attemptID, leaseToken, resultID string) error {
	now := time.Now().UTC()
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var runID, taskID, status string
		if err := tx.QueryRowContext(ctx, `SELECT run_id,task_id,status FROM orchestration_attempts
WHERE id=? AND lease_token=?`, attemptID, leaseToken).Scan(&runID, &taskID, &status); err != nil {
			if err == sql.ErrNoRows {
				return ErrLeaseLost
			}
			return err
		}
		if status != string(AttemptClaimed) && status != string(AttemptRunning) {
			return ErrLeaseLost
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts SET
status=?,completed_at=?,result_id=? WHERE id=?`,
			AttemptSucceeded, unixMillis(now), resultID, attemptID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,version=version+1,updated_at=? WHERE run_id=? AND id=?`,
			TaskCompleted, unixMillis(now), runID, taskID); err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, AttemptID: attemptID, Type: "attempt.completed", CreatedAt: now,
		})
	})
}

func (s *Store) FailAttempt(ctx context.Context, attemptID, leaseToken string, cause error, policy RetryPolicy) (bool, time.Time, error) {
	now := time.Now().UTC()
	var retry bool
	var next time.Time
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		var runID, taskID, status string
		var number int
		if err := tx.QueryRowContext(ctx, `SELECT run_id,task_id,attempt_number,status
FROM orchestration_attempts WHERE id=? AND lease_token=?`, attemptID, leaseToken).
			Scan(&runID, &taskID, &number, &status); err != nil {
			if err == sql.ErrNoRows {
				return ErrLeaseLost
			}
			return err
		}
		if status != string(AttemptClaimed) && status != string(AttemptRunning) {
			return ErrLeaseLost
		}
		message := ""
		if cause != nil {
			message = cause.Error()
		}
		retry = policy.ShouldRetry(number, cause)
		taskStatus := TaskFailed
		if retry {
			taskStatus = TaskRetrying
			next = now.Add(policy.Delay(number, runID+":"+taskID))
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts SET
status=?,completed_at=?,error_text=? WHERE id=?`,
			AttemptFailed, unixMillis(now), message, attemptID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,next_attempt_at=?,version=version+1,updated_at=? WHERE run_id=? AND id=?`,
			taskStatus, unixMillis(next), unixMillis(now), runID, taskID); err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, AttemptID: attemptID, Type: "attempt.failed",
			CreatedAt: now, Payload: map[string]any{"error": message, "retry": retry, "next_attempt_at": next},
		})
	})
	return retry, next, err
}

func (s *Store) updateOwnedAttempt(
	ctx context.Context,
	attemptID, leaseToken, expected string,
	status AttemptStatus,
	eventType string,
	payload map[string]any,
) error {
	now := time.Now().UTC()
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var runID, taskID string
		err := tx.QueryRowContext(ctx, `SELECT run_id,task_id FROM orchestration_attempts
WHERE id=? AND lease_token=? AND status=?`, attemptID, leaseToken, expected).Scan(&runID, &taskID)
		if err == sql.ErrNoRows {
			return ErrLeaseLost
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts SET status=?,heartbeat_at=? WHERE id=?`,
			status, unixMillis(now), attemptID); err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, AttemptID: attemptID,
			Type: eventType, Payload: payload, CreatedAt: now,
		})
	})
}

// ReconcileExpiredLeases marks expired work lost and schedules a retry when
// budget remains. It returns the number of reconciled attempts.
func (s *Store) ReconcileExpiredLeases(ctx context.Context, now time.Time, policy RetryPolicy) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	count := 0
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT a.id,a.run_id,a.task_id,a.attempt_number,
t.max_retries,t.metadata_json,
CASE WHEN t.timeout_ms>0 AND a.started_at+t.timeout_ms<=? THEN 1 ELSE 0 END
FROM orchestration_attempts a
JOIN orchestration_tasks t ON t.run_id=a.run_id AND t.id=a.task_id
WHERE a.status IN ('claimed','running') AND (
 a.lease_expires_at<=? OR (t.timeout_ms>0 AND a.started_at+t.timeout_ms<=?))`,
			unixMillis(now), unixMillis(now), unixMillis(now))
		if err != nil {
			return err
		}
		type expired struct {
			id, runID, taskID string
			number            int
			maxRetries        int
			idempotency       bool
			timedOut          bool
			metadata          map[string]any
		}
		var items []expired
		for rows.Next() {
			var item expired
			var metadata string
			if err := rows.Scan(
				&item.id, &item.runID, &item.taskID, &item.number,
				&item.maxRetries, &metadata, &item.timedOut,
			); err != nil {
				_ = rows.Close()
				return err
			}
			item.metadata = unmarshalObject(metadata)
			item.idempotency, _ = item.metadata["idempotency_required"].(bool)
			items = append(items, item)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range items {
			itemPolicy := policy
			itemPolicy.MaxRetries = item.maxRetries
			category := "lease_lost"
			if item.timedOut {
				category = "timeout"
			}
			retry := !item.idempotency && RetryCategoryAllowed(item.metadata, category) &&
				itemPolicy.ShouldRetry(item.number, ErrLeaseLost)
			taskStatus := TaskFailed
			var next time.Time
			if retry {
				taskStatus = TaskRetrying
				next = now.Add(itemPolicy.Delay(item.number, item.runID+":"+item.taskID))
			}
			attemptStatus := AttemptLost
			errorText := "lease expired"
			eventType := "attempt.lease_expired"
			if item.timedOut {
				attemptStatus = AttemptTimedOut
				errorText = "attempt timeout exceeded"
				eventType = "attempt.timed_out"
			}
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_attempts SET
status=?,completed_at=?,error_text=? WHERE id=?`,
				attemptStatus, unixMillis(now), errorText, item.id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,next_attempt_at=?,version=version+1,updated_at=? WHERE run_id=? AND id=?`,
				taskStatus, unixMillis(next), unixMillis(now), item.runID, item.taskID); err != nil {
				return err
			}
			if err := appendEventTx(ctx, tx, Event{
				RunID: item.runID, TaskID: item.taskID, AttemptID: item.id,
				Type: eventType, CreatedAt: now,
				Payload: map[string]any{
					"retry": retry, "next_attempt_at": next,
					"manual_reconciliation_required": item.idempotency,
				},
			}); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

func (s *Store) PutResult(ctx context.Context, result Result) (*Result, error) {
	if result.ID == "" {
		result.ID = uuid.NewString()
	}
	if result.ContentType == "" {
		result.ContentType = "application/json"
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	metadata, err := marshalObject(result.Metadata)
	if err != nil {
		return nil, err
	}
	err = s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_results(
id,run_id,task_id,attempt_id,execution_key,value_blob,content_type,metadata_json,expires_at,created_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
			result.ID, result.RunID, result.TaskID, result.AttemptID, result.ExecutionKey,
			result.Value, result.ContentType, metadata, unixMillis(result.ExpiresAt), unixMillis(result.CreatedAt))
		if err != nil {
			return err
		}
		return appendEventTx(ctx, tx, Event{
			RunID: result.RunID, TaskID: result.TaskID, AttemptID: result.AttemptID,
			Type: "result.persisted", CreatedAt: result.CreatedAt,
			Payload: map[string]any{"result_id": result.ID, "execution_key": result.ExecutionKey},
		})
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) GetCachedResult(ctx context.Context, executionKey string, now time.Time) (*Result, error) {
	if executionKey == "" {
		return nil, ErrNotFound
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var result Result
	var metadata string
	var expires, created int64
	err := s.db.QueryRowContext(ctx, `SELECT id,run_id,task_id,attempt_id,execution_key,
value_blob,content_type,metadata_json,expires_at,created_at
FROM orchestration_results
WHERE execution_key=? AND (expires_at=0 OR expires_at>?)
ORDER BY created_at DESC LIMIT 1`, executionKey, unixMillis(now)).Scan(
		&result.ID, &result.RunID, &result.TaskID, &result.AttemptID, &result.ExecutionKey,
		&result.Value, &result.ContentType, &metadata, &expires, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	result.Metadata = unmarshalObject(metadata)
	result.ExpiresAt, result.CreatedAt = fromUnixMillis(expires), fromUnixMillis(created)
	return &result, nil
}

func (s *Store) DeleteExpiredResults(ctx context.Context, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	count := 0
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM orchestration_results
WHERE expires_at>0 AND expires_at<=?`, unixMillis(now))
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		count = int(n)
		return nil
	})
	return count, err
}

func (s *Store) CreateInput(ctx context.Context, input InputRequest) (*InputRequest, error) {
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Kind == "" {
		input.Kind = "approval"
	}
	if len(input.Schema) == 0 {
		input.Schema = json.RawMessage(`{}`)
	}
	if len(input.InitialValue) == 0 {
		input.InitialValue = json.RawMessage(`null`)
	}
	if input.Status == "" {
		input.Status = InputPending
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO orchestration_inputs(
id,run_id,task_id,kind,schema_json,initial_json,decision_key,requester,approver,
status,response_json,expires_at,created_at,resolved_at)
VALUES(?,?,?,?,?,?,?,?,?,?, 'null',?,?,0)`,
			input.ID, input.RunID, input.TaskID, input.Kind, string(input.Schema), string(input.InitialValue),
			input.DecisionKey, input.Requester, input.Approver, input.Status,
			unixMillis(input.ExpiresAt), unixMillis(input.CreatedAt))
		if err != nil {
			return err
		}
		if input.TaskID != "" && input.RunID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,version=version+1,updated_at=? WHERE run_id=? AND id=?`,
				TaskAwaitingInput, unixMillis(input.CreatedAt), input.RunID, input.TaskID); err != nil {
				return err
			}
		}
		if input.RunID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_runs SET
status=?,version=version+1,updated_at=? WHERE id=? AND status=?`,
				RunSuspended, unixMillis(input.CreatedAt), input.RunID, RunRunning); err != nil {
				return err
			}
		}
		return appendEventTx(ctx, tx, Event{
			RunID: input.RunID, TaskID: input.TaskID, Type: "input.requested",
			CreatedAt: input.CreatedAt, Payload: map[string]any{"input_id": input.ID, "kind": input.Kind},
		})
	})
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *Store) ResolveInput(ctx context.Context, id, approver string, response json.RawMessage) (*InputRequest, error) {
	if len(response) == 0 || !json.Valid(response) {
		return nil, errors.New("response must be valid JSON")
	}
	var resolved InputRequest
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		var status, schema, initial, storedResponse string
		var expires, created, resolvedAt int64
		err := tx.QueryRowContext(ctx, `SELECT id,run_id,task_id,kind,schema_json,initial_json,
decision_key,requester,approver,status,response_json,expires_at,created_at,resolved_at
FROM orchestration_inputs WHERE id=?`, id).Scan(
			&resolved.ID, &resolved.RunID, &resolved.TaskID, &resolved.Kind, &schema, &initial,
			&resolved.DecisionKey, &resolved.Requester, &resolved.Approver, &status, &storedResponse,
			&expires, &created, &resolvedAt)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if InputStatus(status) != InputPending {
			return ErrAlreadyResolved
		}
		now := time.Now().UTC()
		if expires > 0 && expires <= unixMillis(now) {
			return ErrAlreadyResolved
		}
		if resolved.Approver != "" && approver != "" && resolved.Approver != approver {
			return fmt.Errorf("%w: approver mismatch", ErrConflict)
		}
		if err := validateInputResponse(json.RawMessage(schema), response); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_inputs SET
status=?,approver=?,response_json=?,resolved_at=? WHERE id=? AND status=?`,
			InputAnswered, approver, string(response), unixMillis(now), id, InputPending); err != nil {
			return err
		}
		if resolved.RunID != "" && resolved.TaskID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,version=version+1,updated_at=? WHERE run_id=? AND id=? AND status=?`,
				TaskPending, unixMillis(now), resolved.RunID, resolved.TaskID, TaskAwaitingInput); err != nil {
				return err
			}
		}
		if resolved.RunID != "" {
			if err := resumeRunAfterInputsTx(ctx, tx, resolved.RunID, now); err != nil {
				return err
			}
		}
		if err := appendEventTx(ctx, tx, Event{
			RunID: resolved.RunID, TaskID: resolved.TaskID, Type: "input.answered",
			CreatedAt: now, Payload: map[string]any{"input_id": id, "approver": approver},
		}); err != nil {
			return err
		}
		resolved.Status = InputAnswered
		resolved.Schema = json.RawMessage(schema)
		resolved.InitialValue = json.RawMessage(initial)
		resolved.Response = append(json.RawMessage(nil), response...)
		resolved.ExpiresAt, resolved.CreatedAt, resolved.ResolvedAt =
			fromUnixMillis(expires), fromUnixMillis(created), now
		resolved.Approver = approver
		return nil
	})
	return &resolved, err
}

func (s *Store) ExpireInput(ctx context.Context, id, reason string) error {
	now := time.Now().UTC()
	response, _ := json.Marshal(map[string]any{"reason": reason})
	return s.transaction(ctx, func(tx *sql.Tx) error {
		var runID, taskID string
		var status string
		err := tx.QueryRowContext(ctx, `SELECT run_id,task_id,status FROM orchestration_inputs WHERE id=?`, id).
			Scan(&runID, &taskID, &status)
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if InputStatus(status) != InputPending {
			return ErrAlreadyResolved
		}
		if _, err := tx.ExecContext(ctx, `UPDATE orchestration_inputs SET
status=?,response_json=?,resolved_at=? WHERE id=? AND status=?`,
			InputExpired, string(response), unixMillis(now), id, InputPending); err != nil {
			return err
		}
		if runID != "" && taskID != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE orchestration_tasks SET
status=?,version=version+1,updated_at=? WHERE run_id=? AND id=? AND status=?`,
				TaskFailed, unixMillis(now), runID, taskID, TaskAwaitingInput); err != nil {
				return err
			}
		}
		if runID != "" {
			if err := resumeRunAfterInputsTx(ctx, tx, runID, now); err != nil {
				return err
			}
		}
		return appendEventTx(ctx, tx, Event{
			RunID: runID, TaskID: taskID, Type: "input.expired", CreatedAt: now,
			Payload: map[string]any{"input_id": id, "reason": reason},
		})
	})
}

func (s *Store) ListPendingInputs(ctx context.Context) ([]InputRequest, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,run_id,task_id,kind,schema_json,initial_json,
decision_key,requester,approver,status,response_json,expires_at,created_at,resolved_at
FROM orchestration_inputs WHERE status=? ORDER BY created_at`, InputPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InputRequest
	for rows.Next() {
		var item InputRequest
		var status, schema, initial, response string
		var expires, created, resolved int64
		if err := rows.Scan(&item.ID, &item.RunID, &item.TaskID, &item.Kind, &schema, &initial,
			&item.DecisionKey, &item.Requester, &item.Approver, &status, &response,
			&expires, &created, &resolved); err != nil {
			return nil, err
		}
		item.Status = InputStatus(status)
		item.Schema, item.InitialValue, item.Response =
			json.RawMessage(schema), json.RawMessage(initial), json.RawMessage(response)
		item.ExpiresAt, item.CreatedAt, item.ResolvedAt =
			fromUnixMillis(expires), fromUnixMillis(created), fromUnixMillis(resolved)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ExpireDueInputs(ctx context.Context, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM orchestration_inputs
WHERE status=? AND expires_at>0 AND expires_at<=?`, InputPending, unixMillis(now))
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := s.ExpireInput(ctx, id, "input deadline exceeded"); err != nil &&
			!errors.Is(err, ErrAlreadyResolved) {
			return 0, err
		}
	}
	return len(ids), nil
}

func (s *Store) AppendEvent(ctx context.Context, event Event) error {
	return s.transaction(ctx, func(tx *sql.Tx) error {
		return appendEventTx(ctx, tx, event)
	})
}

func (s *Store) AppendEventOnce(ctx context.Context, event Event) (bool, error) {
	appended := false
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
SELECT 1 FROM orchestration_events WHERE run_id=? AND task_id=? AND event_type=?)`,
			event.RunID, event.TaskID, event.Type).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return nil
		}
		if err := appendEventTx(ctx, tx, event); err != nil {
			return err
		}
		appended = true
		return nil
	})
	return appended, err
}

func (s *Store) ListEvents(ctx context.Context, runID string, afterID int64, limit int) ([]Event, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,run_id,task_id,attempt_id,event_type,payload_json,created_at
FROM orchestration_events WHERE run_id=? AND id>? ORDER BY id LIMIT ?`, runID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		var payload string
		var created int64
		if err := rows.Scan(&event.ID, &event.RunID, &event.TaskID, &event.AttemptID,
			&event.Type, &payload, &created); err != nil {
			return nil, err
		}
		event.Payload = unmarshalObject(payload)
		event.CreatedAt = fromUnixMillis(created)
		events = append(events, event)
	}
	return events, rows.Err()
}

func appendEventTx(ctx context.Context, tx *sql.Tx, event Event) error {
	if event.Type == "" {
		return errors.New("event type is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	payload, err := marshalObject(event.Payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO orchestration_events(
run_id,task_id,attempt_id,event_type,payload_json,created_at) VALUES(?,?,?,?,?,?)`,
		event.RunID, event.TaskID, event.AttemptID, event.Type, payload, unixMillis(event.CreatedAt))
	return err
}

func marshalObject(value map[string]any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func unmarshalObject(raw string) map[string]any {
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil || value == nil {
		return map[string]any{}
	}
	return value
}

func unixMillis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixMilli()
}

func fromUnixMillis(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func containsRunStatus(statuses []RunStatus, status RunStatus) bool {
	for _, candidate := range statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hasCapabilities(available, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(available))
	for _, item := range available {
		set[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := set[item]; !ok {
			return false
		}
	}
	return true
}

func validateInputResponse(schemaRaw, response json.RawMessage) error {
	if len(schemaRaw) == 0 || string(schemaRaw) == "{}" {
		return nil
	}
	var definition map[string]any
	if err := json.Unmarshal(schemaRaw, &definition); err != nil {
		return fmt.Errorf("invalid stored input schema: %w", err)
	}
	var value any
	if err := json.Unmarshal(response, &value); err != nil {
		return fmt.Errorf("invalid input response: %w", err)
	}
	if expected, _ := definition["type"].(string); expected == "object" {
		object, ok := value.(map[string]any)
		if !ok {
			return errors.New("input response must be an object")
		}
		if required, ok := definition["required"].([]any); ok {
			for _, rawKey := range required {
				key, _ := rawKey.(string)
				field, exists := object[key]
				if !exists || field == nil {
					return fmt.Errorf("input response missing required field %q", key)
				}
				if text, ok := field.(string); ok && strings.TrimSpace(text) == "" {
					return fmt.Errorf("input response field %q cannot be empty", key)
				}
			}
		}
		properties, _ := definition["properties"].(map[string]any)
		for key, rawRule := range properties {
			field, exists := object[key]
			if !exists {
				continue
			}
			rule, _ := rawRule.(map[string]any)
			if expected, _ := rule["type"].(string); expected == "string" {
				text, ok := field.(string)
				if !ok {
					return fmt.Errorf("input response field %q must be a string", key)
				}
				if min, ok := rule["minLength"].(float64); ok && len(text) < int(min) {
					return fmt.Errorf("input response field %q is shorter than %d", key, int(min))
				}
			}
			if allowed, ok := rule["enum"].([]any); ok {
				match := false
				for _, candidate := range allowed {
					if fmt.Sprint(field) == fmt.Sprint(candidate) {
						match = true
						break
					}
				}
				if !match {
					return fmt.Errorf("input response field %q is not an allowed value", key)
				}
			}
		}
	}
	return nil
}

func resumeRunAfterInputsTx(ctx context.Context, tx *sql.Tx, runID string, now time.Time) error {
	var pending int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM orchestration_inputs
WHERE run_id=? AND status=?`, runID, InputPending).Scan(&pending); err != nil {
		return err
	}
	if pending == 0 {
		_, err := tx.ExecContext(ctx, `UPDATE orchestration_runs SET
status=?,version=version+1,updated_at=? WHERE id=? AND status=?`,
			RunRunning, unixMillis(now), runID, RunSuspended)
		return err
	}
	return nil
}

func RetryCategoryAllowed(metadata map[string]any, category string) bool {
	raw, exists := metadata["retry_on"]
	if !exists {
		return true
	}
	switch values := raw.(type) {
	case []any:
		if len(values) == 0 {
			return true
		}
		for _, value := range values {
			if text, ok := value.(string); ok && text == category {
				return true
			}
		}
	case []string:
		if len(values) == 0 {
			return true
		}
		for _, value := range values {
			if value == category {
				return true
			}
		}
	}
	return false
}
