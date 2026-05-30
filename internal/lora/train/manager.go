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
)

// Status is a training job lifecycle state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusExporting Status = "exporting"
	StatusTraining  Status = "training"
	StatusComposing Status = "composing"
	StatusDone      Status = "done"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// HyperParams are optional LoRA training knobs.
type HyperParams struct {
	Rank       int     `json:"rank,omitempty"`
	Epochs     int     `json:"epochs,omitempty"`
	LearningRate float64 `json:"learning_rate,omitempty"`
	MaxSeqLen  int     `json:"max_seq_len,omitempty"`
}

// StartRequest starts a new training job.
type StartRequest struct {
	Source           lorexport.SourceKind `json:"source"`
	SourceID         string               `json:"source_id"`
	ThreadID         string               `json:"thread_id,omitempty"`
	AgentName        string               `json:"agent_name,omitempty"`
	AgentID          string               `json:"agent_id,omitempty"`
	IncludeLearnings bool                 `json:"include_learnings,omitempty"`
	LearningRows     []lorexport.Row      `json:"-"`
	BaseOllamaTag    string               `json:"base_ollama_tag"`
	OllamaTag        string               `json:"ollama_tag"`
	HyperParams      HyperParams          `json:"hyperparams,omitempty"`
}

// Job is the persisted view of one training run.
type Job struct {
	ID            string               `json:"id"`
	Status        Status               `json:"status"`
	Source        lorexport.SourceKind `json:"source"`
	SourceID      string               `json:"source_id"`
	BaseOllamaTag string               `json:"base_ollama_tag"`
	OllamaTag     string               `json:"ollama_tag"`
	RowCount      int                  `json:"row_count,omitempty"`
	LogTail       []string             `json:"log_tail,omitempty"`
	Error         string               `json:"error,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// HubMessages adapts hub message getters (see lorexport.MessageSource).

// CollabLookup loads collaboration snapshots.
type CollabLookup interface {
	GetCollaborationSnapshot(id string) (*collaboration.Collaboration, error)
}

// Manager runs at most one LoRA training job at a time.
type Manager struct {
	mu      sync.Mutex
	active  *runningJob
	rootDir string
	script  string
	python  string
	msgs    lorexport.MessageSource
	collab  CollabLookup
}

type runningJob struct {
	Job
	cancel context.CancelFunc
	cmd    *exec.Cmd
}

// NewManager creates a job manager. scriptPath points to scripts/lora_train.py.
func NewManager(rootDir, scriptPath, pythonPath string, msgs lorexport.MessageSource, collab CollabLookup) *Manager {
	if rootDir == "" {
		home, _ := os.UserHomeDir()
		rootDir = filepath.Join(home, ".neural-junkie", "lora-training")
	}
	if strings.TrimSpace(pythonPath) == "" {
		pythonPath = "python3"
	}
	return &Manager{rootDir: rootDir, script: scriptPath, python: pythonPath, msgs: msgs, collab: collab}
}

// Start enqueues and runs a training job asynchronously.
func (m *Manager) Start(ctx context.Context, req StartRequest) (*Job, error) {
	if strings.TrimSpace(req.SourceID) == "" {
		return nil, fmt.Errorf("source_id is required")
	}
	if strings.TrimSpace(req.BaseOllamaTag) == "" || strings.TrimSpace(req.OllamaTag) == "" {
		return nil, fmt.Errorf("base_ollama_tag and ollama_tag are required")
	}
	m.mu.Lock()
	if m.active != nil && m.active.Status != StatusDone && m.active.Status != StatusFailed && m.active.Status != StatusCancelled {
		m.mu.Unlock()
		return nil, fmt.Errorf("another training job is already running")
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	id := uuid.NewString()
	rj := &runningJob{
		Job: Job{
			ID:            id,
			Status:        StatusQueued,
			Source:        req.Source,
			SourceID:      strings.TrimSpace(req.SourceID),
			BaseOllamaTag: strings.TrimSpace(req.BaseOllamaTag),
			OllamaTag:     strings.TrimSpace(req.OllamaTag),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		cancel: cancel,
	}
	m.active = rj
	m.mu.Unlock()

	go m.run(jobCtx, rj, req)
	return copyJob(&rj.Job), nil
}

func (m *Manager) run(ctx context.Context, rj *runningJob, req StartRequest) {
	defer rj.cancel()
	m.setStatus(rj, StatusExporting, "")
	datasetPath := filepath.Join(m.rootDir, rj.ID, "dataset.jsonl")
	var collab *collaboration.Collaboration
	if req.Source == lorexport.SourceCollaboration && m.collab != nil {
		snap, err := m.collab.GetCollaborationSnapshot(req.SourceID)
		if err != nil {
			m.fail(rj, "load collaboration: "+err.Error())
			return
		}
		collab = snap
	}
	exportReq := lorexport.Request{
		Source:    req.Source,
		SourceID:  req.SourceID,
		ThreadID:  req.ThreadID,
		AgentName: req.AgentName,
		ExtraRows: req.LearningRows,
	}
	n, err := lorexport.Export(exportReq, m.msgs, collab, datasetPath)
	if err != nil {
		m.fail(rj, err.Error())
		return
	}
	m.mu.Lock()
	rj.RowCount = n
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
	args := []string{
		m.script,
		"--dataset", datasetPath,
		"--output-dir", outDir,
		"--base-model", mapBaseToHF(req.BaseOllamaTag),
		"--rank", fmt.Sprintf("%d", hp.Rank),
		"--epochs", fmt.Sprintf("%d", hp.Epochs),
		"--learning-rate", fmt.Sprintf("%g", hp.LearningRate),
		"--max-seq-len", fmt.Sprintf("%d", hp.MaxSeqLen),
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
	m.setStatus(rj, StatusDone, "")
}

func mapBaseToHF(tag string) string {
	switch strings.TrimSpace(tag) {
	case "qwen2.5-coder:14b":
		return "Qwen/Qwen2.5-Coder-14B-Instruct"
	case "llama3:8b":
		return "meta-llama/Meta-Llama-3-8B-Instruct"
	default:
		return tag
	}
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
}

func (m *Manager) setStatus(rj *runningJob, st Status, errMsg string) {
	m.mu.Lock()
	rj.Status = st
	rj.Error = errMsg
	rj.UpdatedAt = time.Now()
	m.mu.Unlock()
}

func (m *Manager) fail(rj *runningJob, msg string) {
	m.setStatus(rj, StatusFailed, msg)
	m.appendLog(rj, msg)
}

// Get returns the active or last job by id.
func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil && m.active.ID == id {
		return copyJob(&m.active.Job), true
	}
	return nil, false
}

// Active returns the current running job if any.
func (m *Manager) Active() (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return nil, false
	}
	if m.active.Status == StatusDone || m.active.Status == StatusFailed || m.active.Status == StatusCancelled {
		return copyJob(&m.active.Job), true
	}
	return copyJob(&m.active.Job), m.active.Status != StatusDone
}

// Cancel stops the running job.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil || m.active.ID != id {
		return fmt.Errorf("job not found or not active")
	}
	if m.active.cmd != nil && m.active.cmd.Process != nil {
		_ = m.active.cmd.Process.Kill()
	}
	m.active.cancel()
	m.active.Status = StatusCancelled
	m.active.UpdatedAt = time.Now()
	return nil
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
