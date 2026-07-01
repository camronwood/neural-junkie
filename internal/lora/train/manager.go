package train

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

	"github.com/camronwood/neural-junkie/internal/collaboration"
	"github.com/camronwood/neural-junkie/internal/hfhub"
	lorexport "github.com/camronwood/neural-junkie/internal/lora/export"
	"github.com/camronwood/neural-junkie/internal/lora/registry"
)

// Status is a training job lifecycle state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusExporting Status = "exporting"
	StatusTraining  Status = "training"
	StatusComposing Status = "composing"
	StatusEvaluating Status = "evaluating"
	StatusDone      Status = "done"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// HyperParams are optional LoRA training knobs.
type HyperParams struct {
	Rank         int     `json:"rank,omitempty"`
	Epochs       int     `json:"epochs,omitempty"`
	LearningRate float64 `json:"learning_rate,omitempty"`
	MaxSeqLen    int     `json:"max_seq_len,omitempty"`
}

// StartRequest starts a new training job.
type StartRequest struct {
	Source             lorexport.SourceKind `json:"source"`
	SourceID           string               `json:"source_id"`
	ThreadID           string               `json:"thread_id,omitempty"`
	AgentName          string               `json:"agent_name,omitempty"`
	AgentID            string               `json:"agent_id,omitempty"`
	IncludeLearnings   bool                 `json:"include_learnings,omitempty"`
	LearningRows       []lorexport.Row      `json:"-"`
	BaseOllamaTag      string               `json:"base_ollama_tag"`
	OllamaTag          string               `json:"ollama_tag"`
	HyperParams        HyperParams          `json:"hyperparams,omitempty"`
	Incremental        bool                 `json:"incremental,omitempty"`
	PriorAdapterID     string               `json:"prior_adapter_id,omitempty"`
	SinceJobID         string               `json:"since_job_id,omitempty"`
	RowIDs             []string             `json:"row_ids,omitempty"`
	ApprovedTasksOnly  bool                 `json:"approved_tasks_only,omitempty"`
	Backend            string               `json:"backend,omitempty"` // unsloth | mlx | auto
	SkipEval           bool                 `json:"skip_eval,omitempty"`
}

