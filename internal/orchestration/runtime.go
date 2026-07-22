package orchestration

import (
	"context"
	"encoding/json"
	"time"
)

type WorkerStatus string

const (
	WorkerReady   WorkerStatus = "ready"
	WorkerPaused  WorkerStatus = "paused"
	WorkerOffline WorkerStatus = "offline"
)

type Worker struct {
	ID            string         `json:"id"`
	Queue         string         `json:"queue"`
	Capabilities  []string       `json:"capabilities,omitempty"`
	Status        WorkerStatus   `json:"status"`
	LastHeartbeat time.Time      `json:"last_heartbeat"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Deployment struct {
	ID                string
	DefinitionID      string
	DefinitionVersion int
	Queue             string
	Schedule          string
	EventFilter       map[string]any
	Enabled           bool
	Parameters        map[string]any
	NextRunAt         time.Time
	LastTriggeredAt   time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Automation struct {
	ID         string
	Name       string
	EventType  string
	Posture    string
	Threshold  int
	Within     time.Duration
	ActionType string
	Action     map[string]any
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type WorkRequest struct {
	RunID          string
	TaskID         string
	AttemptID      string
	AttemptNumber  int
	LeaseToken     string
	ExecutionKey   string
	Timeout        time.Duration
	Payload        json.RawMessage
	CapabilityTags []string
}

type Progress struct {
	Message    string
	Percentage float64
	Metadata   map[string]any
}

type ProgressReporter func(context.Context, Progress) error

// Runner is the stable execution boundary for local agents, sidecars, and
// future remote workers.
type Runner interface {
	ID() string
	Capabilities() []string
	Execute(context.Context, WorkRequest, ProgressReporter) ([]byte, string, map[string]any, error)
	Cancel(context.Context, string) error
}

type RunnerFunc struct {
	RunnerID  string
	Tags      []string
	ExecuteFn func(context.Context, WorkRequest, ProgressReporter) ([]byte, string, map[string]any, error)
	CancelFn  func(context.Context, string) error
}

func (r RunnerFunc) ID() string { return r.RunnerID }

func (r RunnerFunc) Capabilities() []string {
	return append([]string(nil), r.Tags...)
}

func (r RunnerFunc) Execute(ctx context.Context, request WorkRequest, report ProgressReporter) ([]byte, string, map[string]any, error) {
	return r.ExecuteFn(ctx, request, report)
}

func (r RunnerFunc) Cancel(ctx context.Context, attemptID string) error {
	if r.CancelFn == nil {
		return nil
	}
	return r.CancelFn(ctx, attemptID)
}

type AutomationActionHandler interface {
	HandleAutomation(context.Context, Automation, Event) error
}

type DeploymentLauncher interface {
	LaunchDeployment(context.Context, Deployment, map[string]any) (string, error)
}
