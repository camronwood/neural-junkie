package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/camronwood/neural-junkie/internal/ollama"
	"github.com/google/uuid"
)

const (
	arenaBenchmarkStatusQueued    = "queued"
	arenaBenchmarkStatusRunning   = "running"
	arenaBenchmarkStatusCompleted = "completed"
	arenaBenchmarkStatusFailed    = "failed"
	arenaBenchmarkStatusCancelled = "cancelled"

	arenaBenchmarkModelPending   = "pending"
	arenaBenchmarkModelRunning   = "running"
	arenaBenchmarkModelCompleted = "completed"
	arenaBenchmarkModelFailed    = "failed"
	arenaBenchmarkModelSkipped   = "skipped"
)

// ArenaBenchmarkStartRequest starts a multi-model arena benchmark job.
type ArenaBenchmarkStartRequest struct {
	Models     []string `json:"models"`
	Suite      string   `json:"suite"`
	ProviderID string   `json:"provider_id"`
}

// ArenaBenchmarkScenarioResult is one challenge result for a model.
type ArenaBenchmarkScenarioResult struct {
	Scenario           string  `json:"scenario"`
	LogicCorrect       int     `json:"logic_correct"`
	LogicTotal         int     `json:"logic_total"`
	Connect4Completed  bool    `json:"connect4_completed,omitempty"`
	Wins               int     `json:"wins"`
	Losses             int     `json:"losses"`
	Draws              int     `json:"draws"`
	IllegalMoves       int     `json:"illegal_moves"`
	PassRate           float64 `json:"pass_rate"`
	CapabilityPassed   bool    `json:"capability_passed"`
	Error              string  `json:"error,omitempty"`
}

// ArenaBenchmarkModelResult aggregates suite results for one model.
type ArenaBenchmarkModelResult struct {
	Model              string                         `json:"model"`
	Status             string                         `json:"status"`
	LogicCorrect       int                            `json:"logic_correct"`
	LogicTotal         int                            `json:"logic_total"`
	Connect4Completed  bool                           `json:"connect4_completed"`
	Wins               int                            `json:"wins"`
	Losses             int                            `json:"losses"`
	Draws              int                            `json:"draws"`
	IllegalMoves       int                            `json:"illegal_moves"`
	PassRate           float64                        `json:"pass_rate"`
	CapabilityPassed   bool                           `json:"capability_passed"`
	Scenarios          []ArenaBenchmarkScenarioResult `json:"scenarios,omitempty"`
	Lines              []string                       `json:"lines,omitempty"`
	Error              string                         `json:"error,omitempty"`
}

// ArenaBenchmarkRank is a sorted scoreboard row.
type ArenaBenchmarkRank struct {
	Rank               int     `json:"rank"`
	Model              string  `json:"model"`
	PassRate           float64 `json:"pass_rate"`
	LogicCorrect       int     `json:"logic_correct"`
	LogicTotal         int     `json:"logic_total"`
	Connect4Completed  bool    `json:"connect4_completed"`
	CapabilityPassed   bool    `json:"capability_passed"`
	Error              string  `json:"error,omitempty"`
}