// Job is the persisted view of one training run.
type Job struct {
	ID            string               `json:"id"`
	Status        Status               `json:"status"`
	Source        lorexport.SourceKind `json:"source"`
	SourceID      string               `json:"source_id"`
	AgentID       string               `json:"agent_id,omitempty"`
	BaseOllamaTag string               `json:"base_ollama_tag"`
	OllamaTag     string               `json:"ollama_tag"`
	RowCount      int                  `json:"row_count,omitempty"`
	QueuePosition int                  `json:"queue_position,omitempty"`
	AdapterID     string               `json:"adapter_id,omitempty"`
	EvalScore     float64              `json:"eval_score,omitempty"`
	LogTail       []string             `json:"log_tail,omitempty"`
	Error         string               `json:"error,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// CollabLookup loads collaboration snapshots.
type CollabLookup interface {
	GetCollaborationSnapshot(id string) (*collaboration.Collaboration, error)
}

// OnCompleteFunc is called after successful compose (registry, eval, etc.).
type OnCompleteFunc func(ctx context.Context, job *Job, artifactDir, datasetPath string) error

// Manager runs LoRA training jobs with optional queue.
type Manager struct {
	mu         sync.Mutex
	active     *runningJob
	queue      []*queuedJob
	jobs       map[string]*Job
	rootDir    string
	jobsPath   string
	script     string
	mlxScript  string
	python     string
	msgs       lorexport.MessageSource
	collab     CollabLookup
	registry   *registry.Store
	onComplete OnCompleteFunc
}

type runningJob struct {
	Job
	cancel context.CancelFunc
	cmd    *exec.Cmd
	req    StartRequest
}

type queuedJob struct {
	req StartRequest
	id  string
}

type jobsFile struct {
	Jobs []Job `json:"jobs"`
}

// NewManager creates a job manager.
func NewManager(rootDir, scriptPath, pythonPath string, msgs lorexport.MessageSource, collab CollabLookup) *Manager {
	if rootDir == "" {
		home, _ := os.UserHomeDir()
		rootDir = filepath.Join(home, ".neural-junkie", "lora-training")
	}
	if strings.TrimSpace(pythonPath) == "" {
		pythonPath = "python3"
	}
	m := &Manager{
		rootDir:  rootDir,
		jobsPath: filepath.Join(rootDir, "jobs.json"),
		script:   scriptPath,
		python:   pythonPath,
		msgs:     msgs,
		collab:   collab,
		jobs:     make(map[string]*Job),
	}
	mlx := filepath.Join(filepath.Dir(scriptPath), "lora_train_mlx.py")
	if _, err := os.Stat(mlx); err == nil {
		m.mlxScript = mlx
	}
	_ = m.loadJobs()
	return m
}

// SetRegistry attaches the adapter registry.
func (m *Manager) SetRegistry(reg *registry.Store) {
	m.mu.Lock()
	m.registry = reg
	m.mu.Unlock()
}

// SetOnComplete sets post-training hook.
func (m *Manager) SetOnComplete(fn OnCompleteFunc) {
	m.mu.Lock()
	m.onComplete = fn
	m.mu.Unlock()
}

func (m *Manager) loadJobs() error {
	raw, err := os.ReadFile(m.jobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var f jobsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	for i := range f.Jobs {
		j := f.Jobs[i]
		m.jobs[j.ID] = &j
	}
	return nil
}

func (m *Manager) persistJobsLocked() error {
	list := make([]Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		list = append(list, *j)
	}
	raw, err := json.MarshalIndent(jobsFile{Jobs: list}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.jobsPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(m.jobsPath, raw, 0644)
}

// Start enqueues and runs a training job asynchronously.
func (m *Manager) Start(ctx context.Context, req StartRequest) (*Job, error) {
	if strings.TrimSpace(req.SourceID) == "" {
		return nil, fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(req.BaseOllamaTag) == "" || strings.TrimSpace(req.OllamaTag) == "" {
		return nil, fmt.Errorf("base_ollama_tag and ollama_tag are required")
	}
	if err := hfhub.ValidateLoRATrainingBase(req.BaseOllamaTag); err != nil {
		return nil, err
	}
	m.mu.Lock()
	id := uuid.NewString()
	job := &Job{
		ID:            id,
		Status:        StatusQueued,
		Source:        req.Source,
		SourceID:      strings.TrimSpace(req.SourceID),
		AgentID:       strings.TrimSpace(req.AgentID),
		BaseOllamaTag: strings.TrimSpace(req.BaseOllamaTag),
		OllamaTag:     strings.TrimSpace(req.OllamaTag),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.jobs[id] = job
	_ = m.persistJobsLocked()

	busy := m.active != nil && m.active.Status != StatusDone && m.active.Status != StatusFailed && m.active.Status != StatusCancelled
	if busy {
		m.queue = append(m.queue, &queuedJob{id: id, req: req})
		job.QueuePosition = len(m.queue)
		m.mu.Unlock()
		return copyJob(job), nil
	}
	m.mu.Unlock()
	go m.startJob(id, req)
	return copyJob(job), nil
}

func (m *Manager) startJob(id string, req StartRequest) {
	jobCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	j := m.jobs[id]
	if j == nil {
		m.mu.Unlock()
		cancel()
		return
	}
	rj := &runningJob{Job: *j, cancel: cancel, req: req}
	m.active = rj
	m.mu.Unlock()
	m.run(jobCtx, rj, req)
	m.mu.Lock()
	m.active = nil
	if len(m.queue) > 0 {
		next := m.queue[0]
		m.queue = m.queue[1:]
		m.mu.Unlock()
		go m.startJob(next.id, next.req)
		return
	}
	m.mu.Unlock()
}

func (m *Manager) run(ctx context.Context, rj *runningJob, req StartRequest) {
	defer rj.cancel()
	m.setStatus(rj, StatusExporting, "")

	exportReq := m.buildExportRequest(req)
	var collab *collaboration.Collaboration
	if req.Source == lorexport.SourceCollaboration && m.collab != nil {
		snap, err := m.collab.GetCollaborationSnapshot(req.SourceID)
		if err != nil {
			m.fail(rj, "load collaboration: "+err.Error())
			return
		}
		collab = snap
	}
	datasetPath := filepath.Join(m.rootDir, rj.ID, "dataset.jsonl")
	n, err := lorexport.Export(exportReq, m.msgs, collab, datasetPath)
	if err != nil {
		m.fail(rj, err.Error())
		return
	}
	m.mu.Lock()
	rj.RowCount = n
	m.jobs[rj.ID].RowCount = n
	_ = m.persistJobsLocked()
	m.mu.Unlock()

	m.setStatus(rj, StatusTraining, "")
	outDir := filepath.Join(m.rootDir, rj.ID, "output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		m.fail(rj, err.Error())
		return
	}
	hp := req.HyperParams
	if hp.Rank <= 0 {
		hp.Rank = 16
	}
	if hp.Epochs <= 0 {
		hp.Epochs = 1
	}
	if hp.LearningRate <= 0 {
		hp.LearningRate = 2e-4
	}
	if hp.MaxSeqLen <= 0 {
		hp.MaxSeqLen = 2048
	}
	script, backend := m.resolveTrainer(req)
	args := []string{
		script,
		"--dataset", datasetPath,
		"--output-dir", outDir,
		"--base-model", mapBaseToHF(req.BaseOllamaTag),
		"--rank", fmt.Sprintf("%d", hp.Rank),
		"--epochs", fmt.Sprintf("%d", hp.Epochs),
		"--learning-rate", fmt.Sprintf("%g", hp.LearningRate),
		"--max-seq-len", fmt.Sprintf("%d", hp.MaxSeqLen),
		"--backend", backend,
	}
	if resume := m.resumeAdapterPath(req); resume != "" {
		args = append(args, "--resume-adapter", resume)
	}
	cmd := exec.CommandContext(ctx, m.python, args...)
	rj.cmd = cmd
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		m.fail(rj, "start trainer: "+err.Error())
		return
	}
	go m.pipeLines(stdout, rj)
	go m.pipeLines(stderr, rj)
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			m.setStatus(rj, StatusCancelled, "cancelled")
			return
		}
		m.fail(rj, "training failed: "+err.Error())
		return
	}

	adapterPath := filepath.Join(outDir, "adapter_model.safetensors")
	if _, err := os.Stat(adapterPath); err != nil {
		m.fail(rj, "adapter output missing after training")
		return
	}

	m.setStatus(rj, StatusComposing, "")
	if err := hfhub.ImportAdapterToOllama(ctx, rj.BaseOllamaTag, adapterPath, rj.OllamaTag); err != nil {
		m.fail(rj, "compose in Ollama: "+err.Error())
		return
	}

	m.mu.Lock()
	onComplete := m.onComplete
	m.mu.Unlock()
	if onComplete != nil {
		m.setStatus(rj, StatusEvaluating, "")
		if err := onComplete(ctx, &rj.Job, outDir, datasetPath); err != nil {
			m.appendLog(rj, "on_complete: "+err.Error())
		}
	}
	m.setStatus(rj, StatusDone, "")
}

func (m *Manager) buildExportRequest(req StartRequest) lorexport.Request {
	exp := lorexport.Request{
		Source:            req.Source,
		SourceID:          req.SourceID,
		ThreadID:          req.ThreadID,
		AgentName:         req.AgentName,
		ExtraRows:         req.LearningRows,
		RowIDs:            req.RowIDs,
		ApprovedTasksOnly: req.ApprovedTasksOnly,
	}
	if req.Incremental || req.SinceJobID != "" || req.PriorAdapterID != "" {
		m.mu.Lock()
		reg := m.registry
		m.mu.Unlock()
		if reg != nil {
			id := strings.TrimSpace(req.PriorAdapterID)
			if id == "" {
				id = strings.TrimSpace(req.SinceJobID)
			}
			if e, ok := reg.Get(id); ok {
				exp.SinceJobExportedAt = e.ExportedAt
			} else if e, ok := reg.ActiveForAgent(req.AgentID); ok && req.Incremental {
				exp.SinceJobExportedAt = e.ExportedAt
			}
		}
	}
	return exp
}

func (m *Manager) resumeAdapterPath(req StartRequest) string {
	m.mu.Lock()
	reg := m.registry
	m.mu.Unlock()
	if reg == nil {
		return ""
	}
	id := strings.TrimSpace(req.PriorAdapterID)
	if id == "" && req.Incremental {
		if e, ok := reg.ActiveForAgent(req.AgentID); ok {
			id = e.ID
		}
	}
	if id == "" {
		return ""
	}
	e, ok := reg.Get(id)
	if !ok || e.ArtifactDir == "" {
		return ""
	}
	p := filepath.Join(e.ArtifactDir, "adapter_model.safetensors")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func (m *Manager) resolveTrainer(req StartRequest) (script, backend string) {
	backend = strings.ToLower(strings.TrimSpace(req.Backend))
	if backend == "" || backend == "auto" {
		if m.mlxScript != "" && preferMLX() {
			return m.mlxScript, "mlx"
		}
		return m.script, "unsloth"
	}
	if backend == "mlx" && m.mlxScript != "" {
		return m.mlxScript, "mlx"
	}
	return m.script, "unsloth"
}

func mapBaseToHF(tag string) string {
	if hf := hfhub.MapLoRABaseToHF(tag); hf != "" {
		return hf
	}
	return strings.TrimSpace(tag)
}

func (m *Manager) pipeLines(r io.Reader, rj *runningJob) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		m.appendLog(rj, line)
		var prog map[string]interface{}
		if json.Unmarshal([]byte(line), &prog) == nil {
			if st, ok := prog["status"].(string); ok && st != "" {
				m.appendLog(rj, fmt.Sprintf("[progress] %s", st))
			}
		}
	}
}

func (m *Manager) appendLog(rj *runningJob, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rj.LogTail = append(rj.LogTail, line)
	if len(rj.LogTail) > 200 {
		rj.LogTail = rj.LogTail[len(rj.LogTail)-200:]
	}
	rj.UpdatedAt = time.Now()
	if j := m.jobs[rj.ID]; j != nil {
		j.LogTail = append([]string(nil), rj.LogTail...)
		j.UpdatedAt = rj.UpdatedAt
	}
}

func (m *Manager) setStatus(rj *runningJob, st Status, errMsg string) {
	m.mu.Lock()
	rj.Status = st
	rj.Error = errMsg
	rj.UpdatedAt = time.Now()
	if j := m.jobs[rj.ID]; j != nil {
		j.Status = st
		j.Error = errMsg
		j.UpdatedAt = rj.UpdatedAt
		j.RowCount = rj.RowCount
		j.AdapterID = rj.AdapterID
		j.EvalScore = rj.EvalScore
		_ = m.persistJobsLocked()
	}
	m.mu.Unlock()
}

func (m *Manager) fail(rj *runningJob, msg string) {
	m.setStatus(rj, StatusFailed, msg)
	m.appendLog(rj, msg)
}

// Get returns a job by id.
func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && m.active.ID == id {
		return copyJob(&m.active.Job), true
	}
	if j, ok := m.jobs[id]; ok {
		return copyJob(j), true
	}
	return nil, false
}

// List returns recent jobs newest first.
func (m *Manager) List(limit int) []*Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]*Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, copyJob(j))
	}
	// simple sort by CreatedAt desc
	for i := 0; i < len(out); i++ {
		for k := i + 1; k < len(out); k++ {
			if out[k].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[k] = out[k], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Active returns the current running job if any.
func (m *Manager) Active() (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return nil, false
	}
	return copyJob(&m.active.Job), true
}

// Cancel stops the running job.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && m.active.ID == id {
		if m.active.cmd != nil && m.active.cmd.Process != nil {
			_ = m.active.cmd.Process.Kill()
		}
		m.active.cancel()
		m.active.Status = StatusCancelled
		m.active.UpdatedAt = time.Now()
		if j := m.jobs[id]; j != nil {
			j.Status = StatusCancelled
			j.UpdatedAt = m.active.UpdatedAt
			_ = m.persistJobsLocked()
		}
		return nil
	}
	for i, q := range m.queue {
		if q.id == id {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			if j := m.jobs[id]; j != nil {
				j.Status = StatusCancelled
				j.UpdatedAt = time.Now()
				_ = m.persistJobsLocked()
			}
			return nil
		}
	}
	return fmt.Errorf("job not found or not active")
}

// Preview estimates row count for a source.
func (m *Manager) Preview(req lorexport.Request) (int, error) {
	var collab *collaboration.Collaboration
	if req.Source == lorexport.SourceCollaboration && m.collab != nil {
		snap, err := m.collab.GetCollaborationSnapshot(req.SourceID)
		if err != nil {
			return 0, err
		}
		collab = snap
	}
	return lorexport.PreviewRowCount(req, m.msgs, collab)
}

// PreviewDataset returns rows for curation.
func (m *Manager) PreviewDataset(req lorexport.Request) ([]lorexport.PreviewRow, error) {
	var collab *collaboration.Collaboration
	if req.Source == lorexport.SourceCollaboration && m.collab != nil {
		snap, err := m.collab.GetCollaborationSnapshot(req.SourceID)
		if err != nil {
			return nil, err
		}
		collab = snap
	}
	return lorexport.PreviewRows(req, m.msgs, collab)
}

func copyJob(j *Job) *Job {
	if j == nil {
		return nil
	}
	out := *j
	if len(j.LogTail) > 0 {
		out.LogTail = append([]string(nil), j.LogTail...)
	}
	return &out
}
