package secondaryanalysis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const maxLogLines = 200

// BiologySettings supplies paths for Python subprocess jobs.
type BiologySettings struct {
	ToolsPath        string
	PythonExecutable string
	CumulativeQCDir  string
}

// Manager runs secondary analysis jobs (Python subprocess or future native workflows).
type Manager struct {
	mu       sync.Mutex
	jobs     map[string]*runningJob
	rootDir  string
	settings func() BiologySettings
}

type runningJob struct {
	Job
	cancel context.CancelFunc
	cmd    *exec.Cmd
}

// NewManager creates a job manager. rootDir stores job metadata under ~/.neural-junkie/secondary-analysis.
func NewManager(rootDir string, settings func() BiologySettings) *Manager {
	if rootDir == "" {
		home, _ := os.UserHomeDir()
		rootDir = filepath.Join(home, ".neural-junkie", "secondary-analysis")
	}
	_ = os.MkdirAll(rootDir, 0o755)
	return &Manager{
		jobs:     make(map[string]*runningJob),
		rootDir:  rootDir,
		settings: settings,
	}
}

// Start enqueues and runs a job asynchronously.
func (m *Manager) Start(ctx context.Context, req StartRequest) (*Job, error) {
	if strings.TrimSpace(string(req.Workflow)) == "" {
		return nil, fmt.Errorf("workflow is required")
	}
	m.mu.Lock()
	for _, rj := range m.jobs {
		if rj.Status == StatusQueued || rj.Status == StatusRunning {
			m.mu.Unlock()
			return nil, fmt.Errorf("another secondary analysis job is already running")
		}
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	id := uuid.NewString()
	rj := &runningJob{
		Job: Job{
			ID:          id,
			Workflow:    req.Workflow,
			Status:      StatusQueued,
			WorkspaceID: strings.TrimSpace(req.WorkspaceID),
			Config:      req.Config,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		cancel: cancel,
	}
	m.jobs[id] = rj
	m.mu.Unlock()

	go m.run(jobCtx, rj)
	return copyJob(&rj.Job), nil
}

func (m *Manager) run(ctx context.Context, rj *runningJob) {
	defer rj.cancel()
	m.setStatus(rj, StatusRunning, "")

	settings := m.settings()
	toolsPath := strings.TrimSpace(settings.ToolsPath)
	if toolsPath == "" {
		m.fail(rj, "secondary_analysis_tools_path is not configured in Settings → Life sciences tools")
		return
	}

	script, err := scriptForWorkflow(rj.Workflow)
	if err != nil {
		m.fail(rj, err.Error())
		return
	}
	scriptPath := filepath.Join(toolsPath, script)
	if _, err := os.Stat(scriptPath); err != nil {
		m.fail(rj, fmt.Sprintf("script not found: %s (set secondary_analysis_tools_path in Settings)", scriptPath))
		return
	}

	cfgDir := filepath.Join(m.rootDir, rj.ID)
	_ = os.MkdirAll(cfgDir, 0o755)
	cfgPath := filepath.Join(cfgDir, "config.json")
	if rj.Config == nil {
		rj.Config = map[string]any{}
	}
	if wr, ok := rj.Config["workspace_root"].(string); ok && strings.TrimSpace(wr) != "" {
		outDir := filepath.Join(wr, ".neural-junkie", "analysis-runs", rj.ID)
		if existing, ok := rj.Config["out_dir"].(string); !ok || strings.TrimSpace(existing) == "" ||
			strings.Contains(existing, ".neural-junkie/analysis-runs/") {
			rj.Config["out_dir"] = outDir
		}
	}
	cfgBytes, _ := json.MarshalIndent(rj.Config, "", "  ")
	if err := os.WriteFile(cfgPath, cfgBytes, 0o644); err != nil {
		m.fail(rj, "write config: "+err.Error())
		return
	}

	python := settings.PythonExecutableOrDefault()
	cmd := exec.CommandContext(ctx, python, scriptPath, "--config", cfgPath)
	cmd.Dir = toolsPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.fail(rj, err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.fail(rj, err.Error())
		return
	}

	m.mu.Lock()
	rj.cmd = cmd
	m.mu.Unlock()

	if err := cmd.Start(); err != nil {
		m.fail(rj, err.Error())
		return
	}

	go m.drainLogs(rj, stdout)
	go m.drainLogs(rj, stderr)

	err = cmd.Wait()
	m.mu.Lock()
	rj.cmd = nil
	m.mu.Unlock()

	if ctx.Err() == context.Canceled {
		m.setStatus(rj, StatusCancelled, "cancelled")
		return
	}
	if err != nil {
		m.fail(rj, err.Error())
		return
	}
	if out, ok := rj.Config["out_dir"].(string); ok {
		rj.OutputDir = out
	}
	m.setStatus(rj, StatusDone, "")
}

func (settings BiologySettings) PythonExecutableOrDefault() string {
	if p := strings.TrimSpace(settings.PythonExecutable); p != "" {
		return p
	}
	return "python3"
}

func scriptForWorkflow(w Workflow) (string, error) {
	switch w {
	case WorkflowComparator:
		return filepath.Join("cli", "run_comparator.py"), nil
	case WorkflowEndogenous:
		return filepath.Join("cli", "run_endogenous.py"), nil
	case WorkflowStdCurves:
		return filepath.Join("cli", "run_std_curves.py"), nil
	case WorkflowPrintOrder:
		return filepath.Join("cli", "run_print_order.py"), nil
	case Workflow12PlexQCExcel:
		return filepath.Join("cli", "run_12plex_qc.py"), nil
	case WorkflowSPCCharts:
		return filepath.Join("cli", "run_spc_charts.py"), nil
	default:
		return "", fmt.Errorf("unknown workflow %q", w)
	}
}

func (m *Manager) drainLogs(rj *runningJob, r io.Reader) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		m.appendLog(rj, sc.Text())
	}
}

func (m *Manager) appendLog(rj *runningJob, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rj.LogTail = append(rj.LogTail, line)
	if len(rj.LogTail) > maxLogLines {
		rj.LogTail = rj.LogTail[len(rj.LogTail)-maxLogLines:]
	}
	rj.UpdatedAt = time.Now()
}

func (m *Manager) setStatus(rj *runningJob, st Status, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rj.Status = st
	rj.Error = errMsg
	rj.UpdatedAt = time.Now()
}

func (m *Manager) fail(rj *runningJob, msg string) {
	m.setStatus(rj, StatusFailed, msg)
	m.appendLog(rj, "ERROR: "+msg)
}

// Get returns a job copy by id.
func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rj, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	return copyJob(&rj.Job), true
}

// Cancel stops a running job.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	rj, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("job not found")
	}
	if rj.Status != StatusQueued && rj.Status != StatusRunning {
		m.mu.Unlock()
		return fmt.Errorf("job is not running")
	}
	if rj.cancel != nil {
		rj.cancel()
	}
	if rj.cmd != nil && rj.cmd.Process != nil {
		_ = rj.cmd.Process.Kill()
	}
	m.mu.Unlock()
	return nil
}

func copyJob(j *Job) *Job {
	out := *j
	if len(j.LogTail) > 0 {
		out.LogTail = append([]string(nil), j.LogTail...)
	}
	if j.Config != nil {
		out.Config = make(map[string]any, len(j.Config))
		for k, v := range j.Config {
			out.Config[k] = v
		}
	}
	return &out
}
