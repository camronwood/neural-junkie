package secondaryanalysis

import "time"

// Status is a secondary analysis job lifecycle state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusDone      Status = "done"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Workflow identifies which secondary analysis pipeline to run.
type Workflow string

const (
	WorkflowComparator    Workflow = "comparator"
	WorkflowEndogenous    Workflow = "endogenous"
	WorkflowStdCurves     Workflow = "std_curves"
	WorkflowPrintOrder    Workflow = "print_order"
	Workflow12PlexQCExcel Workflow = "12plex_qc_excel"
	WorkflowSPCCharts     Workflow = "spc_charts"
)

// StartRequest starts an async secondary analysis job.
type StartRequest struct {
	Workflow    Workflow       `json:"workflow"`
	WorkspaceID string         `json:"workspace_id"`
	Config      map[string]any `json:"config"`
}

// Job is the persisted view of one secondary analysis run.
type Job struct {
	ID          string         `json:"id"`
	Workflow    Workflow       `json:"workflow"`
	Status      Status         `json:"status"`
	WorkspaceID string         `json:"workspace_id,omitempty"`
	OutputDir   string         `json:"output_dir,omitempty"`
	LogTail     []string       `json:"log_tail,omitempty"`
	Error       string         `json:"error,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
