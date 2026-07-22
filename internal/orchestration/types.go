// Package orchestration provides the durable execution substrate used by
// collaboration runs. It intentionally contains no agent or hub dependencies.
package orchestration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunSuspended RunStatus = "suspended"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

type TaskStatus string

const (
	TaskPending       TaskStatus = "pending"
	TaskRunning       TaskStatus = "running"
	TaskRetrying      TaskStatus = "retrying"
	TaskAwaitingInput TaskStatus = "awaiting_input"
	TaskCompleted     TaskStatus = "completed"
	TaskFailed        TaskStatus = "failed"
	TaskCancelled     TaskStatus = "cancelled"
	TaskSkipped       TaskStatus = "skipped"
)

type AttemptStatus string

const (
	AttemptClaimed   AttemptStatus = "claimed"
	AttemptRunning   AttemptStatus = "running"
	AttemptSucceeded AttemptStatus = "succeeded"
	AttemptFailed    AttemptStatus = "failed"
	AttemptTimedOut  AttemptStatus = "timed_out"
	AttemptLost      AttemptStatus = "lost"
	AttemptCancelled AttemptStatus = "cancelled"
)

type InputStatus string

const (
	InputPending   InputStatus = "pending"
	InputAnswered  InputStatus = "answered"
	InputExpired   InputStatus = "expired"
	InputCancelled InputStatus = "cancelled"
)

type Run struct {
	ID                string         `json:"id"`
	DefinitionID      string         `json:"definition_id,omitempty"`
	DefinitionVersion int            `json:"definition_version,omitempty"`
	Status            RunStatus      `json:"status"`
	MaxConcurrency    int            `json:"max_concurrency,omitempty"`
	Version           int64          `json:"version"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type Task struct {
	ID             string         `json:"id"`
	RunID          string         `json:"run_id"`
	Title          string         `json:"title,omitempty"`
	Status         TaskStatus     `json:"status"`
	Queue          string         `json:"queue"`
	CapabilityTags []string       `json:"capability_tags,omitempty"`
	MaxRetries     int            `json:"max_retries,omitempty"`
	Timeout        time.Duration  `json:"timeout,omitempty"`
	AttemptCount   int            `json:"attempt_count"`
	NextAttemptAt  time.Time      `json:"next_attempt_at,omitempty"`
	ExecutionKey   string         `json:"execution_key,omitempty"`
	CachePolicy    CachePolicy    `json:"cache_policy"`
	Version        int64          `json:"version"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Attempt struct {
	ID             string        `json:"id"`
	RunID          string        `json:"run_id"`
	TaskID         string        `json:"task_id"`
	Number         int           `json:"number"`
	Status         AttemptStatus `json:"status"`
	WorkerID       string        `json:"worker_id"`
	LeaseToken     string        `json:"-"`
	LeaseExpiresAt time.Time     `json:"lease_expires_at"`
	HeartbeatAt    time.Time     `json:"heartbeat_at"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    time.Time     `json:"completed_at,omitempty"`
	Error          string        `json:"error,omitempty"`
	ResultID       string        `json:"result_id,omitempty"`
}

type InputRequest struct {
	ID           string          `json:"id"`
	RunID        string          `json:"run_id,omitempty"`
	TaskID       string          `json:"task_id,omitempty"`
	Kind         string          `json:"kind"`
	Schema       json.RawMessage `json:"schema,omitempty"`
	InitialValue json.RawMessage `json:"initial_value,omitempty"`
	DecisionKey  string          `json:"decision_key,omitempty"`
	Requester    string          `json:"requester,omitempty"`
	Approver     string          `json:"approver,omitempty"`
	Status       InputStatus     `json:"status"`
	Response     json.RawMessage `json:"response,omitempty"`
	ExpiresAt    time.Time       `json:"expires_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	ResolvedAt   time.Time       `json:"resolved_at,omitempty"`
}

type Result struct {
	ID           string         `json:"id"`
	RunID        string         `json:"run_id,omitempty"`
	TaskID       string         `json:"task_id,omitempty"`
	AttemptID    string         `json:"attempt_id,omitempty"`
	ExecutionKey string         `json:"execution_key,omitempty"`
	Value        []byte         `json:"value,omitempty"`
	ContentType  string         `json:"content_type"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Event struct {
	ID        int64          `json:"id"`
	RunID     string         `json:"run_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	AttemptID string         `json:"attempt_id,omitempty"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type RetryPolicy struct {
	MaxRetries   int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	JitterFactor float64
	RetryIf      func(error) bool
}

func (p RetryPolicy) Normalized() RetryPolicy {
	if p.MaxRetries < 0 {
		p.MaxRetries = 0
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = time.Second
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = time.Minute
	}
	if p.MaxDelay < p.BaseDelay {
		p.MaxDelay = p.BaseDelay
	}
	if p.JitterFactor < 0 {
		p.JitterFactor = 0
	}
	if p.JitterFactor > 1 {
		p.JitterFactor = 1
	}
	return p
}

func (p RetryPolicy) ShouldRetry(attemptNumber int, err error) bool {
	p = p.Normalized()
	if attemptNumber > p.MaxRetries {
		return false
	}
	return p.RetryIf == nil || p.RetryIf(err)
}

// Delay returns deterministic exponential backoff with bounded jitter. Keeping
// jitter deterministic makes restart and fault-injection tests reproducible.
func (p RetryPolicy) Delay(attemptNumber int, seed string) time.Duration {
	p = p.Normalized()
	if attemptNumber < 1 {
		attemptNumber = 1
	}
	exponent := math.Pow(2, float64(attemptNumber-1))
	delay := time.Duration(float64(p.BaseDelay) * exponent)
	if delay > p.MaxDelay || delay < 0 {
		delay = p.MaxDelay
	}
	if p.JitterFactor == 0 {
		return delay
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", seed, attemptNumber)))
	unit := float64(uint16(sum[0])<<8|uint16(sum[1])) / 65535
	factor := 1 + ((unit*2)-1)*p.JitterFactor
	jittered := time.Duration(float64(delay) * factor)
	if jittered < 0 {
		return 0
	}
	if jittered > p.MaxDelay {
		return p.MaxDelay
	}
	return jittered
}

type CachePolicy struct {
	Enabled    bool          `json:"enabled"`
	Expiration time.Duration `json:"expiration,omitempty"`
	Refresh    bool          `json:"refresh,omitempty"`
}

type ExecutionKeyInput struct {
	DefinitionID      string         `json:"definition_id"`
	DefinitionVersion int            `json:"definition_version"`
	TaskID            string         `json:"task_id"`
	Inputs            map[string]any `json:"inputs,omitempty"`
	ContextHash       string         `json:"context_hash,omitempty"`
	Provider          string         `json:"provider,omitempty"`
	Model             string         `json:"model,omitempty"`
	PolicyVersion     string         `json:"policy_version,omitempty"`
}

func ExecutionKey(input ExecutionKeyInput) (string, error) {
	if input.DefinitionID == "" || input.TaskID == "" {
		return "", errors.New("definition_id and task_id are required")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal execution key: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

type ClaimOptions struct {
	WorkerID     string
	Lease        time.Duration
	Queue        string
	Capabilities []string
	Now          time.Time
}

var (
	ErrNotFound        = errors.New("orchestration record not found")
	ErrConflict        = errors.New("orchestration state conflict")
	ErrConcurrencyFull = errors.New("run concurrency limit reached")
	ErrNotReady        = errors.New("task is not ready")
	ErrLeaseLost       = errors.New("attempt lease is no longer owned")
	ErrAlreadyResolved = errors.New("input already resolved")
)