// ArenaBenchmarkJob is a persisted multi-model benchmark run.
type ArenaBenchmarkJob struct {
	ID                string                      `json:"id"`
	Status            string                      `json:"status"`
	Suite             string                      `json:"suite"`
	SuiteTitle        string                      `json:"suite_title,omitempty"`
	Challenges        []string                    `json:"challenges"`
	Models            []string                    `json:"models"`
	ProviderID        string                      `json:"provider_id,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	CurrentModel      string                      `json:"current_model,omitempty"`
	CurrentChallenge  string                      `json:"current_challenge,omitempty"`
	ModelIndex        int                         `json:"model_index"`
	ChallengeIndex    int                         `json:"challenge_index"`
	Results           []ArenaBenchmarkModelResult `json:"results"`
	Ranking           []ArenaBenchmarkRank        `json:"ranking,omitempty"`
	Error             string                      `json:"error,omitempty"`
}

type arenaBenchmarkRuntime struct {
	cancel context.CancelFunc
}

var (
	arenaBenchmarkMu    sync.Mutex
	arenaBenchmarkJobs  = map[string]*ArenaBenchmarkJob{}
	arenaBenchmarkRuns  = map[string]*arenaBenchmarkRuntime{}
	arenaBenchmarkDirFn = defaultArenaBenchmarkDir
)

func defaultArenaBenchmarkDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".neural-junkie", "arena", "benchmarks"), nil
}

func arenaBenchmarkPath(id string) (string, error) {
	dir, err := arenaBenchmarkDirFn()
	if err != nil {
		return "", err
	}
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)
	if safe == "" {
		return "", fmt.Errorf("invalid benchmark id")
	}
	return filepath.Join(dir, safe+".json"), nil
}

func (h *Hub) persistArenaBenchmarkJob(job *ArenaBenchmarkJob) error {
	if job == nil {
		return fmt.Errorf("nil job")
	}
	path, err := arenaBenchmarkPath(job.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func loadArenaBenchmarkJobFromDisk(id string) (*ArenaBenchmarkJob, error) {
	path, err := arenaBenchmarkPath(id)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var job ArenaBenchmarkJob
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func cloneArenaBenchmarkJob(job *ArenaBenchmarkJob) *ArenaBenchmarkJob {
	if job == nil {
		return nil
	}
	raw, err := json.Marshal(job)
	if err != nil {
		cp := *job
		return &cp
	}
	var out ArenaBenchmarkJob
	if err := json.Unmarshal(raw, &out); err != nil {
		cp := *job
		return &cp
	}
	return &out
}

func rankArenaBenchmarkResults(results []ArenaBenchmarkModelResult) []ArenaBenchmarkRank {
	type row struct {
		ArenaBenchmarkModelResult
	}
	rows := make([]row, 0, len(results))
	for _, r := range results {
		if r.Status == arenaBenchmarkModelPending || r.Status == arenaBenchmarkModelRunning {
			continue
		}
		rows = append(rows, row{r})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].PassRate != rows[j].PassRate {
			return rows[i].PassRate > rows[j].PassRate
		}
		if rows[i].LogicCorrect != rows[j].LogicCorrect {
			return rows[i].LogicCorrect > rows[j].LogicCorrect
		}
		return rows[i].Model < rows[j].Model
	})
	out := make([]ArenaBenchmarkRank, 0, len(rows))
	for i, r := range rows {
		out = append(out, ArenaBenchmarkRank{
			Rank:              i + 1,
			Model:             r.Model,
			PassRate:          r.PassRate,
			LogicCorrect:      r.LogicCorrect,
			LogicTotal:        r.LogicTotal,
			Connect4Completed: r.Connect4Completed,
			CapabilityPassed:  r.CapabilityPassed,
			Error:             r.Error,
		})
	}
	return out
}

func aggregateArenaModelPassRate(r *ArenaBenchmarkModelResult, challenges []string) {
	if r == nil {
		return
	}
	parts := make([]float64, 0, 2)
	hasLogic := false
	hasC4 := false
	for _, c := range challenges {
		switch c {
		case "logic-set":
			hasLogic = true
		case "connect4-smoke":
			hasC4 = true
		}
	}
	if hasLogic && r.LogicTotal > 0 {
		parts = append(parts, float64(r.LogicCorrect)/float64(r.LogicTotal))
	} else if hasLogic {
		parts = append(parts, 0)
	}
	if hasC4 {
		if r.Connect4Completed {
			parts = append(parts, 1)
		} else {
			parts = append(parts, 0)
		}
	}
	if len(parts) == 0 {
		r.PassRate = 0
		return
	}
	sum := 0.0
	for _, p := range parts {
		sum += p
	}
	r.PassRate = float64(int(sum/float64(len(parts))*10000+0.5)) / 10000
	minLogic := arenaLogicPassRateMin()
	logicOK := !hasLogic || (r.LogicTotal > 0 && float64(r.LogicCorrect)/float64(r.LogicTotal) >= minLogic)
	c4OK := !hasC4 || r.Connect4Completed
	r.CapabilityPassed = logicOK && c4OK && r.Error == ""
}

// ListInstalledOllamaModels returns local Ollama tags via the hub's configured endpoint.
func (h *Hub) ListInstalledOllamaModels(ctx context.Context) ([]string, error) {
	endpoint := "http://localhost:11434"
	if h != nil && h.commandHandler != nil && h.commandHandler.appConfig != nil {
		if ep := strings.TrimSpace(h.commandHandler.appConfig.FirstOllamaEndpoint()); ep != "" {
			endpoint = ep
		}
	}
	mgr := ollama.NewManager(endpoint)
	return mgr.ListModels(ctx)
}

// StartArenaBenchmark validates models/suite and launches a background job.
func (h *Hub) StartArenaBenchmark(ctx context.Context, req ArenaBenchmarkStartRequest) (*ArenaBenchmarkJob, error) {
	if h == nil || !h.ArenaAvailable() {
		return nil, fmt.Errorf("model arena disabled (install and enable the Model Arena pack)")
	}
	suite, err := resolveArenaBenchmarkSuite(req.Suite)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(req.Models))
	seen := map[string]struct{}{}
	for _, m := range req.Models {
		m = strings.TrimSpace(m)
		if m == "" || strings.EqualFold(m, arenaHumanPlayer) {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		models = append(models, m)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("select at least one installed model")
	}

	installed, err := h.ListInstalledOllamaModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installed models: %w", err)
	}
	for _, m := range models {
		if !ollama.TagInstalled(installed, m) {
			return nil, fmt.Errorf("model %q is not installed in Ollama", m)
		}
	}

	// Probe sidecar early.
	if _, err := h.ArenaSidecarGet(ctx, "/api/arena/challenges"); err != nil {
		return nil, fmt.Errorf("arena sidecar unavailable: %w", err)
	}

	now := time.Now().UTC()
	results := make([]ArenaBenchmarkModelResult, 0, len(models))
	for _, m := range models {
		results = append(results, ArenaBenchmarkModelResult{
			Model:  m,
			Status: arenaBenchmarkModelPending,
		})
	}
	job := &ArenaBenchmarkJob{
		ID:         uuid.NewString(),
		Status:     arenaBenchmarkStatusQueued,
		Suite:      suite.ID,
		SuiteTitle: suite.Title,
		Challenges: append([]string{}, suite.Challenges...),
		Models:     models,
		ProviderID: strings.TrimSpace(req.ProviderID),
		CreatedAt:  now,
		UpdatedAt:  now,
		Results:    results,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	arenaBenchmarkMu.Lock()
	arenaBenchmarkJobs[job.ID] = job
	arenaBenchmarkRuns[job.ID] = &arenaBenchmarkRuntime{cancel: cancel}
	arenaBenchmarkMu.Unlock()
	_ = h.persistArenaBenchmarkJob(job)

	go h.runArenaBenchmarkJob(runCtx, job.ID)
	return cloneArenaBenchmarkJob(job), nil
}

func arenaBenchmarkStatusTerminal(status string) bool {
	switch status {
	case arenaBenchmarkStatusCompleted, arenaBenchmarkStatusFailed, arenaBenchmarkStatusCancelled:
		return true
	default:
		return false
	}
}

// healOrphanedArenaBenchmarkJob marks disk-only queued/running jobs as failed after a hub restart.
func (h *Hub) healOrphanedArenaBenchmarkJob(job *ArenaBenchmarkJob, reason string) *ArenaBenchmarkJob {
	if job == nil || arenaBenchmarkStatusTerminal(job.Status) {
		return job
	}
	job.Status = arenaBenchmarkStatusFailed
	if strings.TrimSpace(job.Error) == "" {
		job.Error = reason
	}
	job.CurrentModel = ""
	job.CurrentChallenge = ""
	job.UpdatedAt = time.Now().UTC()
	for i := range job.Results {
		if job.Results[i].Status == arenaBenchmarkModelPending || job.Results[i].Status == arenaBenchmarkModelRunning {
			job.Results[i].Status = arenaBenchmarkModelSkipped
		}
	}
	job.Ranking = rankArenaBenchmarkResults(job.Results)
	_ = h.persistArenaBenchmarkJob(job)
	return job
}

// GetArenaBenchmark returns a job by id (memory first, then disk).
func (h *Hub) GetArenaBenchmark(id string) (*ArenaBenchmarkJob, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("benchmark id required")
	}
	arenaBenchmarkMu.Lock()
	job := arenaBenchmarkJobs[id]
	if job != nil {
		out := cloneArenaBenchmarkJob(job)
		arenaBenchmarkMu.Unlock()
		return out, nil
	}
	arenaBenchmarkMu.Unlock()
	job, err := loadArenaBenchmarkJobFromDisk(id)
	if err != nil {
		return nil, fmt.Errorf("benchmark not found")
	}
	return h.healOrphanedArenaBenchmarkJob(job, "benchmark interrupted by hub restart"), nil
}

// CancelArenaBenchmark requests cancellation between challenges.
func (h *Hub) CancelArenaBenchmark(id string) (*ArenaBenchmarkJob, error) {
	id = strings.TrimSpace(id)
	arenaBenchmarkMu.Lock()
	job := arenaBenchmarkJobs[id]
	rt := arenaBenchmarkRuns[id]
	if job == nil {
		arenaBenchmarkMu.Unlock()
		disk, err := loadArenaBenchmarkJobFromDisk(id)
		if err != nil {
			return nil, fmt.Errorf("benchmark not found")
		}
		if arenaBenchmarkStatusTerminal(disk.Status) {
			return disk, nil
		}
		// Orphaned after restart: mark cancelled so the UI can recover.
		disk.Status = arenaBenchmarkStatusCancelled
		disk.Error = "benchmark was not running in this hub process"
		disk.CurrentModel = ""
		disk.CurrentChallenge = ""
		disk.UpdatedAt = time.Now().UTC()
		for i := range disk.Results {
			if disk.Results[i].Status == arenaBenchmarkModelPending || disk.Results[i].Status == arenaBenchmarkModelRunning {
				disk.Results[i].Status = arenaBenchmarkModelSkipped
			}
		}
		disk.Ranking = rankArenaBenchmarkResults(disk.Results)
		_ = h.persistArenaBenchmarkJob(disk)
		return disk, nil
	}
	if arenaBenchmarkStatusTerminal(job.Status) {
		out := cloneArenaBenchmarkJob(job)
		arenaBenchmarkMu.Unlock()
		return out, nil
	}
	if rt != nil && rt.cancel != nil {
		rt.cancel()
	}
	out := cloneArenaBenchmarkJob(job)
	arenaBenchmarkMu.Unlock()
	return out, nil
}

// ListArenaBenchmarkHistory returns recent persisted runs (newest first).
func (h *Hub) ListArenaBenchmarkHistory(limit int) ([]*ArenaBenchmarkJob, error) {
	if limit <= 0 {
		limit = 20
	}
	dir, err := arenaBenchmarkDirFn()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ArenaBenchmarkJob{}, nil
		}
		return nil, err
	}
	type item struct {
		job  *ArenaBenchmarkJob
		mod  time.Time
	}
	items := make([]item, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		job, err := loadArenaBenchmarkJobFromDisk(id)
		if err != nil {
			continue
		}
		// History should not advertise stale queued/running jobs from a prior hub process.
		arenaBenchmarkMu.Lock()
		_, live := arenaBenchmarkJobs[id]
		arenaBenchmarkMu.Unlock()
		if !live {
			job = h.healOrphanedArenaBenchmarkJob(job, "benchmark interrupted by hub restart")
		}
		info, _ := e.Info()
		mod := job.UpdatedAt
		if info != nil {
			mod = info.ModTime()
		}
		items = append(items, item{job: job, mod: mod})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].mod.After(items[j].mod)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]*ArenaBenchmarkJob, 0, len(items))
	for _, it := range items {
		out = append(out, it.job)
	}
	return out, nil
}

func (h *Hub) updateArenaBenchmarkJob(id string, mutate func(*ArenaBenchmarkJob)) *ArenaBenchmarkJob {
	arenaBenchmarkMu.Lock()
	defer arenaBenchmarkMu.Unlock()
	job := arenaBenchmarkJobs[id]
	if job == nil {
		return nil
	}
	mutate(job)
	job.UpdatedAt = time.Now().UTC()
	job.Ranking = rankArenaBenchmarkResults(job.Results)
	_ = h.persistArenaBenchmarkJob(job)
	return cloneArenaBenchmarkJob(job)
}

func (h *Hub) runArenaBenchmarkJob(ctx context.Context, id string) {
	defer func() {
		arenaBenchmarkMu.Lock()
		delete(arenaBenchmarkRuns, id)
		arenaBenchmarkMu.Unlock()
	}()

	h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
		job.Status = arenaBenchmarkStatusRunning
	})

	arenaBenchmarkMu.Lock()
	jobSnap := cloneArenaBenchmarkJob(arenaBenchmarkJobs[id])
	arenaBenchmarkMu.Unlock()
	if jobSnap == nil {
		return
	}

	for mi, model := range jobSnap.Models {
		if ctx.Err() != nil {
			h.markArenaBenchmarkCancelled(id)
			return
		}
		h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
			job.ModelIndex = mi
			job.CurrentModel = model
			job.CurrentChallenge = ""
			if mi < len(job.Results) {
				job.Results[mi].Status = arenaBenchmarkModelRunning
			}
		})

		modelResult := ArenaBenchmarkModelResult{
			Model:     model,
			Status:    arenaBenchmarkModelRunning,
			Scenarios: nil,
			Lines:     nil,
		}
		cancelled := false
		for ci, challenge := range jobSnap.Challenges {
			if ctx.Err() != nil {
				cancelled = true
				break
			}
			h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
				job.ChallengeIndex = ci
				job.CurrentChallenge = challenge
			})
			scen, lines, err := h.runArenaBenchmarkChallenge(ctx, challenge, model, jobSnap.ProviderID)
			if err != nil {
				if ctx.Err() != nil {
					cancelled = true
					break
				}
				scen.Error = err.Error()
				modelResult.Error = err.Error()
				modelResult.Lines = append(modelResult.Lines, lines...)
				modelResult.Lines = append(modelResult.Lines, "FAIL: "+err.Error())
				modelResult.Scenarios = append(modelResult.Scenarios, scen)
				// Continue other challenges for this model when possible.
				continue
			}
			modelResult.LogicCorrect += scen.LogicCorrect
			modelResult.LogicTotal += scen.LogicTotal
			modelResult.Wins += scen.Wins
			modelResult.Losses += scen.Losses
			modelResult.Draws += scen.Draws
			modelResult.IllegalMoves += scen.IllegalMoves
			if scen.Connect4Completed {
				modelResult.Connect4Completed = true
			}
			modelResult.Scenarios = append(modelResult.Scenarios, scen)
			modelResult.Lines = append(modelResult.Lines, lines...)
		}
		if cancelled {
			modelResult.Status = arenaBenchmarkModelSkipped
			aggregateArenaModelPassRate(&modelResult, jobSnap.Challenges)
			h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
				if mi < len(job.Results) {
					job.Results[mi] = modelResult
				}
			})
			h.markArenaBenchmarkCancelled(id)
			return
		}
		if modelResult.Error != "" && len(modelResult.Scenarios) == 0 {
			modelResult.Status = arenaBenchmarkModelFailed
		} else {
			modelResult.Status = arenaBenchmarkModelCompleted
		}
		aggregateArenaModelPassRate(&modelResult, jobSnap.Challenges)
		h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
			if mi < len(job.Results) {
				job.Results[mi] = modelResult
			}
			job.CurrentChallenge = ""
		})
	}

	h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
		job.Status = arenaBenchmarkStatusCompleted
		job.CurrentModel = ""
		job.CurrentChallenge = ""
		job.Ranking = rankArenaBenchmarkResults(job.Results)
	})
}

func (h *Hub) markArenaBenchmarkCancelled(id string) {
	h.updateArenaBenchmarkJob(id, func(job *ArenaBenchmarkJob) {
		job.Status = arenaBenchmarkStatusCancelled
		job.CurrentChallenge = ""
		for i := range job.Results {
			if job.Results[i].Status == arenaBenchmarkModelPending || job.Results[i].Status == arenaBenchmarkModelRunning {
				job.Results[i].Status = arenaBenchmarkModelSkipped
			}
		}
		job.Ranking = rankArenaBenchmarkResults(job.Results)
	})
}

func (h *Hub) runArenaBenchmarkChallenge(ctx context.Context, challenge, model, providerID string) (ArenaBenchmarkScenarioResult, []string, error) {
	switch challenge {
	case "logic-set":
		return h.runArenaBenchmarkLogicSet(ctx, model, providerID)
	case "connect4-smoke":
		return h.runArenaBenchmarkConnect4(ctx, model, providerID)
	default:
		return ArenaBenchmarkScenarioResult{Scenario: challenge}, nil, fmt.Errorf("unknown challenge %q", challenge)
	}
}

func arenaSessionStatus(session map[string]any) string {
	if session == nil {
		return ""
	}
	if s, _ := session["status"].(string); strings.TrimSpace(s) != "" {
		return strings.ToLower(strings.TrimSpace(s))
	}
	st := mapStringAny(session["state"])
	if st == nil {
		return ""
	}
	s, _ := st["status"].(string)
	return strings.ToLower(strings.TrimSpace(s))
}

func arenaSessionResult(session map[string]any) string {
	if session == nil {
		return ""
	}
	if s, _ := session["result"].(string); strings.TrimSpace(s) != "" {
		return strings.ToLower(strings.TrimSpace(s))
	}
	st := mapStringAny(session["state"])
	if st == nil {
		return ""
	}
	s, _ := st["result"].(string)
	return strings.ToLower(strings.TrimSpace(s))
}

func (h *Hub) runArenaBenchmarkLogicSet(ctx context.Context, model, providerID string) (ArenaBenchmarkScenarioResult, []string, error) {
	out := ArenaBenchmarkScenarioResult{Scenario: "logic-set"}
	lines := []string{}
	challenges, err := h.ArenaSidecarGet(ctx, "/api/arena/challenges")
	if err != nil {
		return out, lines, err
	}
	puzzlesAny, _ := challenges["puzzles"].([]any)
	puzzles := make([]map[string]any, 0, len(puzzlesAny))
	for _, p := range puzzlesAny {
		if m, ok := p.(map[string]any); ok {
			puzzles = append(puzzles, m)
		}
	}
	if len(puzzles) == 0 {
		// Match CLI fallback: run a few anonymous sessions.
		puzzles = []map[string]any{{}, {}, {}}
	}
	correct := 0
	total := 0
	maxSteps := arenaLogicMaxSteps()
	for idx, puzzle := range puzzles {
		if ctx.Err() != nil {
			return out, lines, ctx.Err()
		}
		puzzleID, _ := puzzle["id"].(string)
		puzzleID = strings.TrimSpace(puzzleID)
		label := puzzleID
		if label == "" {
			label = fmt.Sprintf("logic-%d", idx+1)
		}
		body := map[string]any{
			"challenge": "logic",
			"white":     "human",
			"black":     model,
		}
		if puzzleID != "" {
			body["puzzle_id"] = puzzleID
		}
		session, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions", body)
		if err != nil {
			lines = append(lines, "FAIL: "+label+" create: "+err.Error())
			total++
			continue
		}
		sessionID, _ := session["id"].(string)
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			lines = append(lines, "FAIL: "+label+" missing session id")
			total++
			continue
		}
		final, err := h.ArenaRunMatchAuto(ctx, ArenaMatchRunRequest{
			SessionID:  sessionID,
			ProviderID: providerID,
			Model:      model,
			MaxSteps:   maxSteps,
		})
		if err != nil {
			if ctx.Err() != nil {
				return out, lines, ctx.Err()
			}
			// Re-fetch in case answer was recorded before error.
			if got, getErr := h.ArenaSidecarGet(ctx, "/api/arena/sessions/"+sessionID); getErr == nil {
				final = got
			} else {
				lines = append(lines, "FAIL: "+label+" "+err.Error())
				total++
				continue
			}
		}
		status := arenaSessionStatus(final)
		total++
		if status == "correct" {
			correct++
			lines = append(lines, "PASS: "+label+" correct")
		} else {
			lines = append(lines, fmt.Sprintf("FAIL: %s status=%s", label, statusOrUnknown(status)))
		}
	}
	out.LogicCorrect = correct
	out.LogicTotal = total
	rate := 0.0
	if total > 0 {
		rate = float64(correct) / float64(total)
	}
	out.PassRate = float64(int(rate*10000+0.5)) / 10000
	out.CapabilityPassed = total > 0 && rate >= arenaLogicPassRateMin()
	return out, lines, nil
}

func (h *Hub) runArenaBenchmarkConnect4(ctx context.Context, model, providerID string) (ArenaBenchmarkScenarioResult, []string, error) {
	out := ArenaBenchmarkScenarioResult{Scenario: "connect4-smoke"}
	lines := []string{}
	session, err := h.ArenaSidecarPost(ctx, "/api/arena/sessions", map[string]any{
		"challenge": "connect4",
		"white":     model,
		"black":     model,
	})
	if err != nil {
		return out, lines, err
	}
	sessionID, _ := session["id"].(string)
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		lines = append(lines, "FAIL: connect4-smoke missing session id")
		return out, lines, fmt.Errorf("missing session id")
	}
	final, err := h.ArenaRunMatchAuto(ctx, ArenaMatchRunRequest{
		SessionID:  sessionID,
		ProviderID: providerID,
		Model:      model,
		MaxSteps:   arenaConnect4MaxSteps(),
	})
	if err != nil && ctx.Err() == nil {
		if got, getErr := h.ArenaSidecarGet(ctx, "/api/arena/sessions/"+sessionID); getErr == nil {
			final = got
		} else {
			return out, lines, err
		}
	}
	if ctx.Err() != nil {
		return out, lines, ctx.Err()
	}
	if got, getErr := h.ArenaSidecarGet(ctx, "/api/arena/sessions/"+sessionID); getErr == nil {
		final = got
	}
	status := arenaSessionStatus(final)
	result := arenaSessionResult(final)
	completed := status == "finished" || status == "draw" || result == "draw"
	wins, losses, draws := 0, 0, 0
	if status == "draw" || result == "draw" {
		draws = 1
	} else if completed {
		if result == "red" || result == "yellow" || result == "white" || result == "black" {
			wins = 1
		} else {
			draws = 1
		}
	}
	out.Connect4Completed = completed
	out.Wins = wins
	out.Losses = losses
	out.Draws = draws
	out.PassRate = 0
	if completed {
		out.PassRate = 1
		lines = append(lines, fmt.Sprintf("PASS: connect4-smoke status=%s result=%s", statusOrUnknown(status), statusOrUnknown(result)))
	} else {
		lines = append(lines, fmt.Sprintf("FAIL: connect4-smoke status=%s result=%s", statusOrUnknown(status), statusOrUnknown(result)))
	}
	out.CapabilityPassed = completed
	return out, lines, nil
}

func statusOrUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}
